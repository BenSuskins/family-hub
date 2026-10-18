import Foundation

public struct ChoreReminders: Codable, Equatable, Sendable {
    public var enabled: Bool
    public var times: [String]
    public var timezone: String
    public var pushAvailable: Bool

    public init(enabled: Bool = false, times: [String] = ["10:00", "16:30"], timezone: String = "Europe/London", pushAvailable: Bool = false) {
        self.enabled = enabled
        self.times = times
        self.timezone = timezone
        self.pushAvailable = pushAvailable
    }
}

public struct NotificationDevice: Encodable, Equatable, Sendable {
    public let id: String
    public let token: String
    public let environment: String

    public init(id: String, token: String, environment: String) {
        self.id = id
        self.token = token
        self.environment = environment
    }
}

public struct BroadcastResult: Codable, Equatable, Sendable {
    public let accepted: Int
    public let failed: Int

    public init(accepted: Int, failed: Int) {
        self.accepted = accepted
        self.failed = failed
    }
}
