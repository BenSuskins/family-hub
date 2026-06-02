import Foundation

struct User: Codable, Identifiable {
    let id: String
    let name: String
    let email: String
    let avatarURL: String
    let role: String

    var isAdmin: Bool { role == "admin" }

    var initials: String {
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
    var keyedByID: [String: User] {
        Dictionary(map { ($0.id, $0) }, uniquingKeysWith: { _, latest in latest })
    }
}
