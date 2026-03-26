import XCTest
@testable import FamilyHub

@MainActor
final class ConfigStoreTests: XCTestCase {
    private func makeStore() -> ConfigStore {
        ConfigStore(defaults: UserDefaults(suiteName: "test.\(UUID().uuidString)")!)
    }

    func testIsConfiguredFalseWhenAllFieldsEmpty() {
        let store = makeStore()
        XCTAssertFalse(store.isConfigured)
    }

    func testIsConfiguredFalseWhenAnyFieldEmpty() {
        let store = makeStore()
        store.baseURL = "https://hub.example.com"
        store.clientID = "client"
        store.authorizationEndpoint = "https://auth.example.com/authorize"
        // tokenEndpoint left empty
        XCTAssertFalse(store.isConfigured)
    }

    func testIsConfiguredTrueWhenAllFieldsNonEmpty() {
        let store = makeStore()
        store.baseURL = "https://hub.example.com"
        store.clientID = "client"
        store.authorizationEndpoint = "https://auth.example.com/authorize"
        store.tokenEndpoint = "https://auth.example.com/token"
        XCTAssertTrue(store.isConfigured)
    }

    func testSaveAndInitRoundTrip() {
        let suiteName = "test.\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: suiteName)!

        let store = ConfigStore(defaults: defaults)
        store.baseURL = "https://hub.example.com"
        store.clientID = "my-client"
        store.authorizationEndpoint = "https://auth.example.com/authorize"
        store.tokenEndpoint = "https://auth.example.com/token"
        store.save()

        let reloaded = ConfigStore(defaults: defaults)
        XCTAssertEqual(reloaded.baseURL, "https://hub.example.com")
        XCTAssertEqual(reloaded.clientID, "my-client")
        XCTAssertEqual(reloaded.authorizationEndpoint, "https://auth.example.com/authorize")
        XCTAssertEqual(reloaded.tokenEndpoint, "https://auth.example.com/token")
    }

    func testSaveDoesNotMutateProperties() {
        let store = makeStore()
        store.baseURL = "https://hub.example.com"
        store.clientID = "my-client"
        store.authorizationEndpoint = "https://auth.example.com/authorize"
        store.tokenEndpoint = "https://auth.example.com/token"

        var changeCount = 0
        withObservationTracking {
            _ = store.baseURL
            _ = store.clientID
            _ = store.authorizationEndpoint
            _ = store.tokenEndpoint
        } onChange: {
            changeCount += 1
        }

        store.save()

        XCTAssertEqual(changeCount, 0)
    }
}
