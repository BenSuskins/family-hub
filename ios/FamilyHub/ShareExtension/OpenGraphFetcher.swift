import Foundation

struct OGMetadata {
    var title: String?
    var description: String?
    var imageURL: String?
    var ingredients: [String]?
    var steps: [String]?
    var prepTime: String?
    var cookTime: String?
    var servings: Int?
}

enum OpenGraphFetcher {
    static func fetch(url: URL) async -> OGMetadata {
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 10
        let session = URLSession(configuration: config)

        guard let (data, _) = try? await session.data(from: url),
              let html = String(data: data, encoding: .utf8) ?? String(data: data, encoding: .isoLatin1)
        else { return OGMetadata() }

        var metadata = OGMetadata(
            title: extractOGTitle(from: html),
            description: extractMeta(property: "og:description", name: "description", from: html),
            imageURL: extractMeta(property: "og:image", name: nil, from: html)
        )

        if let recipe = extractRecipeJSONLD(from: html) {
            metadata.ingredients = recipe.ingredients
            metadata.steps = recipe.steps
            metadata.prepTime = recipe.prepTime
            metadata.cookTime = recipe.cookTime
            metadata.servings = recipe.servings
            if metadata.title == nil || metadata.title?.isEmpty == true,
               let recipeTitle = recipe.title {
                metadata.title = recipeTitle
            }
            if metadata.imageURL == nil, let recipeImage = recipe.imageURL {
                metadata.imageURL = recipeImage
            }
        }

        return metadata
    }

    // Extracts og:title, falling back to <title>
    private static func extractOGTitle(from html: String) -> String? {
        if let value = extractMeta(property: "og:title", name: "title", from: html) {
            return value
        }
        // <title>...</title> fallback
        if let range = html.range(of: "<title", options: .caseInsensitive),
           let closeTag = html.range(of: ">", range: range.upperBound..<html.endIndex),
           let endTag = html.range(of: "</title>", options: .caseInsensitive, range: closeTag.upperBound..<html.endIndex) {
            let title = String(html[closeTag.upperBound..<endTag.lowerBound])
                .trimmingCharacters(in: .whitespacesAndNewlines)
            return title.isEmpty ? nil : htmlDecode(title)
        }
        return nil
    }

    /// Search for <meta property="{property}" content="..."> or <meta name="{name}" content="...">
    private static func extractMeta(property: String, name: String?, from html: String) -> String? {
        let lowered = html.lowercased()

        var patterns = [
            "property=\"\(property)\"",
            "property='\(property)'"
        ]
        if let name {
            patterns.append("name=\"\(name)\"")
            patterns.append("name='\(name)'")
        }

        for pattern in patterns {
            guard let tagStart = findMetaTagStart(containing: pattern, in: lowered, fullHTML: html) else { continue }
            if let content = extractContent(fromMetaTag: tagStart) {
                return htmlDecode(content)
            }
        }
        return nil
    }

    /// Find the full <meta ...> tag that contains the given pattern
    private static func findMetaTagStart(containing pattern: String, in lowered: String, fullHTML: String) -> String? {
        var searchRange = lowered.startIndex..<lowered.endIndex
        while let patternRange = lowered.range(of: pattern, range: searchRange) {
            // Walk back to find the opening <
            var start = patternRange.lowerBound
            while start > lowered.startIndex {
                let prev = lowered.index(before: start)
                if lowered[prev] == "<" { start = prev; break }
                start = prev
            }
            // Walk forward to find the closing >
            if let endRange = lowered.range(of: ">", range: patternRange.upperBound..<lowered.endIndex) {
                let tag = String(fullHTML[start..<endRange.upperBound])
                return tag
            }
            searchRange = patternRange.upperBound..<lowered.endIndex
        }
        return nil
    }

    private static func extractContent(fromMetaTag tag: String) -> String? {
        let lowTag = tag.lowercased()
        for prefix in ["content=\"", "content='"] {
            guard let start = lowTag.range(of: prefix) else { continue }
            let quote: Character = prefix.last == "\"" ? "\"" : "'"
            let valueStart = tag.index(tag.startIndex, offsetBy: lowTag.distance(from: lowTag.startIndex, to: start.upperBound))
            if let end = tag[valueStart...].firstIndex(of: quote) {
                return String(tag[valueStart..<end])
            }
        }
        return nil
    }

