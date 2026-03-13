import XCTest
@testable import FamilyHub

final class KeychainStoreTests: XCTestCase {
    var store: KeychainStore!

    override func setUp() {
        super.setUp()
        store = KeychainStore(service: "uk.co.suskins.familyhub.tests.\(UUID().uuidString)")
    }

    override func tearDown() {
        store.clear()
        super.tearDown()
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
}
