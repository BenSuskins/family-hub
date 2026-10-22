import XCTest
import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
@testable import FamilyHubKit

private final class AvatarURLProtocol: URLProtocol, @unchecked Sendable {
    override class func canInit(with request: URLRequest) -> Bool { request.url?.host == "avatar.example" }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let upload = request.httpMethod == "POST"
        let data = upload
            ? Data(#"{"ID":"one","Name":"Test User","Email":"","AvatarURL":"/avatar/one","Role":"member"}"#.utf8)
            : Data((request.value(forHTTPHeaderField: "Authorization") == "Bearer after-upload" ? "new-photo" : "old-photo").utf8)
        client?.urlProtocol(self, didReceive: HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}
}

final class AvatarAPIClientTests: XCTestCase {
    @MainActor
    func testUploadedPhotoReplacesTheCachedPhoto() async throws {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [AvatarURLProtocol.self]
        let session = URLSession(configuration: configuration)
        defer { session.invalidateAndCancel() }
        var token = "before-upload"
        let provider = StaticTokenProvider(read: { token })
        let client = APIClient(baseURL: URL(string: "https://avatar.example")!, session: session, tokenProvider: provider, retryPolicy: .none)
        let previous = try await client.fetchUserAvatar(id: "one")
        XCTAssertEqual(previous, Data("old-photo".utf8))
        _ = try await client.uploadAvatar(imageData: Data("new-photo".utf8), mimeType: "image/png")
        token = "after-upload"
        let updated = try await client.fetchUserAvatar(id: "one")
        XCTAssertEqual(updated, Data("new-photo".utf8))
    }
}
