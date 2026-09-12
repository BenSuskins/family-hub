import Foundation

public struct AppSettings: Codable {
    public let familyName: String

    public init(familyName: String) {
        self.familyName = familyName
    }

    enum CodingKeys: String, CodingKey {
        case familyName = "family_name"
    }
}
