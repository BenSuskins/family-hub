import Foundation

public struct HomeInfo: Codable {
    public var latitude: Double?
    public var longitude: Double?
    public var timezone: String
    public var arrivalMarginMinutes: Int

    public init(latitude: Double? = nil, longitude: Double? = nil, timezone: String = TimeZone.current.identifier, arrivalMarginMinutes: Int = 10) {
        self.latitude = latitude
        self.longitude = longitude
        self.timezone = timezone
        self.arrivalMarginMinutes = arrivalMarginMinutes
    }
}

public struct WeatherHour: Codable, Identifiable {
    public let time: String
    public let temperature: Double
    public let code: Int
    public let rainProbability: Double
    public let windSpeed: Double
    public var id: String { time }

    public var condition: String {
        switch code {
        case 0: return "Clear"
        case 1, 2: return "Partly cloudy"
        case 3: return "Overcast"
        case 45, 48: return "Fog"
        case 51...67, 80...82: return "Rain"
        case 71...77, 85, 86: return "Snow"
        case 95...99: return "Thunderstorms"
        default: return "Unknown conditions"
        }
    }

    public var symbol: String {
        switch code {
        case 0: return "sun.max"
        case 1, 2: return "cloud.sun"
        case 45, 48: return "cloud.fog"
        case 51...67, 80...82: return "cloud.rain"
        case 71...77, 85, 86: return "cloud.snow"
        case 95...99: return "cloud.bolt.rain"
        default: return "cloud"
        }
    }
}

public struct HouseholdWeather: Codable {
    public let status: String
    public let fetchedAt: String
    public let current: WeatherHour
    public let high: Double
    public let low: Double
    public let hours: [WeatherHour]
    public let advice: [String]
}

public struct Departure: Codable, Identifiable {
    public let eventID: String
    public let title: String
    public let destination: String
    public let startTime: String
    public let leaveAt: String
    public let drivingSeconds: Int
    public let fetchedAt: String
    public var id: String { eventID }

    public func countdown(at now: Date) -> String? {
        guard let start = APIDate.parse(startTime), let departure = APIDate.parse(leaveAt), now < start else { return nil }
        if now >= departure { return "Leave now" }
        let minutes = Int(ceil(departure.timeIntervalSince(now) / 60))
        let hours = minutes / 60
        let remainingMinutes = minutes % 60
        guard hours > 0 else { return "Leave in \(minutes) min" }
        if remainingMinutes == 0 {
            return "Leave in \(hours) hr"
        }
        return "Leave in \(hours) hr \(remainingMinutes) min"
    }
}

public struct HouseholdDepartures: Codable {
    public let status: String
    public let items: [Departure]
    public let selectedEventID: String?
}

public struct ShoppingItem: Codable, Identifiable {
    public let id: String
    public let name: String
    public let inventoryItemID: String?
    public let purchased: Bool
    public let purchaseID: String?
    public let restocked: Bool
    public let version: Int

    public init(id: String, name: String, inventoryItemID: String? = nil, purchased: Bool = false, purchaseID: String? = nil, restocked: Bool = false, version: Int = 0) {
        self.id = id
        self.name = name
        self.inventoryItemID = inventoryItemID
        self.purchased = purchased
        self.purchaseID = purchaseID
        self.restocked = restocked
        self.version = version
    }
}

public struct ShoppingChange: Encodable {
    public let name: String?
    public let purchased: Bool?
    public let version: Int
    public init(name: String? = nil, purchased: Bool? = nil, version: Int) {
        self.name = name
        self.purchased = purchased
        self.version = version
    }
}

public struct ShoppingRestock: Encodable {
    public let purchaseID: String
    public let quantity: Int?
    public let level: Int?
    public init(purchaseID: String, quantity: Int? = nil, level: Int? = nil) {
        self.purchaseID = purchaseID
        self.quantity = quantity
        self.level = level
    }
}

public struct ShoppingSnapshot: Codable {
    public let items: [ShoppingItem]
    public let capturedAt: Date
}

public final class ShoppingCache {
    private let defaults: UserDefaults
    private static let prefix = "shopping_snapshot_"
    private static let generationKey = "shopping_cache_generation"
    private let generation: Int
    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        self.generation = defaults.integer(forKey: Self.generationKey)
    }

    private func key(server: String, userID: String) -> String {
        Self.prefix + Data(server.utf8).base64EncodedString() + ":" + Data(userID.utf8).base64EncodedString()
    }
    public func save(_ items: [ShoppingItem], server: String, userID: String, at date: Date = Date()) {
        guard generation == defaults.integer(forKey: Self.generationKey) else { return }
        let snapshot = ShoppingSnapshot(items: items, capturedAt: date)
        defaults.set(try? JSONEncoder().encode(snapshot), forKey: key(server: server, userID: userID))
        defaults.set(userID, forKey: Self.prefix + "user_" + server)
    }
    public func load(server: String, userID: String) -> ShoppingSnapshot? {
        guard let data = defaults.data(forKey: key(server: server, userID: userID)) else { return nil }
        return try? JSONDecoder().decode(ShoppingSnapshot.self, from: data)
    }
    public func lastUserID(server: String) -> String? {
        defaults.string(forKey: Self.prefix + "user_" + server)
    }
    public func clear() {
        defaults.set(defaults.integer(forKey: Self.generationKey) + 1, forKey: Self.generationKey)
        for key in defaults.dictionaryRepresentation().keys where key.hasPrefix(Self.prefix) {
            defaults.removeObject(forKey: key)
        }
    }
}
