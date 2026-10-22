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
}
