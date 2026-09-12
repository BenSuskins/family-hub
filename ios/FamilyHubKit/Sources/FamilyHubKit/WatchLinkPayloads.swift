import Foundation

/// What the phone sends the watch so it can talk to the server.
///
/// The watch can never obtain these itself: signing in needs
/// `ASWebAuthenticationSession`, which does not exist on watchOS, and neither
/// the keychain nor the app group crosses the device boundary. So the phone
/// hands them over via `WCSession.updateApplicationContext`, whose
/// latest-state-wins, delivered-in-the-background semantics are exactly right
/// for "here is the current token".
public struct WatchCredentials: Codable, Equatable, Sendable {
    public let apiToken: String
    public let baseURL: String

    public init(apiToken: String, baseURL: String) {
        self.apiToken = apiToken
        self.baseURL = baseURL
    }
}

/// A recipe handed from the phone to the watch to cook.
///
/// The whole recipe travels, not just its id. A recipe is a few KB of text, and
/// carrying it means the watch can cook with no network at all — which is the
/// point, given a kitchen is often the worst reception in the house. Sent with
/// `transferUserInfo`, which queues and is guaranteed, so it survives the phone
/// being asleep or out of range.
///
/// The image is deliberately absent: cook mode never shows one.
public struct CookHandoff: Codable, Equatable, Sendable {
    public let recipe: Recipe
    /// When the phone sent it, used to pick the newest if several queue up.
    public let sentAt: Date

    public init(recipe: Recipe, sentAt: Date = Date()) {
        self.recipe = recipe
        self.sentAt = sentAt
    }
}

// MARK: - Dictionary bridging

// WatchConnectivity speaks `[String: Any]`, and accepts **property-list types
// only**. An earlier version of this file decomposed the encoded JSON into a
// dictionary of plain values with `JSONSerialization.jsonObject`, which looks
// right and is not: JSON `null` decodes to `NSNull`, and `NSNull` is not a
// property-list type.
//
// `Recipe.stepDurations` is `[Int?]` — a step with no recognisable timing
// encodes as `null` — so virtually every real recipe produced a payload
// WatchConnectivity would not carry, and the handoff was dropped in transit
// while the phone reported success.
//
// So the JSON travels as a single `Data` value instead. `Data` *is* a
// property-list type, which makes the payload valid by construction and
// independent of whatever optionality the models grow later.

/// The one key both payloads use.
nonisolated(unsafe) private let watchPayloadKey = "json"

extension Encodable {
    /// Encode to the `[String: Any]` dictionary WatchConnectivity requires.
    public func watchPayload() throws -> [String: Any] {
        [watchPayloadKey: try JSONEncoder.watchLink.encode(self)]
    }
}

extension Decodable {
    /// Decode from a WatchConnectivity payload dictionary.
    public static func fromWatchPayload(_ payload: [String: Any]) throws -> Self {
        let data: Data
        if let wrapped = payload[watchPayloadKey] as? Data {
            data = wrapped
        } else {
            // A payload in the old decomposed shape. Accepted so a watch that
            // updates before the phone — or one holding a stored application
            // context from the previous build — still reads it.
            data = try JSONSerialization.data(withJSONObject: payload)
        }
        return try JSONDecoder.watchLink.decode(Self.self, from: data)
    }
}

/// Whether a value is one WatchConnectivity will actually carry.
///
/// `WCSession` raises rather than returning an error when handed something
/// else, so this is worth asserting in tests rather than discovering on a
/// wrist. Mirrors the property-list type list: string, number, boolean, date,
/// data, and arrays and dictionaries of those.
public nonisolated func isWatchTransportable(_ value: Any) -> Bool {
    switch value {
    case is String, is Bool, is Int, is Double, is Date, is Data:
        return true
    case let array as [Any]:
        return array.allSatisfy(isWatchTransportable)
    case let dictionary as [String: Any]:
        return dictionary.values.allSatisfy(isWatchTransportable)
    default:
        // Covers NSNull, and anything else that would raise in transit.
        return false
    }
}

extension JSONEncoder {
    /// Shared encoder for watch payloads. ISO8601 dates so `sentAt` survives.
    public static let watchLink: JSONEncoder = {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        return encoder
    }()
}

extension JSONDecoder {
    public static let watchLink: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return decoder
    }()
}
