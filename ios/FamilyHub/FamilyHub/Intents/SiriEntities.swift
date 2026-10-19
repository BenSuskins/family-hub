import AppIntents
import Foundation
import FamilyHubKit

nonisolated struct ChoreEntity: AppEntity {
    let rawID: String
    let scope: String
    let title: String
    let detail: String
    var id: String { scope + "|" + rawID }
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Chore" }
    static var defaultQuery: ChoreQuery { ChoreQuery() }
    var displayRepresentation: DisplayRepresentation {
        DisplayRepresentation(title: "\(title)", subtitle: "\(detail)")
    }
}

struct ChoreQuery: EntityStringQuery {
    nonisolated init() {}

    @MainActor
    func entities(for identifiers: [String]) async throws -> [ChoreEntity] {
        try await load().filter { identifiers.contains($0.id) }
    }

    @MainActor
    func entities(matching string: String) async throws -> [ChoreEntity] {
        let matches = try await load().filter { SiriNames.normalized($0.title).contains(SiriNames.normalized(string)) }
        return matches
    }

    @MainActor
    func suggestedEntities() async throws -> [ChoreEntity] {
        guard SharedContainer.isConfigured else { return [] }
        return Array(try await load().prefix(10))
    }

    @MainActor
    private func load() async throws -> [ChoreEntity] {
        let service = try SiriEnvironment.service()
        let scope = try await SiriEnvironment.scope(service)
        let chores = try await service.myChores()
        return chores.map { ChoreEntity(rawID: $0.id, scope: scope, title: $0.name, detail: $0.dueDate ?? "") }
    }
}

nonisolated struct RecipeEntity: AppEntity {
    let rawID: String
    let scope: String
    let title: String
    let detail: String
    var id: String { scope + "|" + rawID }
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Recipe" }
    static var defaultQuery: RecipeQuery { RecipeQuery() }
    var displayRepresentation: DisplayRepresentation {
        DisplayRepresentation(title: "\(title)", subtitle: "\(detail)")
    }
}

struct RecipeQuery: EntityStringQuery {
    nonisolated init() {}

    @MainActor
    func entities(for identifiers: [String]) async throws -> [RecipeEntity] {
        try await load().filter { identifiers.contains($0.id) }
    }

    @MainActor
    func entities(matching string: String) async throws -> [RecipeEntity] {
        let matches = try await load().filter { SiriNames.normalized($0.title).contains(SiriNames.normalized(string)) }
        return matches
    }

    @MainActor
    func suggestedEntities() async throws -> [RecipeEntity] {
        guard SharedContainer.isConfigured else { return [] }
        return Array(try await load().prefix(10))
    }

    @MainActor
    private func load() async throws -> [RecipeEntity] {
        let service = try SiriEnvironment.service()
        let scope = try await SiriEnvironment.scope(service)
        let recipes = try await service.client.fetchRecipes(forceRefresh: true)
        return recipes.map { RecipeEntity(rawID: $0.id, scope: scope, title: $0.title, detail: "") }
    }
}

nonisolated struct InventoryItemEntity: AppEntity {
    let rawID: String
    let scope: String
    let title: String
    let detail: String
    var id: String { scope + "|" + rawID }
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Inventory item" }
    static var defaultQuery: InventoryItemQuery { InventoryItemQuery() }
    var displayRepresentation: DisplayRepresentation {
        DisplayRepresentation(title: "\(title)", subtitle: "\(detail)")
    }
}

struct InventoryItemQuery: EntityStringQuery {
    nonisolated init() {}

    @MainActor
    func entities(for identifiers: [String]) async throws -> [InventoryItemEntity] {
        try await load().filter { identifiers.contains($0.id) }
    }

    @MainActor
    func entities(matching string: String) async throws -> [InventoryItemEntity] {
        let matches = try await load().filter { SiriNames.normalized($0.title).contains(SiriNames.normalized(string)) }
        guard !matches.isEmpty else { throw SiriActionError.invalid("That item is not tracked in Family Hub.") }
        return matches
    }

    @MainActor
    func suggestedEntities() async throws -> [InventoryItemEntity] {
        guard SharedContainer.isConfigured else { return [] }
        return Array(try await load().prefix(10))
    }

    @MainActor
    private func load() async throws -> [InventoryItemEntity] {
        let service = try SiriEnvironment.service()
        let scope = try await SiriEnvironment.scope(service)
        let areas = try await service.inventory()
        return areas.flatMap { area in area.items.map { InventoryItemEntity(rawID: $0.id, scope: scope, title: $0.name, detail: area.name) } }
    }
}

nonisolated struct InventoryAreaEntity: AppEntity {
    let rawID: String
    let scope: String
    let title: String
    let detail: String
    var id: String { scope + "|" + rawID }
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Storage area" }
    static var defaultQuery: InventoryAreaQuery { InventoryAreaQuery() }
    var displayRepresentation: DisplayRepresentation {
        DisplayRepresentation(title: "\(title)", subtitle: "\(detail)")
    }
}

struct InventoryAreaQuery: EntityStringQuery {
    nonisolated init() {}

    @MainActor
    func entities(for identifiers: [String]) async throws -> [InventoryAreaEntity] {
        try await load().filter { identifiers.contains($0.id) }
    }

    @MainActor
    func entities(matching string: String) async throws -> [InventoryAreaEntity] {
        let matches = try await load().filter { SiriNames.normalized($0.title).contains(SiriNames.normalized(string)) }
        return matches
    }

    @MainActor
    func suggestedEntities() async throws -> [InventoryAreaEntity] {
        guard SharedContainer.isConfigured else { return [] }
        return Array(try await load().prefix(10))
    }

    @MainActor
    private func load() async throws -> [InventoryAreaEntity] {
        let service = try SiriEnvironment.service()
        let scope = try await SiriEnvironment.scope(service)
        let areas = try await service.inventory()
        return areas.map { InventoryAreaEntity(rawID: $0.id, scope: scope, title: $0.name, detail: "") }
    }
}
