import XCTest
@testable import FamilyHub

final class OIDCDiscoveryServiceTests: XCTestCase {
    private func makeService(responses: [URL: (Data, Int)]) -> OIDCDiscoveryService {
        URLSessionOIDCDiscoveryService(session: FakeURLSession(responses: responses))
    }

    func testSuccessfulDiscovery() async throws {
        let baseURL = URL(string: "https://hub.example.com")!
        let issuerURL = URL(string: "https://auth.example.com/application/o/familyhub")!

        let clientConfigData = try JSONEncoder().encode([
            "clientID": "familyhub-ios",
            "issuer": issuerURL.absoluteString
        ])

        let discoveryData = try JSONEncoder().encode([
            "authorization_endpoint": "https://auth.example.com/authorize",
            "token_endpoint": "https://auth.example.com/token"
        ])

        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (clientConfigData, 200),
            URL(string: "https://auth.example.com/application/o/familyhub/.well-known/openid-configuration")!: (discoveryData, 200),
        ])

        let result = try await service.discover(baseURL: baseURL)

        XCTAssertEqual(result.clientID, "familyhub-ios")
        XCTAssertEqual(result.authorizationEndpoint.absoluteString, "https://auth.example.com/authorize")
        XCTAssertEqual(result.tokenEndpoint.absoluteString, "https://auth.example.com/token")
    }

    func testClientConfigHTTPErrorThrows() async {
        let baseURL = URL(string: "https://hub.example.com")!
        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (Data(), 500),
        ])

        do {
            _ = try await service.discover(baseURL: baseURL)
            XCTFail("Expected error")
        } catch OIDCDiscoveryError.clientConfigFetchFailed(let statusCode) {
            XCTAssertEqual(statusCode, 500)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testDiscoveryDocumentHTTPErrorThrows() async throws {
        let baseURL = URL(string: "https://hub.example.com")!
        let issuerURL = URL(string: "https://auth.example.com/application/o/familyhub")!

        let clientConfigData = try JSONEncoder().encode([
            "clientID": "familyhub-ios",
            "issuer": issuerURL.absoluteString
        ])

        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (clientConfigData, 200),
            URL(string: "https://auth.example.com/application/o/familyhub/.well-known/openid-configuration")!: (Data(), 404),
        ])

        do {
            _ = try await service.discover(baseURL: baseURL)
            XCTFail("Expected error")
        } catch OIDCDiscoveryError.discoveryDocumentFetchFailed(let statusCode) {
            XCTAssertEqual(statusCode, 404)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testMissingIssuerThrows() async throws {
        let baseURL = URL(string: "https://hub.example.com")!
        let clientConfigData = try JSONEncoder().encode(["clientID": "familyhub-ios"])

        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (clientConfigData, 200),
        ])

        do {
            _ = try await service.discover(baseURL: baseURL)
            XCTFail("Expected error")
        } catch OIDCDiscoveryError.missingField(let field) {
            XCTAssertEqual(field, "issuer")
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }
}

// MARK: - Fake

final class FakeURLSession: URLSessionProtocol {
    private let responses: [URL: (Data, Int)]

    init(responses: [URL: (Data, Int)]) {
        self.responses = responses
    }

    func data(from url: URL) async throws -> (Data, URLResponse) {
        guard let (data, statusCode) = responses[url] else {
            throw URLError(.badURL)
        }
        let response = HTTPURLResponse(url: url, statusCode: statusCode, httpVersion: nil, headerFields: nil)!
        return (data, response)
    }
}
