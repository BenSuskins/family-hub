import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class RecipesViewModelTests: XCTestCase {
    func testRepeatedTapDoesNotAddAnIngredientTwice() async throws {
        let client = FakeAPIClient()
        let model = RecipesViewModel(apiClient: client)
        let index = IngredientIndex(group: 0, item: 0)
        let started = expectation(description: "Shopping request starts")
        var finishAddition: CheckedContinuation<Void, Never>?
        client.beforeAddShopping = {
            await withCheckedContinuation { continuation in
                finishAddition = continuation
                started.fulfill()
            }
        }
        let adding = Task { await model.addIngredient("2 onions", at: index) }
        await fulfillment(of: [started], timeout: 2)

        await model.addIngredient("2 onions", at: index)
        finishAddition?.resume()
        await adding.value
        await model.addIngredient("2 onions", at: index)

        let items = try await client.fetchShopping()
        XCTAssertEqual(items.map(\.name), ["2 onions"])
    }

    func testMatchingIngredientTextInDifferentRowsHasSeparateActions() async throws {
        let client = FakeAPIClient()
        let model = RecipesViewModel(apiClient: client)
        let first = IngredientIndex(group: 0, item: 0)
        let second = IngredientIndex(group: 1, item: 0)

        await model.addIngredient("1 tbsp olive oil", at: first)

        XCTAssertEqual(model.ingredientShoppingState(at: first), .added)
        XCTAssertEqual(model.ingredientShoppingState(at: second), .ready)
        await model.addIngredient("1 tbsp olive oil", at: second)
        XCTAssertEqual(model.ingredientShoppingState(at: second), .added)
    }

    func testNewRecipeVisitResetsIngredientActions() async {
        let model = RecipesViewModel(apiClient: FakeAPIClient())
        let index = IngredientIndex(group: 0, item: 0)
        await model.addIngredient("2 onions", at: index)

        model.beginRecipeVisit()

        XCTAssertEqual(model.ingredientShoppingState(at: index), .ready)
    }

    func testPendingIngredientAdditionDoesNotChangeANewRecipeVisit() async {
        let client = FakeAPIClient()
        let model = RecipesViewModel(apiClient: client)
        let index = IngredientIndex(group: 0, item: 0)
        let started = expectation(description: "Shopping request starts")
        var finishAddition: CheckedContinuation<Void, Never>?
        client.beforeAddShopping = {
            await withCheckedContinuation { continuation in
                finishAddition = continuation
                started.fulfill()
            }
        }
        let adding = Task { await model.addIngredient("2 onions", at: index) }
        await fulfillment(of: [started], timeout: 2)

        XCTAssertEqual(model.ingredientShoppingState(at: index), .adding)
        model.beginRecipeVisit()
        finishAddition?.resume()
        await adding.value

        XCTAssertEqual(model.ingredientShoppingState(at: index), .ready)
    }

    func testFailedIngredientAdditionCanBeRetried() async throws {
        let client = FakeAPIClient()
        let model = RecipesViewModel(apiClient: client)
        let index = IngredientIndex(group: 0, item: 0)
        client.addShoppingError = APIError.offline

        await model.addIngredient("2 onions", at: index)

        XCTAssertEqual(model.ingredientShoppingState(at: index), .ready)
        XCTAssertNotNil(model.actionError)
        let failedItems = try await client.fetchShopping()
        XCTAssertTrue(failedItems.isEmpty)
        client.addShoppingError = nil

        await model.addIngredient("2 onions", at: index)

        XCTAssertEqual(model.ingredientShoppingState(at: index), .added)
        XCTAssertNil(model.actionError)
        let items = try await client.fetchShopping()
        XCTAssertEqual(items.map(\.name), ["2 onions"])
    }

    func testIngredientAddsOnlyItsFullTextToShopping() async throws {
        let client = FakeAPIClient()
        let model = RecipesViewModel(apiClient: client)
        model.beginRecipeVisit()
        let index = IngredientIndex(group: 0, item: 1)

        await model.addIngredient("200 g flour", at: index)

        let items = try await client.fetchShopping()
        XCTAssertEqual(items.map(\.name), ["200 g flour"])
        XCTAssertNil(items.first?.inventoryItemID)
        XCTAssertEqual(model.ingredientShoppingState(at: index), .added)
    }

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
        fake.recipesResult = .failure(APIError.server(status: 503, serverMessage: nil))
        let viewModel = RecipesViewModel(apiClient: fake)

        await viewModel.load()

        guard case .failed = viewModel.state else {
            XCTFail("expected failed state")
            return
        }
    }
}
