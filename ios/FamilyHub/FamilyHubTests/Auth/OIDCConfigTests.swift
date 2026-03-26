import XCTest
@testable import FamilyHub

@MainActor
final class OIDCConfigTests: XCTestCase {
    private func makeStore(
        baseURL: String = "https://hub.example.com",
        clientID: String = "my-client",
        authorizationEndpoint: String = "https://auth.example.com/authorize",
        tokenEndpoint: String = "https://auth.example.com/token"
    ) -> ConfigStore {
        let store = ConfigStore(defaults: UserDefaults(suiteName: "test.\(UUID().uuidString)")!)
        store.baseURL = baseURL
        store.clientID = clientID
        store.authorizationEndpoint = authorizationEndpoint
        store.tokenEndpoint = tokenEndpoint
        return store
    }

    func testValidHTTPURLsSucceeds() throws {
        let store = makeStore(
            baseURL: "http://192.168.1.10:8080",
            authorizationEndpoint: "http://auth.local/authorize",
            tokenEndpoint: "http://auth.local/token"
        )
        let config = try OIDCConfig.from(configStore: store)
        XCTAssertEqual(config.clientID, "my-client")
        XCTAssertEqual(config.baseURL.scheme, "http")
    }

    func testValidHTTPSURLsSucceeds() throws {
        let store = makeStore()
        let config = try OIDCConfig.from(configStore: store)
        XCTAssertEqual(config.baseURL.absoluteString, "https://hub.example.com")
        XCTAssertEqual(config.authorizationEndpoint.absoluteString, "https://auth.example.com/authorize")
        XCTAssertEqual(config.tokenEndpoint.absoluteString, "https://auth.example.com/token")
    }

    func testEmptyBaseURLThrowsEmptyField() {
        let store = makeStore(baseURL: "")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .emptyField(let field) = configError else {
                return XCTFail("Expected emptyField, got \(error)")
            }
            XCTAssertEqual(field, "Server URL")
        }
    }

    func testEmptyClientIDThrowsEmptyField() {
        let store = makeStore(clientID: "")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .emptyField(let field) = configError else {
                return XCTFail("Expected emptyField, got \(error)")
            }
            XCTAssertEqual(field, "Client ID")
        }
    }

    func testEmptyAuthorizationEndpointThrowsEmptyField() {
        let store = makeStore(authorizationEndpoint: "")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .emptyField(let field) = configError else {
                return XCTFail("Expected emptyField, got \(error)")
            }
            XCTAssertEqual(field, "Authorization Endpoint")
        }
    }

    func testEmptyTokenEndpointThrowsEmptyField() {
        let store = makeStore(tokenEndpoint: "")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .emptyField(let field) = configError else {
                return XCTFail("Expected emptyField, got \(error)")
            }
            XCTAssertEqual(field, "Token Endpoint")
        }
    }

    func testNonURLStringThrowsInvalidURL() {
        let store = makeStore(baseURL: "not a url at all")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .invalidURL(let field, _) = configError else {
                return XCTFail("Expected invalidURL, got \(error)")
            }
            XCTAssertEqual(field, "Server URL")
        }
    }

    func testFileURLThrowsInvalidURL() {
        let store = makeStore(baseURL: "file:///etc/hosts")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .invalidURL = configError else {
                return XCTFail("Expected invalidURL, got \(error)")
            }
        }
    }

    func testRelativePathThrowsInvalidURL() {
        let store = makeStore(baseURL: "relative/path")
        XCTAssertThrowsError(try OIDCConfig.from(configStore: store)) { error in
            guard let configError = error as? ConfigurationError,
                  case .invalidURL = configError else {
                return XCTFail("Expected invalidURL, got \(error)")
            }
        }
    }
}
