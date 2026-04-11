// ios/FamilyHub/Features/Recipes/RecipesViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class RecipesViewModel {
    var state: ViewState<[Recipe]> = .idle
    var searchQuery: String = ""
    var selectedMealType: String? = nil
    var errorMessage: String?
    var recipeImages: [String: Data] = [:]

    static let mealTypeOptions: [String] = ["breakfast", "lunch", "dinner", "side", "dessert"]

    var filteredRecipes: [Recipe] {
        guard case .loaded(let recipes) = state else { return [] }
        var result = recipes
        if let mealType = selectedMealType {
            result = result.filter { $0.mealType == mealType }
        }
        if !searchQuery.isEmpty {
            result = result.filter { $0.title.localizedCaseInsensitiveContains(searchQuery) }
        }
        return result
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
            await preloadImages(for: recipes)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }

    func preloadImages(for recipes: [Recipe]) async {
        let missing = recipes.filter { $0.hasImage && recipeImages[$0.id] == nil }
        guard !missing.isEmpty else { return }
        let client = apiClient
        let fetched: [(String, Data)] = await withTaskGroup(of: (String, Data)?.self) { group in
            for recipe in missing {
                group.addTask {
                    guard let data = try? await client.fetchRecipeImage(id: recipe.id) else { return nil }
                    return (recipe.id, data)
                }
            }
            var results: [(String, Data)] = []
            for await case let item? in group {
                results.append(item)
            }
            return results
        }
        for (id, data) in fetched {
            recipeImages[id] = data
        }
    }

    func createRecipe(_ request: RecipeRequest) async -> Recipe? {
        do {
            let created = try await apiClient.createRecipe(request)
            if case .loaded(var recipes) = state {
                recipes.append(created)
                state = .loaded(recipes)
            }
            return created
        } catch {
            errorMessage = "Failed to create recipe"
            return nil
        }
    }

    func updateRecipe(id: String, _ request: RecipeRequest) async -> Recipe? {
        do {
            let updated = try await apiClient.updateRecipe(id: id, request)
            if case .loaded(var recipes) = state {
                if let index = recipes.firstIndex(where: { $0.id == id }) {
                    recipes[index] = updated
                }
                state = .loaded(recipes)
            }
            return updated
        } catch {
            errorMessage = "Failed to update recipe"
            return nil
        }
    }

    func deleteRecipe(id: String) async -> Bool {
        do {
            try await apiClient.deleteRecipe(id: id)
            if case .loaded(var recipes) = state {
                recipes.removeAll { $0.id == id }
                state = .loaded(recipes)
            }
            return true
        } catch {
            errorMessage = "Failed to delete recipe"
            return false
        }
    }
}
