import AppIntents
import Foundation
import FamilyHubKit

struct AddShoppingIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Add to Shopping List" }
    nonisolated static var description: IntentDescription { "Add an item to the household shopping list, linking a matching inventory item when available." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }
    @Parameter(title: "Item name", requestValueDialog: "What should I add to the shopping list?") var name: String
    @Parameter(title: "Inventory item") var inventoryItem: InventoryItemEntity?

    static var parameterSummary: some ParameterSummary { Summary("Add \(\.$name) to the shopping list") }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog {
        let service = try SiriEnvironment.service()
        let areas = try await service.inventory()
        let scope = try await SiriEnvironment.scope(service)
        let matches = areas.flatMap { area in
            area.items.filter { SiriNames.normalized($0.name) == SiriNames.normalized(name) }
                .map { InventoryItemEntity(rawID: $0.id, scope: scope, title: $0.name, detail: area.name) }
        }
        var identifier: String?
        if let inventoryItem {
            try await SiriEnvironment.validate(scope: inventoryItem.scope, service: service)
            guard matches.contains(where: { $0.id == inventoryItem.id }) else { throw SiriActionError.invalid("Select an inventory item that matches the name.") }
            identifier = inventoryItem.rawID
        } else if matches.count == 1 { identifier = matches.first?.rawID }
        else if matches.count > 1 { identifier = try await $inventoryItem.requestDisambiguation(among: matches, dialog: "Which storage area?").rawID }
        _ = try await service.addShopping(name: name, inventoryItemID: identifier)
        SiriEnvironment.changed()
        return .result(dialog: "Added \(name) to the shopping list.")
    }
}
