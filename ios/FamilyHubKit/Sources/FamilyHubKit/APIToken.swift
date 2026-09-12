import Foundation

public struct APIToken: Codable, Identifiable {
    public let id: String
    public let name: String
    public let createdAt: Date

    public init(id: String, name: String, createdAt: Date) {
        self.id = id
        self.name = name
        self.createdAt = createdAt
    }

    enum CodingKeys: String, CodingKey {
        case id        = "ID"
        case name      = "Name"
        case createdAt = "CreatedAt"
    }
}

public struct CreatedToken: Codable, Identifiable {
    public let id: String
    public let name: String
    public let plaintext: String

    public init(id: String, name: String, plaintext: String) {
        self.id = id
        self.name = name
        self.plaintext = plaintext
    }

    enum CodingKeys: String, CodingKey {
        case id        = "id"
        case name      = "name"
        case plaintext = "token"
    }
}
