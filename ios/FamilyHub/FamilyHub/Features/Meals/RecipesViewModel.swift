// ios/FamilyHub/Features/Recipes/RecipesViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class RecipesViewModel {
    var state: ViewState<[Recipe]> = .idle
    var searchQuery: String = ""

    var filteredRecipes: [Recipe] {
        guard case .loaded(let recipes) = state else { return [] }
        guard !searchQuery.isEmpty else { return recipes }
        return recipes.filter { $0.title.localizedCaseInsensitiveContains(searchQuery) }
    }

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        do {
            let recipes = try await apiClient.fetchRecipes()
            state = .loaded(recipes)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }
}
