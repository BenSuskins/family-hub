import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class ChoresViewModel {
    var state: ViewState<[ChoreDefinition]> = .idle
    var actionError: APIError?
    var users: [String: User] = [:]
    var categories: [ChoreCategory] = []
    var showInactive = false
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func definition(id: String) -> ChoreDefinition? {
        guard case .loaded(let definitions) = state else { return nil }
        return definitions.first { $0.id == id }
    }

    func categoryName(_ id: String?) -> String? {
        categories.first { $0.id == id }?.name
    }

    func load(refresh: Bool = false) async {
        if !refresh { state = .loading }
        async let definitions = apiClient.fetchChoreDefinitions(includeInactive: showInactive)
        async let people = apiClient.fetchUsers()
        async let categoryList = apiClient.fetchCategories()
        do {
            let (loaded, userList) = try await (definitions, people)
            users = userList.keyedByID
            categories = (try? await categoryList) ?? []
            state = .loaded(loaded)
        } catch {
            if refresh { actionError = .from(error) }
            else { state = .failed(.from(error)) }
        }
    }

    func createChore(_ request: ChoreRequest) async -> Chore? {
        await save { try await self.apiClient.createChoreDefinition(request) }
    }

    func updateChore(id: String, _ request: ChoreRequest) async -> Chore? {
        await save { try await self.apiClient.updateChoreDefinition(id: id, request) }
    }

    private func save(_ operation: () async throws -> ChoreDefinition) async -> Chore? {
        actionError = nil
        do {
            let result = try await operation()
            await changed()
            return result.chore
        } catch {
            actionError = .from(error)
            return nil
        }
    }

    func deleteChore(id: String) async -> Bool {
        actionError = nil
        do {
            try await apiClient.deleteChoreDefinition(id: id)
            await changed()
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }

    private func changed() async {
        WidgetPublisher.invalidate()
        await load(refresh: true)
    }
}
