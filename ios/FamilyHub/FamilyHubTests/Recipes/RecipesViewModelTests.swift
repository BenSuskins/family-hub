import XCTest
@testable import FamilyHub

@MainActor
final class RecipesViewModelTests: XCTestCase {
    private func makeRecipe(id: String, title: String) -> Recipe {
        Recipe(id: id, title: title, steps: [], ingredients: [], mealType: nil, servings: nil, prepTime: nil, cookTime: nil, sourceURL: nil, categoryID: nil, hasImage: false)
    }

    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.recipesResult = .success([makeRecipe(id: "1", title: "Pasta")])
        let viewModel = RecipesViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let recipes) = viewModel.state else {
            XCTFail("expected loaded state")
            return
        }
        XCTAssertEqual(recipes.count, 1)
        XCTAssertEqual(recipes.first?.title, "Pasta")
    }

    func testLoadFailure() async {
        let fake = FakeAPIClient()
        fake.recipesResult = .failure(APIError.server(503))
        let viewModel = RecipesViewModel(apiClient: fake)

        await viewModel.load()

        guard case .failed = viewModel.state else {
            XCTFail("expected failed state")
            return
        }
    }
}
