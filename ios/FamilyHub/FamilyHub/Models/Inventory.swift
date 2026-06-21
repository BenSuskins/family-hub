import Foundation

/// A storage area in the home (e.g. "Laundry cupboard") holding stocked items.
/// `icon` and `tint` are presentation hints chosen from fixed sets (see
/// InventoryStyle); the server stores them verbatim.
struct InventoryArea: Codable, Identifiable {
    let id: String
    let name: String
    let icon: String
    let tint: String
    let items: [InventoryItem]

    enum CodingKeys: String, CodingKey {
        case id    = "ID"
        case name  = "Name"
        case icon  = "Icon"
        case tint  = "Tint"
        case items = "Items"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decode(String.self, forKey: .name)
        icon = (try? c.decode(String.self, forKey: .icon)) ?? "box"
        tint = (try? c.decode(String.self, forKey: .tint)) ?? "blue"
        items = (try? c.decode([InventoryItem].self, forKey: .items)) ?? []
    }

    init(id: String, name: String, icon: String, tint: String, items: [InventoryItem]) {
        self.id = id
        self.name = name
        self.icon = icon
        self.tint = tint
        self.items = items
    }

    /// Items at or below their par threshold.
    var lowItems: [InventoryItem] { items.filter(\.isLow) }
    var lowCount: Int { lowItems.count }
}

/// A stocked consumable within an area. "Low" is derived as `quantity <= par`.
struct InventoryItem: Codable, Identifiable {
    let id: String
    let areaID: String
    let name: String
    let quantity: Int
    let unit: String
    let par: Int

    enum CodingKeys: String, CodingKey {
        case id       = "ID"
        case areaID   = "AreaID"
        case name     = "Name"
        case quantity = "Quantity"
        case unit     = "Unit"
        case par      = "Par"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        areaID = (try? c.decode(String.self, forKey: .areaID)) ?? ""
        name = try c.decode(String.self, forKey: .name)
        quantity = (try? c.decode(Int.self, forKey: .quantity)) ?? 0
        unit = (try? c.decode(String.self, forKey: .unit)) ?? ""
        par = (try? c.decode(Int.self, forKey: .par)) ?? 0
    }

    init(id: String, areaID: String, name: String, quantity: Int, unit: String, par: Int) {
        self.id = id
        self.areaID = areaID
        self.name = name
        self.quantity = quantity
        self.unit = unit
        self.par = par
    }

    var isLow: Bool { quantity <= par }

    /// "{qty} {unit}" with the unit omitted when blank.
    var quantityLabel: String {
        unit.isEmpty ? "\(quantity)" : "\(quantity) \(unit)"
    }

    /// Returns a copy with quantity changed by `delta`, clamped at 0.
    func adjusting(by delta: Int) -> InventoryItem {
        InventoryItem(id: id, areaID: areaID, name: name, quantity: max(0, quantity + delta), unit: unit, par: par)
    }
}

/// A low item paired with the area it belongs to, for the cross-area
/// "Running low" rollup on the home screen.
struct RunningLowItem: Identifiable {
    let area: InventoryArea
    let item: InventoryItem
    var id: String { item.id }
}

// MARK: - Request bodies (camelCase to match the server)

struct AreaRequest: Encodable {
    let name: String
    let icon: String
    let tint: String
}

struct ItemRequest: Encodable {
    let name: String
    let quantity: Int
    let unit: String
    let par: Int
}
