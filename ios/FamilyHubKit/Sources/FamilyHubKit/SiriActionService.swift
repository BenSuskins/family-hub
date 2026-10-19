import Foundation

public struct SiriChoreCompletion {
    public let completed: [String]
    public let failed: [String]
    public let uncertain: [String]
}

public final class SiriActionService {
    public let client: any APIClientProtocol
    private let now: () -> Date
    private let retainedTokenProvider: (any TokenProviding)?

    public init(client: any APIClientProtocol, now: @escaping () -> Date = Date.init, tokenProvider: (any TokenProviding)? = nil) {
        self.client = client
        self.now = now
        self.retainedTokenProvider = tokenProvider
    }

    public func day(_ components: DateComponents?) async throws -> SiriDay {
        let home = try await client.fetchHomeInfo()
        return try SiriDateRules(timezone: home.timezone).day(components, now: now())
    }

    public func meal(type: String, day: SiriDay) async throws -> MealPlan? {
        try await client.fetchMeals(week: day.queryDate).first { $0.date == day.key && $0.mealType == type }
    }

    public func recipes(named name: String) async throws -> [Recipe] {
        SiriNames.recipes(named: name, in: try await client.fetchRecipes(forceRefresh: true))
    }

    public func saveMeal(type: String, name: String, day: SiriDay, recipeID: String?) async throws -> MealPlan {
        guard ["breakfast", "lunch", "dinner"].contains(type), !SiriNames.normalized(name).isEmpty else {
            throw SiriActionError.invalid("Supply a meal name and breakfast, lunch, or dinner.")
        }
        return try await write { try await self.client.saveMeal(date: day.key, mealType: type, name: name.trimmingCharacters(in: .whitespacesAndNewlines), recipeID: recipeID) }
    }

    public func myChores() async throws -> [Chore] {
        async let user = client.fetchMe()
        async let chores = client.fetchChores()
        async let home = client.fetchHomeInfo()
        return try await SiriDateRules(timezone: home.timezone).eligibleChores(chores, userID: user.id, now: now())
    }

    public func completeChore(id: String) async throws {
        guard try await myChores().contains(where: { $0.id == id }) else {
            throw SiriActionError.invalid("That chore is no longer in your due or overdue chores.")
        }
        do {
            try await write { try await self.client.completeChore(id: id) }
        } catch {
            if (try? await client.fetchChores().first(where: { $0.id == id })?.status) == .completed { return }
            throw error
        }
    }

    public func completeMyChores() async throws -> SiriChoreCompletion {
        let chores = try await myChores()
        var completed: [String] = []
        var failed: [String] = []
        var uncertain: [String] = []
        for chore in chores {
            do {
                try await completeChore(id: chore.id)
                completed.append(chore.name)
            } catch SiriActionError.uncertain {
                uncertain.append(chore.name)
            } catch {
                failed.append(chore.name)
            }
        }
        return SiriChoreCompletion(completed: completed, failed: failed, uncertain: uncertain)
    }

    public func inventory() async throws -> [InventoryArea] {
        try await client.fetchInventory()
    }

    public func changeStock(id: String, change: InventoryStockChange) async throws -> InventoryItem {
        guard let item = try await inventory().flatMap(\.items).first(where: { $0.id == id }) else { throw APIError.notFound }
        _ = try change.applying(to: item)
        do {
            return try await write { try await self.client.changeStock(id: id, change: change) }
        } catch SiriActionError.uncertain {
            guard let saved = try? await inventory().flatMap(\.items).first(where: { $0.id == id }) else {
                throw SiriActionError.uncertain
            }
            switch change {
            case .quantity(let value) where saved.trackingMode == .count && saved.quantity == value: return saved
            case .level(let value) where saved.trackingMode == .level && saved.level == value: return saved
            default: throw SiriActionError.uncertain
            }
        }
    }

    public func createItem(areaID: String, request: ItemRequest) async throws -> InventoryItem {
        let areas = try await inventory()
        guard let area = areas.first(where: { $0.id == areaID }) else { throw APIError.notFound }
        guard !SiriNames.normalized(request.name).isEmpty else { throw SiriActionError.invalid("Supply an item name.") }
        guard !area.items.contains(where: { SiriNames.normalized($0.name) == SiriNames.normalized(request.name) }) else {
            throw SiriActionError.invalid("That item already exists in this area. Use Update Stock in Family Hub.")
        }
        guard request.quantity >= 0, request.lowAt >= 0,
              (0...100).contains(request.level), request.trackingMode != .level || request.lowAt <= 100 else {
            throw SiriActionError.invalid("Supply nonnegative counts and percentages from 0 to 100.")
        }
        return try await write { try await self.client.createItem(areaID: areaID, request) }
    }

    public func addShopping(name: String, inventoryItemID: String?) async throws -> ShoppingItem {
        guard !SiriNames.normalized(name).isEmpty else { throw SiriActionError.invalid("Supply an item name.") }
        return try await write { try await self.client.addShopping(name: name, inventoryItemID: inventoryItemID) }
    }

    private func write<Value>(_ operation: () async throws -> Value) async throws -> Value {
        do { return try await operation() }
        catch {
            switch APIError.from(error) {
            case .offline, .timedOut, .network, .decoding, .server: throw SiriActionError.uncertain
            default: throw error
            }
        }
    }
}
