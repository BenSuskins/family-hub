import XCTest
@testable import FamilyHub

final class InMemoryKeychainStore: KeychainStoring {
    private(set) var accessToken: String?
    private(set) var refreshToken: String?
    private(set) var apiToken: String?

    func save(accessToken: String, refreshToken: String) {
        self.accessToken = accessToken
        self.refreshToken = refreshToken
    }

    func saveAPIToken(_ token: String) {
        self.apiToken = token
    }

    func clear() {
        accessToken = nil
        refreshToken = nil
        apiToken = nil
    }
}

final class KeychainStoreTests: XCTestCase {
    var store: InMemoryKeychainStore!

    override func setUp() {
        super.setUp()
        store = InMemoryKeychainStore()
    }

    func testSaveAndReadTokens() {
        store.save(accessToken: "access123", refreshToken: "refresh456")
        XCTAssertEqual(store.accessToken, "access123")
        XCTAssertEqual(store.refreshToken, "refresh456")
    }

    func testClearRemovesTokens() {
        store.save(accessToken: "access123", refreshToken: "refresh456")
        store.clear()
        XCTAssertNil(store.accessToken)
        XCTAssertNil(store.refreshToken)
    }

    func testReadBeforeSaveReturnsNil() {
        XCTAssertNil(store.accessToken)
        XCTAssertNil(store.refreshToken)
    }

    func testOverwriteToken() {
        store.save(accessToken: "old", refreshToken: "old-refresh")
        store.save(accessToken: "new", refreshToken: "new-refresh")
        XCTAssertEqual(store.accessToken, "new")
    }

    func testSaveAPIToken() {
        store.saveAPIToken("api-token-abc")
        XCTAssertEqual(store.apiToken, "api-token-abc")
    }

    func testClearRemovesAPIToken() {
        store.saveAPIToken("api-token-abc")
        store.clear()
        XCTAssertNil(store.apiToken)
    }
}
