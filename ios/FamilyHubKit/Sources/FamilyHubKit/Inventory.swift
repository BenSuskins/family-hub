import Foundation

/// A storage area in the home (e.g. "Laundry cupboard") holding stocked items.
/// `icon` and `tint` are presentation hints chosen from fixed sets (see
/// InventoryStyle); the server stores them verbatim.
public struct InventoryArea: Codable, Identifiable {
    public let id: String
    public let name: String
    public let icon: String
    public let tint: String
    public let items: [InventoryItem]

    enum CodingKeys: String, CodingKey {
        case id    = "ID"
        case name  = "Name"
        case icon  = "Icon"
        case tint  = "Tint"
        case items = "Items"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decode(String.self, forKey: .name)
        icon = (try? c.decode(String.self, forKey: .icon)) ?? "box"
        tint = (try? c.decode(String.self, forKey: .tint)) ?? "blue"
        items = (try? c.decode([InventoryItem].self, forKey: .items)) ?? []
    }

    public init(id: String, name: String, icon: String, tint: String, items: [InventoryItem]) {
        self.id = id
        self.name = name
        self.icon = icon
        self.tint = tint
        self.items = items
    }

    /// Items at or below their low-at threshold.
    public var lowItems: [InventoryItem] { items.filter(\.isLow) }
    public var lowCount: Int { lowItems.count }
}

/// How an item's stock is measured: `count` in whole units, or `level` as a
/// 0–100 fill percentage (for partially-used items like bottles).
public enum TrackingMode: String, Codable {
    case count
    case level
}

/// A stocked consumable within an area. "Low" is derived as `quantity <= lowAt`
/// for count items, or `level <= lowAt` (read as a percentage) for level items.
public struct InventoryItem: Codable, Identifiable {
    public let id: String
    public let areaID: String
    public let name: String
    public let trackingMode: TrackingMode
    public let quantity: Int
    public let level: Int
    public let unit: String
    public let lowAt: Int

    enum CodingKeys: String, CodingKey {
        case id           = "ID"
        case areaID       = "AreaID"
        case name         = "Name"
        case trackingMode = "TrackingMode"
        case quantity     = "Quantity"
        case level        = "Level"
        case unit         = "Unit"
        case lowAt        = "LowAt"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        areaID = (try? c.decode(String.self, forKey: .areaID)) ?? ""
        name = try c.decode(String.self, forKey: .name)
        trackingMode = (try? c.decode(TrackingMode.self, forKey: .trackingMode)) ?? .count
        quantity = (try? c.decode(Int.self, forKey: .quantity)) ?? 0
        level = (try? c.decode(Int.self, forKey: .level)) ?? 100
        unit = (try? c.decode(String.self, forKey: .unit)) ?? ""
        lowAt = (try? c.decode(Int.self, forKey: .lowAt)) ?? 0
    }

    public init(id: String, areaID: String, name: String, trackingMode: TrackingMode = .count,
         quantity: Int, level: Int = 100, unit: String, lowAt: Int) {
        self.id = id
        self.areaID = areaID
        self.name = name
        self.trackingMode = trackingMode
        self.quantity = quantity
        self.level = level
        self.unit = unit
        self.lowAt = lowAt
    }

    public var isLow: Bool {
        switch trackingMode {
        case .count: return quantity <= lowAt
        case .level: return level <= lowAt
        }
    }

    /// Short stock summary: "{level}%" for level items, else "{qty} {unit}"
    /// (unit omitted when blank).
    public var statusLabel: String {
        switch trackingMode {
        case .level: return "\(level)%"
        case .count: return unit.isEmpty ? "\(quantity)" : "\(quantity) \(unit)"
        }
    }

    /// Returns a copy with count quantity changed by `delta`, clamped at 0. No-op
    /// for level items (their stock is the fill percentage, not a unit count).
    public func adjusting(by delta: Int) -> InventoryItem {
        guard trackingMode == .count else { return self }
        return InventoryItem(id: id, areaID: areaID, name: name, trackingMode: trackingMode,
                             quantity: max(0, quantity + delta), level: level, unit: unit, lowAt: lowAt)
    }

    /// Returns a copy with the fill percentage set to `newLevel`, clamped to 0–100.
    public func withLevel(_ newLevel: Int) -> InventoryItem {
        InventoryItem(id: id, areaID: areaID, name: name, trackingMode: trackingMode,
                      quantity: quantity, level: min(100, max(0, newLevel)), unit: unit, lowAt: lowAt)
    }
}

/// A low item paired with the area it belongs to, for the cross-area
/// "Running low" rollup on the home screen.
public struct RunningLowItem: Identifiable {
    public let area: InventoryArea
    public let item: InventoryItem
    public var id: String { item.id }

    public init(area: InventoryArea, item: InventoryItem) {
        self.area = area
        self.item = item
    }
}

// MARK: - Request bodies (camelCase to match the server)

public struct AreaRequest: Encodable {
    public let name: String
    public let icon: String
    public let tint: String

    public init(name: String, icon: String, tint: String) {
        self.name = name
        self.icon = icon
        self.tint = tint
    }
}

public struct ItemRequest: Encodable {
    public let name: String
    public var trackingMode: TrackingMode = .count
    public let quantity: Int
    public var level: Int = 100
    public let unit: String
    public let lowAt: Int

    public init(
        name: String,
        trackingMode: TrackingMode = .count,
        quantity: Int,
        level: Int = 100,
        unit: String,
        lowAt: Int
    ) {
        self.name = name
        self.trackingMode = trackingMode
        self.quantity = quantity
        self.level = level
        self.unit = unit
        self.lowAt = lowAt
    }
}

extension InventoryItem {
    /// The full request body that round-trips this item unchanged — used by
    /// optimistic mutations (stepper, level slider) that persist a single field.
    public var asRequest: ItemRequest {
        ItemRequest(name: name, trackingMode: trackingMode, quantity: quantity,
                    level: level, unit: unit, lowAt: lowAt)
    }
}
