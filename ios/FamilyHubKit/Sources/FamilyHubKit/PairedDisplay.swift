import Foundation

public struct PairedDisplay: Codable, Identifiable, Equatable {
    public let id: String
    public let name: String
    public let pairedByUserID: String
    public let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case pairedByUserID = "pairedByUserId"
        case createdAt
    }

    public init(id: String, name: String, pairedByUserID: String, createdAt: Date) {
        self.id = id
        self.name = name
        self.pairedByUserID = pairedByUserID
        self.createdAt = createdAt
    }
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
