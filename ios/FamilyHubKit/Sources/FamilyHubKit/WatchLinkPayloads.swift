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

// WatchConnectivity speaks [String: Any], so both payloads round-trip through
// JSON rather than hand-rolling the conversion. Keeping this here means the
// phone and the watch cannot disagree about the encoding.

extension Encodable {
    /// Encode to the `[String: Any]` dictionary WatchConnectivity requires.
    public func watchPayload() throws -> [String: Any] {
        let data = try JSONEncoder.watchLink.encode(self)
        guard let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw WatchPayloadError.notADictionary
        }
        return object
    }
}

extension Decodable {
    /// Decode from a WatchConnectivity payload dictionary.
    public static func fromWatchPayload(_ payload: [String: Any]) throws -> Self {
        let data = try JSONSerialization.data(withJSONObject: payload)
        return try JSONDecoder.watchLink.decode(Self.self, from: data)
    }
}

public enum WatchPayloadError: Error {
    case notADictionary
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