    /// Downloads an image URL and returns a base64 data URI, or nil on failure.
    static func fetchImageAsDataURI(from urlString: String) async -> String? {
        guard let url = URL(string: urlString) else { return nil }
        var config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 15
        let session = URLSession(configuration: config)

        guard let (data, response) = try? await session.data(from: url),
              let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode),
              !data.isEmpty else { return nil }

        let mimeType = httpResponse.mimeType ?? "image/jpeg"
        let base64 = data.base64EncodedString()
        return "data:\(mimeType);base64,\(base64)"
    }

    // MARK: - JSON-LD Recipe Extraction

    private struct RecipeData {
        var title: String?
        var imageURL: String?
        var ingredients: [String]?
        var steps: [String]?
        var prepTime: String?
        var cookTime: String?
        var servings: Int?
    }

    private static func extractRecipeJSONLD(from html: String) -> RecipeData? {
        let scriptBlocks = extractJSONLDBlocks(from: html)
        for block in scriptBlocks {
            if let recipe = findRecipeObject(in: block) {
                return parseRecipeObject(recipe)
            }
        }
        return nil
    }

    private static func extractJSONLDBlocks(from html: String) -> [Any] {
        var blocks: [Any] = []
        let lowered = html.lowercased()
        let scriptTag = "<script type=\"application/ld+json\">"
        let scriptTagAlt = "<script type='application/ld+json'>"
        let closeTag = "</script>"

        for tag in [scriptTag, scriptTagAlt] {
            var searchStart = lowered.startIndex
            while let openRange = lowered.range(of: tag, range: searchStart..<lowered.endIndex) {
                guard let closeRange = lowered.range(of: closeTag, range: openRange.upperBound..<lowered.endIndex) else { break }
                let jsonString = String(html[openRange.upperBound..<closeRange.lowerBound])
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                if let data = jsonString.data(using: .utf8),
                   let parsed = try? JSONSerialization.jsonObject(with: data) {
                    blocks.append(parsed)
                }
                searchStart = closeRange.upperBound
            }
        }
        return blocks
    }

    private static func findRecipeObject(in json: Any) -> [String: Any]? {
        if let dict = json as? [String: Any] {
            if isRecipeType(dict) { return dict }
            if let graph = dict["@graph"] as? [[String: Any]] {
                return graph.first(where: isRecipeType)
            }
        }
        if let array = json as? [Any] {
            for item in array {
                if let dict = item as? [String: Any], isRecipeType(dict) { return dict }
            }
        }
        return nil
    }

    private static func isRecipeType(_ dict: [String: Any]) -> Bool {
        if let type = dict["@type"] as? String {
            return type == "Recipe"
        }
        if let types = dict["@type"] as? [String] {
            return types.contains("Recipe")
        }
        return false
    }

    private static func parseRecipeObject(_ recipe: [String: Any]) -> RecipeData {
        RecipeData(
            title: recipe["name"] as? String,
            imageURL: extractImageURL(from: recipe),
            ingredients: extractIngredients(from: recipe),
            steps: extractSteps(from: recipe),
            prepTime: (recipe["prepTime"] as? String).flatMap(formatISO8601Duration),
            cookTime: (recipe["cookTime"] as? String).flatMap(formatISO8601Duration),
            servings: extractServings(from: recipe)
        )
    }

    private static func extractImageURL(from recipe: [String: Any]) -> String? {
        if let url = recipe["image"] as? String { return url }
        if let images = recipe["image"] as? [String] { return images.first }
        if let imageObj = recipe["image"] as? [String: Any] { return imageObj["url"] as? String }
        if let imageArray = recipe["image"] as? [[String: Any]] {
            return imageArray.first?["url"] as? String
        }
        return nil
    }

    private static func extractIngredients(from recipe: [String: Any]) -> [String]? {
        guard let items = recipe["recipeIngredient"] as? [String], !items.isEmpty else { return nil }
        return items.map { htmlDecode($0).trimmingCharacters(in: .whitespacesAndNewlines) }
    }

    private static func extractSteps(from recipe: [String: Any]) -> [String]? {
        guard let instructions = recipe["recipeInstructions"] else { return nil }

        var result: [String] = []

        if let text = instructions as? String {
            result = text.components(separatedBy: "\n")
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
        } else if let items = instructions as? [Any] {
            for item in items {
                if let step = item as? String {
                    let trimmed = step.trimmingCharacters(in: .whitespacesAndNewlines)
                    if !trimmed.isEmpty { result.append(trimmed) }
                } else if let stepObj = item as? [String: Any] {
                    if let type = stepObj["@type"] as? String, type == "HowToSection" {
                        if let sectionItems = stepObj["itemListElement"] as? [[String: Any]] {
                            for sectionStep in sectionItems {
                                if let text = sectionStep["text"] as? String {
                                    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
                                    if !trimmed.isEmpty { result.append(trimmed) }
                                }
                            }
                        }
                    } else if let text = stepObj["text"] as? String {
                        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
                        if !trimmed.isEmpty { result.append(trimmed) }
                    }
                }
            }
        }

        return result.isEmpty ? nil : result.map { htmlDecode($0) }
    }

    private static func extractServings(from recipe: [String: Any]) -> Int? {
        if let servings = recipe["recipeYield"] as? Int { return servings }
        if let text = recipe["recipeYield"] as? String {
            let digits = text.components(separatedBy: CharacterSet.decimalDigits.inverted).joined()
            return Int(digits)
        }
        if let array = recipe["recipeYield"] as? [Any], let first = array.first {
            if let number = first as? Int { return number }
            if let text = first as? String {
                let digits = text.components(separatedBy: CharacterSet.decimalDigits.inverted).joined()
                return Int(digits)
            }
        }
        return nil
    }

    static func formatISO8601Duration(_ iso: String) -> String? {
        let upper = iso.uppercased()
        guard upper.hasPrefix("PT") || upper.hasPrefix("P") else { return nil }

        var remaining = upper.dropFirst(upper.hasPrefix("PT") ? 2 : 1)
        // Skip date parts until we hit T
        if !iso.uppercased().hasPrefix("PT"), let tIndex = remaining.firstIndex(of: "T") {
            remaining = remaining[remaining.index(after: tIndex)...]
        }

        var hours = 0
        var minutes = 0

        if let hIndex = remaining.firstIndex(of: "H") {
            hours = Int(remaining[remaining.startIndex..<hIndex]) ?? 0
            remaining = remaining[remaining.index(after: hIndex)...]
        }
        if let mIndex = remaining.firstIndex(of: "M") {
            minutes = Int(remaining[remaining.startIndex..<mIndex]) ?? 0
        }

        if hours == 0 && minutes == 0 { return nil }

        var parts: [String] = []
        if hours > 0 { parts.append("\(hours) hr\(hours > 1 ? "s" : "")") }
        if minutes > 0 { parts.append("\(minutes) min\(minutes > 1 ? "s" : "")") }
        return parts.joined(separator: " ")
    }

    // MARK: - HTML Decoding

    private static func htmlDecode(_ string: String) -> String {
        var result = string
        let entities: [(String, String)] = [
            ("&amp;", "&"), ("&lt;", "<"), ("&gt;", ">"),
            ("&quot;", "\""), ("&#39;", "'"), ("&apos;", "'"),
            ("&nbsp;", " ")
        ]
        for (entity, char) in entities {
            result = result.replacingOccurrences(of: entity, with: char)
        }
        // Numeric entities &#NNN;
        var out = ""
        var i = result.startIndex
        while i < result.endIndex {
            if result[i] == "&", let hash = result.index(i, offsetBy: 1, limitedBy: result.endIndex), hash < result.endIndex, result[hash] == "#" {
                if let semi = result[hash...].firstIndex(of: ";") {
                    let numStr = String(result[result.index(after: hash)..<semi])
                    if let code = Int(numStr), let scalar = Unicode.Scalar(code) {
                        out.append(Character(scalar))
                        i = result.index(after: semi)
                        continue
                    }
                }
            }
            out.append(result[i])
            i = result.index(after: i)
        }
        return out
    }
}
