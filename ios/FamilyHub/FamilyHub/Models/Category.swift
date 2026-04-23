import Foundation

struct Category: Codable, Identifiable {
    let id: String
    let name: String

    enum CodingKeys: String, CodingKey {
        case id   = "ID"
        case name = "Name"
    }
}
