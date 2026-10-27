import Foundation

public struct PairedDisplay: Codable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let pairedByUserID: String
    public let createdAt: Date
    public let enabledSections: [String]
    public let commutePreferences: CommutePreferences

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case pairedByUserID = "pairedByUserId"
        case createdAt
        case enabledSections
        case commutePreferences
    }

    public init(id: String, name: String, pairedByUserID: String, createdAt: Date, enabledSections: [String] = PairedDisplay.defaultSections, commutePreferences: CommutePreferences = CommutePreferences()) {
        self.id = id
        self.name = name
        self.pairedByUserID = pairedByUserID
        self.createdAt = createdAt
        self.enabledSections = enabledSections
        self.commutePreferences = commutePreferences
    }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        id = try values.decode(String.self, forKey: .id)
        name = try values.decode(String.self, forKey: .name)
        pairedByUserID = try values.decodeIfPresent(String.self, forKey: .pairedByUserID) ?? ""
        createdAt = try values.decode(Date.self, forKey: .createdAt)
        enabledSections = try values.decodeIfPresent([String].self, forKey: .enabledSections) ?? Self.defaultSections
        commutePreferences = try values.decodeIfPresent(CommutePreferences.self, forKey: .commutePreferences) ?? CommutePreferences()
    }

    public static let defaultSections = ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping", "inventory"]
}

public struct DisplayPairingCode: Codable, Equatable {
    public let apiBaseURL: URL
    public let challengeID: String
    public let pairingCode: String

    enum CodingKeys: String, CodingKey {
        case apiBaseURL = "apiBaseUrl"
        case challengeID = "challengeId"
        case pairingCode
    }
}
