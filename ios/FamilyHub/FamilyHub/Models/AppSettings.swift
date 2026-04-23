import Foundation

struct AppSettings: Codable {
    let familyName: String

    enum CodingKeys: String, CodingKey {
        case familyName = "family_name"
    }
}
