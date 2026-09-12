import Foundation

public struct ChoreCategory: Codable, Identifiable {
    public let id: String
    public let name: String

    public init(id: String, name: String) {
        self.id = id
        self.name = name
    }

    enum CodingKeys: String, CodingKey {
        case id   = "ID"
        case name = "Name"
    }
}
