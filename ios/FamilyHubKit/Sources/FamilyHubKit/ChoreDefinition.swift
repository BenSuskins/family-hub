import Foundation

public struct ChoreDefinition: Codable, Identifiable {
    public let id: String
    public let chore: Chore
    public let currentTask: Chore?
    public let active: Bool
    public let deletedAt: String?

    public init(id: String, chore: Chore, currentTask: Chore?, active: Bool, deletedAt: String? = nil) {
        self.id = id
        self.chore = chore
        self.currentTask = currentTask
        self.active = active
        self.deletedAt = deletedAt
    }

    enum CodingKeys: String, CodingKey {
        case id = "ID", chore = "Chore", currentTask = "CurrentTask", active = "Active", deletedAt = "DeletedAt"
    }
}

public struct ChoreHistoryPage: Codable {
    public let items: [Chore]
    public let nextCursor: String

    public init(items: [Chore], nextCursor: String = "") {
        self.items = items
        self.nextCursor = nextCursor
    }

    enum CodingKeys: String, CodingKey {
        case items = "Items", nextCursor = "NextCursor"
    }
}
