import Foundation

struct APIToken: Codable, Identifiable {
    let id: String
    let name: String
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id        = "ID"
        case name      = "Name"
        case createdAt = "CreatedAt"
    }
}

struct CreatedToken: Codable {
    let id: String
    let name: String
    let plaintext: String

    enum CodingKeys: String, CodingKey {
        case id        = "id"
        case name      = "name"
        case plaintext = "token"
    }
}
