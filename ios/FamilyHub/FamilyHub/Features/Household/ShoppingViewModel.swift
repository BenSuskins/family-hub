import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class ShoppingViewModel {
    var items: [ShoppingItem] = []
    var canEdit = false
    var isLoading = false
    var capturedAt: Date?
    var error: APIError?
    private(set) var rowErrors: [String: APIError] = [:]
    private(set) var changingIDs: Set<String> = []
    private var purchaseTargets: [String: Bool] = [:]
    private var purchaseDeadlines: [String: Date] = [:]
    private var failedPurchaseIDs: Set<String> = []
    @ObservationIgnored private var purchaseTasks: [String: Task<ShoppingItem?, Never>] = [:]
    private let apiClient: any APIClientProtocol
    private let cache: ShoppingCache
    private let now: () -> Date
    private var userID: String?
    private var revision = 0

    var isChanging: Bool { !changingIDs.isEmpty }
    var visibleItems: [ShoppingItem] {
        items.filter { !$0.purchased || purchaseDeadlines[$0.id] != nil || purchaseTargets[$0.id] != nil || failedPurchaseIDs.contains($0.id) }
    }
    var remainingCount: Int { items.filter { !isPurchased($0) }.count }

    init(apiClient: any APIClientProtocol, cache: ShoppingCache? = nil, now: @escaping () -> Date = Date.init) {
        self.apiClient = apiClient
        self.cache = cache ?? ShoppingCache()
        self.now = now
    }

    func isPurchased(_ item: ShoppingItem) -> Bool { purchaseTargets[item.id] ?? item.purchased }

    func remainingUndoSeconds(for item: ShoppingItem) -> Int? {
        guard isPurchased(item), let deadline = purchaseDeadlines[item.id] else { return nil }
        return max(0, Int(ceil(deadline.timeIntervalSince(now()))))
    }

    func expirePurchases() {
        let expired = purchaseDeadlines.filter { $0.value <= now() && !changingIDs.contains($0.key) && purchaseTargets[$0.key] == nil }
        for identifier in expired.keys { purchaseDeadlines.removeValue(forKey: identifier) }
    }

    func load() async {
        guard !isLoading, !isChanging else { return }
        isLoading = true
        let startingRevision = revision
        defer { isLoading = false }
        do {
            if userID == nil { userID = try await apiClient.fetchMe().id }
            let loaded = try await apiClient.fetchShopping()
            guard !Task.isCancelled else { return }
            guard revision == startingRevision, !isChanging else { return }
            items = loaded
            canEdit = true
            error = nil
            expirePurchases()
            saveCache()
        } catch {
            guard !Task.isCancelled else { return }
            canEdit = false
            self.error = .from(error)
            if items.isEmpty, let server = apiClient.serverIdentifier,
               let identifier = userID ?? cache.lastUserID(server: server),
               let snapshot = cache.load(server: server, userID: identifier) {
                items = snapshot.items
                capturedAt = snapshot.capturedAt
            }
        }
    }

    func add(name: String) async {
        let name = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty else { return }
        _ = await change(id: "new-item") { try await self.apiClient.addShopping(name: name, inventoryItemID: nil) }
    }

    func addInventoryItem(_ item: InventoryItem) async {
        _ = await change(id: "new-" + item.id) { try await self.apiClient.addShopping(name: item.name, inventoryItemID: item.id) }
    }

    func toggle(_ item: ShoppingItem) async -> ShoppingItem? {
        guard canEdit, !changingIDs.contains(item.id) || purchaseTasks[item.id] != nil,
              let current = items.first(where: { $0.id == item.id }) else { return nil }
        let target = !isPurchased(current)
        purchaseTargets[item.id] = target
        rowErrors[item.id] = nil
        failedPurchaseIDs.remove(item.id)
        if target { purchaseDeadlines[item.id] = now().addingTimeInterval(5) }
        else { purchaseDeadlines[item.id] = nil }
        if let task = purchaseTasks[item.id] { return await task.value }
        let task = Task { await applyPurchase(id: item.id) }
        purchaseTasks[item.id] = task
        return await task.value
    }

    private func applyPurchase(id: String) async -> ShoppingItem? {
        changingIDs.insert(id)
        defer { purchaseTasks[id] = nil }
        var result: ShoppingItem?
        while let target = purchaseTargets[id], let current = items.first(where: { $0.id == id }) {
            if current.purchased == target {
                purchaseTargets[id] = nil
                result = current
                break
            }
            do {
                let updated = try await apiClient.updateShopping(id: id, change: ShoppingChange(purchased: target, version: current.version))
                replace(updated)
                saveCache()
                result = updated
                if purchaseTargets[id] == target { purchaseTargets[id] = nil }
            } catch {
                let failure = APIError.from(error)
                purchaseTargets[id] = nil
                purchaseDeadlines[id] = nil
                failedPurchaseIDs.insert(id)
                rowErrors[id] = failure
                changingIDs.remove(id)
                // Reconcile uncertain writes before allowing another edit; never replay a purchase blindly.
                if case .badRequest = failure { } else {
                    canEdit = false
                    await load()
                }
                rowErrors[id] = failure
                return nil
            }
        }
        changingIDs.remove(id)
        expirePurchases()
        return result
    }

    @discardableResult
    func rename(_ item: ShoppingItem, name: String) async -> Bool {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, trimmed.utf8.count <= 200 else {
            rowErrors[item.id] = .badRequest(serverMessage: trimmed.isEmpty ? "Enter an item name." : "Use a shorter item name.")
            return false
        }
        guard let current = items.first(where: { $0.id == item.id }) else { return false }
        return await change(id: item.id) {
            try await self.apiClient.updateShopping(id: item.id, change: ShoppingChange(name: trimmed, version: current.version))
        } != nil
    }

    func remove(_ item: ShoppingItem) async {
        guard canEdit, !changingIDs.contains(item.id) else { return }
        changingIDs.insert(item.id)
        defer { changingIDs.remove(item.id) }
        do {
            try await apiClient.deleteShopping(id: item.id)
            revision += 1
            items.removeAll { $0.id == item.id }
            purchaseDeadlines[item.id] = nil
            rowErrors[item.id] = nil
            saveCache()
        } catch { rowErrors[item.id] = .from(error) }
    }

    func restock(_ item: ShoppingItem, quantity: Int?, level: Int?) async -> Bool {
        guard let purchaseID = item.purchaseID else { return false }
        return await change(id: item.id) {
            try await self.apiClient.restockShopping(id: item.id, request: ShoppingRestock(purchaseID: purchaseID, quantity: quantity, level: level))
        } != nil
    }

    private func change(id: String, _ operation: () async throws -> ShoppingItem) async -> ShoppingItem? {
        guard canEdit, !changingIDs.contains(id) else { return nil }
        changingIDs.insert(id)
        defer { changingIDs.remove(id) }
        do {
            let item = try await operation()
            replace(item)
            rowErrors[id] = nil
            error = nil
            saveCache()
            return item
        } catch {
            let failure = APIError.from(error)
            rowErrors[id] = failure
            if id.hasPrefix("new-") { self.error = failure }
            switch failure {
            case .badRequest: break
            default: canEdit = false
            }
            return nil
        }
    }

    private func replace(_ item: ShoppingItem) {
        revision += 1
        if let index = items.firstIndex(where: { $0.id == item.id }) { items[index] = item }
        else { items.append(item) }
    }

    private func saveCache() {
        capturedAt = now()
        if let server = apiClient.serverIdentifier, let userID {
            cache.save(items, server: server, userID: userID)
        }
    }
}
