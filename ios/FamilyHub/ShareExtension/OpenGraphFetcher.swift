import Foundation

enum OpenGraphFetcher {
    static func fetchImageAsDataURI(from urlString: String) async -> String? {
        guard let url = URL(string: urlString) else { return nil }
        let config = URLSessionConfiguration.ephemeral
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
}
