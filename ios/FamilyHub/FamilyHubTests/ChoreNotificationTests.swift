import XCTest
import FamilyHubKit
@testable import FamilyHub

@MainActor
private final class FakeNotificationSystem: ChoreNotificationSystem {
    var allowed = true
    var registers = 0
    var unregisters = 0
    func requestPermission() async throws -> Bool { allowed }
    func hasPermission() async -> Bool { allowed }
    func register() { registers += 1 }
    func unregister() { unregisters += 1 }
    func clearDelivered() {}
}

@MainActor
final class ChoreNotificationTests: XCTestCase {
    func testPermissionDenialDoesNotRegisterADevice() async throws {
        let system = FakeNotificationSystem()
        system.allowed = false
        let manager = ChoreNotificationManager(system: system, defaults: UserDefaults(suiteName: UUID().uuidString)!)
        try await manager.configure(apiClient: FakeAPIClient())
        do {
            try await manager.enable()
            XCTFail("Permission denial must be visible")
        } catch {}
        XCTAssertFalse(manager.isEnabled)
        XCTAssertEqual(system.registers, 0)
    }

    func testTokenReplacementDisablementAndSignOut() async throws {
        let system = FakeNotificationSystem()
        let manager = ChoreNotificationManager(system: system, defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let client = FakeAPIClient()
        try await manager.configure(apiClient: client)
        try await manager.enable()
        await manager.receivedToken(Data([0xab, 0xcd]), environment: "sandbox")
        XCTAssertEqual(client.notificationDevices.values.first?.token, "abcd")
        await manager.receivedToken(Data([0xef, 0x01]), environment: "sandbox")
        XCTAssertEqual(client.notificationDevices.count, 1)
        XCTAssertEqual(client.notificationDevices.values.first?.token, "ef01")
        try await manager.disconnect()
        XCTAssertTrue(client.notificationDevices.isEmpty)
        XCTAssertFalse(manager.isEnabled)
        await manager.receivedToken(Data([0x01]), environment: "sandbox")
        XCTAssertTrue(client.notificationDevices.isEmpty)
    }

    func testNotificationOpensChoresFromAnotherTab() {
        let navigation = AppNavigation()
        navigation.selectedTab = .food
        navigation.open(URL(string: "familyhub://chores")!)
        XCTAssertEqual(navigation.selectedTab, .tasks)
    }
    func testServerChangeRemovesTheOldRegistration() async throws {
        let manager = ChoreNotificationManager(system: FakeNotificationSystem(), defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let oldClient = FakeAPIClient()
        oldClient.serverIdentifier = "https://old.example"
        try await manager.configure(apiClient: oldClient)
        try await manager.enable()
        await manager.receivedToken(Data([0xab]), environment: "sandbox")
        let newClient = FakeAPIClient()
        newClient.serverIdentifier = "https://new.example"
        try await manager.configure(apiClient: newClient)
        XCTAssertTrue(oldClient.notificationDevices.isEmpty)
        XCTAssertTrue(newClient.notificationDevices.isEmpty)
        XCTAssertFalse(manager.isEnabled)
    }

    func testFailedDisablementKeepsTheDeviceStateVisible() async throws {
        let manager = ChoreNotificationManager(system: FakeNotificationSystem(), defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let client = FakeAPIClient()
        try await manager.configure(apiClient: client)
        try await manager.enable()
        await manager.receivedToken(Data([0xab]), environment: "sandbox")
        client.removeNotificationDeviceError = APIError.offline
        do {
            try await manager.disconnect()
            XCTFail("An offline sign-out must report the registration that remains")
        } catch {}
        XCTAssertTrue(manager.isEnabled)
        XCTAssertEqual(client.notificationDevices.count, 1)
        client.removeNotificationDeviceError = nil
        try await manager.disconnect()
        XCTAssertFalse(manager.isEnabled)
        XCTAssertTrue(client.notificationDevices.isEmpty)
    }

    func testRegistrationFailureIsVisibleAndCanRetry() async throws {
        let manager = ChoreNotificationManager(system: FakeNotificationSystem(), defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let client = FakeAPIClient()
        try await manager.configure(apiClient: client)
        try await manager.enable()
        client.registerNotificationDeviceError = APIError.offline
        await manager.receivedToken(Data([0xab]), environment: "sandbox")
        XCTAssertFalse(manager.isRegistered)
        XCTAssertNotNil(manager.actionError)
        client.registerNotificationDeviceError = nil
        await manager.receivedToken(Data([0xab]), environment: "sandbox")
        XCTAssertTrue(manager.isRegistered)
        XCTAssertNil(manager.actionError)
    }

    func testOfflineDisconnectWorksWhenNotificationsAreDisabled() async throws {
        let manager = ChoreNotificationManager(system: FakeNotificationSystem(), defaults: UserDefaults(suiteName: UUID().uuidString)!)
        let client = FakeAPIClient()
        client.removeNotificationDeviceError = APIError.offline
        try await manager.configure(apiClient: client)
        try await manager.disconnect()
        XCTAssertFalse(manager.isEnabled)
    }

}
