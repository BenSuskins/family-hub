import Foundation

public struct User: Codable, Identifiable {
    public let id: String
    public let name: String
    public let email: String
    public let avatarURL: String
    public let role: String

    public init(id: String, name: String, email: String, avatarURL: String, role: String) {
        self.id = id
        self.name = name
        self.email = email
        self.avatarURL = avatarURL
        self.role = role
    }

    public var isAdmin: Bool { role == "admin" }

    public var initials: String {
        let parts = name.split(separator: " ").prefix(2)
        guard !parts.isEmpty else { return "?" }
        return parts.compactMap { $0.first.map(String.init) }.joined()
    }

    enum CodingKeys: String, CodingKey {
        case id        = "ID"
        case name      = "Name"
        case email     = "Email"
        case avatarURL = "AvatarURL"
        case role      = "Role"
    }
}

extension Sequence where Element == User {
    /// Index users by id for O(1) lookup. On the (server-prevented) chance of a
    /// duplicate id, the last one wins.
    public var keyedByID: [String: User] {
        Dictionary(map { ($0.id, $0) }, uniquingKeysWith: { _, latest in latest })
    }
}
