import Foundation

struct OGMetadata {
    var title: String?
    var description: String?
}

enum OpenGraphFetcher {
    static func fetch(url: URL) async -> OGMetadata {
        var config = URLSessionConfiguration.ephemeral
        config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 10
        let session = URLSession(configuration: config)

        guard let (data, _) = try? await session.data(from: url),
              let html = String(data: data, encoding: .utf8) ?? String(data: data, encoding: .isoLatin1)
        else { return OGMetadata() }

        return OGMetadata(
            title: extractOGTitle(from: html),
            description: extractMeta(property: "og:description", name: "description", from: html)
        )
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
    private static func extractMeta(property: String, name: String, from html: String) -> String? {
        let lowered = html.lowercased()

        for pattern in [
            "property=\"\(property)\"",
            "property='\(property)'",
            "name=\"\(name)\"",
            "name='\(name)'"
        ] {
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
