import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class ShoppingViewModel {
    var items: [ShoppingItem] = []
    var canEdit = false
    var isLoading = false
    var isChanging = false
    var capturedAt: Date?
    var error: APIError?
    private let apiClient: any APIClientProtocol
    private let cache: ShoppingCache
    private var userID: String?

    init(apiClient: any APIClientProtocol, cache: ShoppingCache = ShoppingCache()) {
        self.apiClient = apiClient
        self.cache = cache
    }

    func load() async {
        guard !isLoading, !isChanging else { return }
        isLoading = true
        defer { isLoading = false }
        do {
            if userID == nil { userID = try await apiClient.fetchMe().id }
            let loaded = try await apiClient.fetchShopping()
            guard !Task.isCancelled else { return }
            items = loaded
            capturedAt = Date()
            canEdit = true
            error = nil
            saveCache()
        } catch {
            guard !Task.isCancelled else { return }
            canEdit = false
            self.error = .from(error)
            if let server = apiClient.serverIdentifier,
               let identifier = userID ?? cache.lastUserID(server: server),
               let snapshot = cache.load(server: server, userID: identifier) {
                items = snapshot.items
                capturedAt = snapshot.capturedAt
            }
        }
    }

    func add(name: String) async {
        guard !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        _ = await change { try await self.apiClient.addShopping(name: name, inventoryItemID: nil) }
    }

    func toggle(_ item: ShoppingItem) async -> ShoppingItem? {
        await change { try await self.apiClient.updateShopping(id: item.id, change: ShoppingChange(purchased: !item.purchased, version: item.version)) }
    }

    func rename(_ item: ShoppingItem, name: String) async {
        _ = await change { try await self.apiClient.updateShopping(id: item.id, change: ShoppingChange(name: name, version: item.version)) }
    }

    func remove(_ item: ShoppingItem) async {
        await changeVoid { try await self.apiClient.deleteShopping(id: item.id) }
    }

    func clearPurchased() async {
        await changeVoid { try await self.apiClient.clearPurchasedShopping() }
    }

    func restock(_ item: ShoppingItem, quantity: Int?, level: Int?) async -> Bool {
        guard let purchaseID = item.purchaseID else { return false }
        return await change {
            try await self.apiClient.restockShopping(id: item.id, request: ShoppingRestock(purchaseID: purchaseID, quantity: quantity, level: level))
        } != nil
    }

    private func change(_ operation: () async throws -> ShoppingItem) async -> ShoppingItem? {
        guard canEdit, !isChanging else { return nil }
        isChanging = true
        do {
            let item = try await operation()
            items.removeAll { $0.id == item.id }
            items.append(item)
            error = nil
            saveCache()
            isChanging = false
            return item
        } catch {
            isChanging = false
            let failure = APIError.from(error)
            self.error = failure
            switch failure {
            case .conflict:
                await load()
                self.error = failure
            case .badRequest:
                break
            default:
                canEdit = false
            }
            return nil
        }
    }

    private func changeVoid(_ operation: () async throws -> Void) async {
        guard canEdit, !isChanging else { return }
        isChanging = true
        do {
            try await operation()
            isChanging = false
            await load()
        } catch {
            isChanging = false
            canEdit = false
            self.error = .from(error)
        }
    }

    private func saveCache() {
        capturedAt = Date()
        if let server = apiClient.serverIdentifier, let userID {
            cache.save(items, server: server, userID: userID)
        }
    }
}
