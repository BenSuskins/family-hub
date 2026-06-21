import Foundation
import Observation

@Observable
@MainActor
final class InventoryViewModel: MutableListViewModel {
    var state: ViewState<[InventoryArea]> = .idle
    var actionError: APIError?

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    // MARK: - Derived

    var areas: [InventoryArea] {
        guard case .loaded(let areas) = state else { return [] }
        return areas
    }

    /// Every low item across all areas, each carrying its area for display and
    /// navigation — backs the home screen's "Running low" rollup.
    var runningLow: [RunningLowItem] {
        areas.flatMap { area in
            area.lowItems.map { RunningLowItem(area: area, item: $0) }
        }
    }

    var totalRunningLow: Int { runningLow.count }

    /// The current snapshot of an area by id (so detail screens reflect edits).
    func area(id: String) -> InventoryArea? {
        areas.first { $0.id == id }
    }

    // MARK: - Loading

    func load() async {
        state = .loading
        do {
            let areas = try await apiClient.fetchInventory()
            state = .loaded(areas)
        } catch {
            state = .failed(.from(error))
        }
    }

    // MARK: - Area mutations

    func createArea(_ request: AreaRequest) async -> InventoryArea? {
        await performMutation { try await apiClient.createArea(request) } applying: { created, areas in
            areas.append(created)
            areas.sort { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }
    }

    func updateArea(id: String, _ request: AreaRequest) async -> InventoryArea? {
        await performMutation { try await apiClient.updateArea(id: id, request) } applying: { updated, areas in
            if let index = areas.firstIndex(where: { $0.id == id }) {
                areas[index] = updated
            }
            areas.sort { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }
    }

    func deleteArea(id: String) async -> Bool {
        await performDeletion { try await apiClient.deleteArea(id: id) } applying: { areas in
            areas.removeAll { $0.id == id }
        }
    }

    // MARK: - Item mutations

    func createItem(areaID: String, _ request: ItemRequest) async -> InventoryItem? {
        do {
            let created = try await apiClient.createItem(areaID: areaID, request)
            mutateLoaded { Self.upsertItem(created, in: areaID, areas: &$0) }
            return created
        } catch {
            actionError = .from(error)
            return nil
        }
    }

    func updateItem(areaID: String, id: String, _ request: ItemRequest) async -> InventoryItem? {
        do {
            let updated = try await apiClient.updateItem(id: id, request)
            mutateLoaded { Self.upsertItem(updated, in: areaID, areas: &$0) }
            return updated
        } catch {
            actionError = .from(error)
            return nil
        }
    }

    func deleteItem(areaID: String, id: String) async -> Bool {
        await performDeletion { try await apiClient.deleteItem(id: id) } applying: { areas in
            guard let index = areas.firstIndex(where: { $0.id == areaID }) else { return }
            let items = areas[index].items.filter { $0.id != id }
            areas[index] = Self.replacingItems(areas[index], items)
        }
    }

    /// Step an item's quantity by `delta`. Applies the change locally first so the
    /// stepper feels instant, then persists; on failure the local value is
    /// reverted and `actionError` surfaced.
    func adjustQuantity(item: InventoryItem, by delta: Int) async {
        let target = item.adjusting(by: delta)
        if target.quantity == item.quantity { return } // already clamped at 0

        let previous = item
        mutateLoaded { Self.upsertItem(target, in: target.areaID, areas: &$0) }

        do {
            let request = ItemRequest(name: target.name, quantity: target.quantity, unit: target.unit, par: target.par)
            let saved = try await apiClient.updateItem(id: target.id, request)
            mutateLoaded { Self.upsertItem(saved, in: saved.areaID, areas: &$0) }
        } catch {
            mutateLoaded { Self.upsertItem(previous, in: previous.areaID, areas: &$0) }
            actionError = .from(error)
        }
    }

    // MARK: - Helpers

    private static func upsertItem(_ item: InventoryItem, in areaID: String, areas: inout [InventoryArea]) {
        guard let index = areas.firstIndex(where: { $0.id == areaID }) else { return }
        var items = areas[index].items
        if let i = items.firstIndex(where: { $0.id == item.id }) {
            items[i] = item
        } else {
            items.append(item)
            items.sort { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }
        areas[index] = replacingItems(areas[index], items)
    }

    private static func replacingItems(_ area: InventoryArea, _ items: [InventoryItem]) -> InventoryArea {
        InventoryArea(id: area.id, name: area.name, icon: area.icon, tint: area.tint, items: items)
    }
}
