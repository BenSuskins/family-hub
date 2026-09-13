import Foundation

enum OpenGraphFetcher {
    /// Social CDNs (Instagram's especially) reject requests without a browser
    /// user agent, and some check the referring post, so both are sent.
    static func fetchImageAsDataURI(from urlString: String, referer: URL? = nil) async -> String? {
        guard let url = URL(string: urlString), url.scheme?.hasPrefix("http") == true else { return nil }
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 15
        let session = URLSession(configuration: config)

        var request = URLRequest(url: url)
        request.setValue(
            "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
            forHTTPHeaderField: "User-Agent"
        )
        if let referer {
            request.setValue(referer.absoluteString, forHTTPHeaderField: "Referer")
        }

        guard let (data, response) = try? await session.data(for: request),
              let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode),
              !data.isEmpty else { return nil }

        let mimeType = httpResponse.mimeType ?? "image/jpeg"
        let base64 = data.base64EncodedString()
        return "data:\(mimeType);base64,\(base64)"
    }
}
