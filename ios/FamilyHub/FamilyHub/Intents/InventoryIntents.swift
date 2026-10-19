import AppIntents
import Foundation
import FamilyHubKit

struct ReadInventoryIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Read Inventory" }
    nonisolated static var description: IntentDescription { "Read the stock of one item or the contents of a storage area." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }
    @Parameter(title: "Item") var item: InventoryItemEntity?
    @Parameter(title: "Area") var area: InventoryAreaEntity?
    @Parameter(title: "Read") var selection: SiriInventoryRead?

    static var parameterSummary: some ParameterSummary {
        Summary("Read inventory") {
            \.$item
            \.$area
            \.$selection
        }
    }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog & ReturnsValue<[String]> {
        let service = try SiriEnvironment.service()
        let kind: SiriInventoryRead
        if item != nil { kind = .item }
        else if area != nil { kind = .area }
        else if let selection { kind = selection }
        else { kind = try await $selection.requestValue("Read one item or a storage area?") }
        let lines: [String]
        if kind == .area {
            let selected: InventoryAreaEntity
            if let area { selected = area }
            else { selected = try await $area.requestValue("Which storage area?") }
            let areas = try await service.inventory()
            try await SiriEnvironment.validate(scope: selected.scope, service: service)
            guard let saved = areas.first(where: { $0.id == selected.rawID }) else { throw APIError.notFound }
            lines = saved.items.map { "\($0.name): \($0.statusLabel)" }
        } else {
            let selected: InventoryItemEntity
            if let item { selected = item }
            else { selected = try await $item.requestValue("Which item?") }
            let areas = try await service.inventory()
            try await SiriEnvironment.validate(scope: selected.scope, service: service)
            guard let savedArea = areas.first(where: { $0.items.contains(where: { $0.id == selected.rawID }) }),
                  let saved = savedArea.items.first(where: { $0.id == selected.rawID }) else {
                throw SiriActionError.invalid("That item is not tracked in Family Hub.")
            }
            let stock = saved.trackingMode == .count ? saved.quantity : saved.level
            lines = [stock == 0 ? "\(saved.name) in \(savedArea.name) is out of stock." : "\(saved.name) in \(savedArea.name): \(saved.statusLabel)"]
        }
        let spoken = SiriNames.summary(lines, empty: "There are no items in that area.")
        return .result(value: lines, dialog: "\(spoken)")
    }
}

struct ReadLowStockIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Read Low Stock" }
    nonisolated static var description: IntentDescription { "Read inventory items at or below their low-stock threshold, optionally in one area." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }
    @Parameter(title: "Area") var area: InventoryAreaEntity?

    static var parameterSummary: some ParameterSummary { Summary("Read low stock in \(\.$area)") }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog & ReturnsValue<[String]> {
        let service = try SiriEnvironment.service()
        if let area { try await SiriEnvironment.validate(scope: area.scope, service: service) }
        let areas = try await service.inventory().filter { area == nil || $0.id == area?.rawID }
        if area != nil && areas.isEmpty { throw APIError.notFound }
        let lines = areas.flatMap { area in area.lowItems.map { "\($0.name) in \(area.name): \($0.statusLabel)" } }
        let spoken = SiriNames.summary(lines, empty: "No tracked items are low in stock.")
        return .result(value: lines, dialog: "\(spoken)")
    }
}

struct ChangeStockIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Update Stock" }
    nonisolated static var description: IntentDescription { "Set an item's count or percentage, or add or subtract a whole-item count." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }
    @Parameter(title: "Item", requestValueDialog: "Which item?") var item: InventoryItemEntity
    @Parameter(title: "Change", requestValueDialog: "Set count, add count, subtract count, or set percentage?") var operation: SiriStockOperation
    @Parameter(title: "Amount", requestValueDialog: "What whole number?") var amount: Int

    static var parameterSummary: some ParameterSummary { Summary("\(\.$operation) for \(\.$item) by \(\.$amount)") }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog {
        guard amount >= 0 else { throw SiriActionError.invalid("Supply a nonnegative whole number.") }
        let service = try SiriEnvironment.service()
        try await SiriEnvironment.validate(scope: item.scope, service: service)
        let change: InventoryStockChange
        switch operation {
        case .setCount: change = .quantity(amount)
        case .addCount: change = .quantityDelta(amount)
        case .subtractCount: change = .quantityDelta(-amount)
        case .setPercentage: change = .level(amount)
        }
        let saved = try await service.changeStock(id: item.rawID, change: change)
        SiriEnvironment.changed()
        return .result(dialog: "\(saved.name) is now \(saved.statusLabel).")
    }
}

struct CreateInventoryItemIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Add an Inventory Item" }
    nonisolated static var description: IntentDescription { "Create an item in an existing storage area with a count or percentage and low-stock threshold." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }
    @Parameter(title: "Name", requestValueDialog: "What is the item called?") var name: String
    @Parameter(title: "Area", requestValueDialog: "Which storage area?") var area: InventoryAreaEntity
    @Parameter(title: "Measurement", requestValueDialog: "Track a whole-item count or a percentage?") var tracking: SiriTrackingMode
    @Parameter(title: "Initial stock", requestValueDialog: "What is the initial count or percentage?") var stock: Int
    @Parameter(title: "Low-stock threshold", requestValueDialog: "At what count or percentage is this item low?") var lowAt: Int
    @Parameter(title: "Unit") var unit: String?

    static var parameterSummary: some ParameterSummary {
        Summary("Add \(\.$name) to \(\.$area)") {
            \.$tracking
            \.$stock
            \.$unit
            \.$lowAt
        }
    }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog {
        let service = try SiriEnvironment.service()
        try await SiriEnvironment.validate(scope: area.scope, service: service)
        var selectedUnit = ""
        if tracking == .count {
            if let unit { selectedUnit = unit }
            else { selectedUnit = try await $unit.requestValue("What unit, such as tins, bottles, or items?") }
        }
        let request = ItemRequest(name: name.trimmingCharacters(in: .whitespacesAndNewlines), trackingMode: tracking == .count ? .count : .level,
                                  quantity: tracking == .count ? stock : 0, level: tracking == .level ? stock : 100, unit: selectedUnit, lowAt: lowAt)
        let saved = try await service.createItem(areaID: area.rawID, request: request)
        SiriEnvironment.changed()
        return .result(dialog: "Added \(saved.name) to \(area.title) with \(saved.statusLabel).")
    }
}
