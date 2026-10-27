import Foundation

public struct CalendarSubscription: Codable, Identifiable, Hashable, Sendable {
    public let id: String
    public let name: String
    public let url: String
    public var color: String
    public let lastFetchedAt: Date?
    public let createdAt: Date

    public init(id: String, name: String, url: String, color: String, lastFetchedAt: Date? = nil, createdAt: Date = .now) {
        self.id = id
        self.name = name
        self.url = url
        self.color = color
        self.lastFetchedAt = lastFetchedAt
        self.createdAt = createdAt
    }
}

public enum CalendarSubscriptionColor: String, CaseIterable, Identifiable, Sendable {
    case indigo, violet, rose, red, orange, amber, emerald, teal, sky, blue

    public var id: String { rawValue }

    public var hex: String {
        switch self {
        case .indigo: "#6366F1"
        case .violet: "#8B5CF6"
        case .rose: "#F43F5E"
        case .red: "#EF4444"
        case .orange: "#F97316"
        case .amber: "#F59E0B"
        case .emerald: "#10B981"
        case .teal: "#14B8A6"
        case .sky: "#0EA5E9"
        case .blue: "#3B82F6"
        }
    }
}
