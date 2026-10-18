import Foundation
import UIKit
import UserNotifications
import FamilyHubKit

@MainActor
protocol ChoreNotificationSystem {
    func requestPermission() async throws -> Bool
    func hasPermission() async -> Bool
    func register()
    func unregister()
    func clearDelivered()
}

@MainActor
struct SystemChoreNotifications: ChoreNotificationSystem {
    func requestPermission() async throws -> Bool {
        try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound])
    }

    func hasPermission() async -> Bool {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        return settings.authorizationStatus == .authorized || settings.authorizationStatus == .provisional
    }

    func register() { UIApplication.shared.registerForRemoteNotifications() }
    func unregister() { UIApplication.shared.unregisterForRemoteNotifications() }
    func clearDelivered() { UNUserNotificationCenter.current().removeAllDeliveredNotifications() }
}

@Observable
@MainActor
final class ChoreNotificationManager {
    static let shared = ChoreNotificationManager()

    private(set) var isEnabled = false
    private(set) var isRegistered = false
    var pendingNotificationURL: URL?
    var actionError: APIError?

    private let system: any ChoreNotificationSystem
    private let defaults: UserDefaults
    private let installationID: String
    private var apiClient: (any APIClientProtocol)?
    private var account: String?
    private var generation = UUID()

    init(system: (any ChoreNotificationSystem)? = nil, defaults: UserDefaults = .standard) {
        self.system = system ?? SystemChoreNotifications()
        self.defaults = defaults
        if let identifier = defaults.string(forKey: "notificationInstallationID") {
            installationID = identifier
        } else {
            installationID = UUID().uuidString
            defaults.set(installationID, forKey: "notificationInstallationID")
        }
    }

    func configure(apiClient: any APIClientProtocol) async throws {
        let user = try await apiClient.fetchMe()
        let nextAccount = "\(apiClient.serverIdentifier ?? "demo")|\(user.id)"
        if let account, account != nextAccount { try await disconnect() }
        self.apiClient = apiClient
        account = nextAccount
        isEnabled = defaults.bool(forKey: preferenceKey)
        if isEnabled {
            if await system.hasPermission() {
                system.register()
            } else {
                try await disable()
            }
        }
    }

    func enable() async throws {
        guard apiClient != nil else { throw APIError.unauthorized }
        guard try await system.requestPermission() else {
            throw APIError.badRequest(serverMessage: "Allow notifications in iPhone Settings to receive chore reminders.")
        }
        isEnabled = true
        defaults.set(true, forKey: preferenceKey)
        system.register()
    }

    func disable() async throws {
        if (isEnabled || isRegistered), let apiClient { try await apiClient.removeNotificationDevice(id: installationID) }
        defaults.set(false, forKey: preferenceKey)
        isEnabled = false
        isRegistered = false
        generation = UUID()
        system.unregister()
        system.clearDelivered()
    }

    func disconnect() async throws {
        try await disable()
        apiClient = nil
        account = nil
        pendingNotificationURL = nil
    }

    func signedOut() {
        defaults.set(false, forKey: preferenceKey)
        isEnabled = false
        isRegistered = false
        apiClient = nil
        account = nil
        generation = UUID()
        pendingNotificationURL = nil
        system.unregister()
        system.clearDelivered()
    }

    func receivedToken(_ data: Data, environment: String) async {
        guard isEnabled, let apiClient else { return }
        let currentGeneration = generation
        let token = data.map { String(format: "%02x", $0) }.joined()
        do {
            try await apiClient.registerNotificationDevice(NotificationDevice(id: installationID, token: token, environment: environment))
            guard currentGeneration == generation, isEnabled else {
                try? await apiClient.removeNotificationDevice(id: installationID)
                return
            }
            isRegistered = true
            actionError = nil
        } catch {
            guard currentGeneration == generation else { return }
            isRegistered = false
            actionError = .from(error)
        }
    }

    private var preferenceKey: String { "choreNotifications:\(account ?? "signed-out")" }
}

final class NotificationAppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        UNUserNotificationCenter.current().delegate = self
        return true
    }

    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let environment = Bundle.main.object(forInfoDictionaryKey: "APNsEnvironment") as? String == "development" ? "sandbox" : "production"
        Task { @MainActor in await ChoreNotificationManager.shared.receivedToken(deviceToken, environment: environment) }
    }

    func application(_ application: UIApplication, didFailToRegisterForRemoteNotificationsWithError error: Error) {
        Task { @MainActor in
            ChoreNotificationManager.shared.actionError = .badRequest(serverMessage: "The iPhone cannot register for notifications. Try again later.")
        }
    }

    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification, withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .sound])
    }

    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse, withCompletionHandler completionHandler: @escaping () -> Void) {
        guard let link = response.notification.request.content.userInfo["url"] as? String,
              ["familyhub://chores", "familyhub://today"].contains(link), let url = URL(string: link) else {
            completionHandler()
            return
        }
        Task { @MainActor in
            ChoreNotificationManager.shared.pendingNotificationURL = url
            completionHandler()
        }
    }
}
