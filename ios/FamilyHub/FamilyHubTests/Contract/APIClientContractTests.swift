import XCTest
@testable import FamilyHub

/// Behavioural contract every `APIClientProtocol` implementation must satisfy.
///
/// This is an *abstract* base: it owns the assertions but no client. Concrete
/// subclasses supply a client via `makeClient()`, and XCTest re-runs the whole
/// suite against each one. Running the same invariants against the
/// `DemoAPIClient` fake is what keeps the fake honest — it must behave like the
/// real backend, not just compile against the same protocol.
///
/// The base class skips itself (no client), so it contributes no failures.
@MainActor
class APIClientContractTests: XCTestCase {

    /// Override to provide the implementation under test. Throw `XCTSkip` to opt
    /// a suite out (the abstract base and the not-yet-wired live client do this).
    func makeClient() throws -> any APIClientProtocol {
        throw XCTSkip("Abstract contract base — no client to exercise.")
    }

    // MARK: - Chores

    func test_createChore_isReturnedByFetch_asPending() async throws {
        let client = try makeClient()
        let created = try await client.createChore(choreRequest(name: "Water the plants"))

        XCTAssertEqual(created.status, .pending)
        let all = try await client.fetchChores()
        XCTAssertTrue(all.contains { $0.id == created.id && $0.name == "Water the plants" })
    }

    func test_updateChore_reflectedByFetch() async throws {
        let client = try makeClient()
        let created = try await client.createChore(choreRequest(name: "Original"))

        _ = try await client.updateChore(id: created.id, choreRequest(name: "Renamed"))

        let all = try await client.fetchChores()
        XCTAssertEqual(all.first { $0.id == created.id }?.name, "Renamed")
    }

    func test_completeChore_marksItCompleted() async throws {
        let client = try makeClient()
        let created = try await client.createChore(choreRequest(name: "Take out bins"))

        try await client.completeChore(id: created.id)

        let all = try await client.fetchChores()
        XCTAssertEqual(all.first { $0.id == created.id }?.status, .completed)
    }

    func test_deleteChore_removesItFromFetch() async throws {
        let client = try makeClient()
        let created = try await client.createChore(choreRequest(name: "Temp"))

        try await client.deleteChore(id: created.id)

        let all = try await client.fetchChores()
        XCTAssertFalse(all.contains { $0.id == created.id })
    }

    // MARK: - Recipes

    func test_createRecipe_isFetchableByListAndID() async throws {
        let client = try makeClient()
        let created = try await client.createRecipe(recipeRequest(title: "Soup"))

        let list = try await client.fetchRecipes(forceRefresh: true)
        XCTAssertTrue(list.contains { $0.id == created.id })

        let detail = try await client.fetchRecipe(id: created.id)
        XCTAssertEqual(detail.title, "Soup")
    }

    func test_updateRecipe_reflectedByFetch() async throws {
        let client = try makeClient()
        let created = try await client.createRecipe(recipeRequest(title: "Before"))

        _ = try await client.updateRecipe(id: created.id, recipeRequest(title: "After"))

        let detail = try await client.fetchRecipe(id: created.id)
        XCTAssertEqual(detail.title, "After")
    }

    func test_deleteRecipe_removesItAndDetailFails() async throws {
        let client = try makeClient()
        let created = try await client.createRecipe(recipeRequest(title: "Doomed"))

        try await client.deleteRecipe(id: created.id)

        let list = try await client.fetchRecipes(forceRefresh: true)
        XCTAssertFalse(list.contains { $0.id == created.id })
        await assertThrows { _ = try await client.fetchRecipe(id: created.id) }
    }

    // MARK: - Meals

    func test_saveMeal_isReturnedByFetch_andReplacesSameSlot() async throws {
        let client = try makeClient()
        let date = "2099-01-01"

        _ = try await client.saveMeal(date: date, mealType: "dinner", name: "Pasta", recipeID: nil)
        _ = try await client.saveMeal(date: date, mealType: "dinner", name: "Pizza", recipeID: nil)

        let meals = try await client.fetchMeals(week: Date())
        let slot = meals.filter { $0.date == date && $0.mealType == "dinner" }
        XCTAssertEqual(slot.count, 1, "Saving the same slot twice must replace, not duplicate")
        XCTAssertEqual(slot.first?.name, "Pizza")
    }

    func test_deleteMeal_removesIt() async throws {
        let client = try makeClient()
        let date = "2099-02-02"
        _ = try await client.saveMeal(date: date, mealType: "lunch", name: "Salad", recipeID: nil)

        try await client.deleteMeal(date: date, mealType: "lunch")

        let meals = try await client.fetchMeals(week: Date())
        XCTAssertFalse(meals.contains { $0.date == date && $0.mealType == "lunch" })
    }

    // MARK: - Categories

    func test_categoryLifecycle() async throws {
        let client = try makeClient()
        let created = try await client.createCategory(name: "Snacks")
        var all = try await client.fetchCategories()
        XCTAssertTrue(all.contains { $0.id == created.id && $0.name == "Snacks" })

        let updated = try await client.updateCategory(id: created.id, name: "Treats")
        XCTAssertEqual(updated.name, "Treats")
        all = try await client.fetchCategories()
        XCTAssertEqual(all.first { $0.id == created.id }?.name, "Treats")

        try await client.deleteCategory(id: created.id)
        all = try await client.fetchCategories()
        XCTAssertFalse(all.contains { $0.id == created.id })
    }

    // MARK: - Tokens

    func test_tokenLifecycle() async throws {
        let client = try makeClient()
        let created = try await client.createToken(name: "iPad")
        XCTAssertFalse(created.plaintext.isEmpty, "A freshly created token must expose its plaintext once")

        var all = try await client.fetchTokens()
        XCTAssertTrue(all.contains { $0.id == created.id && $0.name == "iPad" })

        try await client.deleteToken(id: created.id)
        all = try await client.fetchTokens()
        XCTAssertFalse(all.contains { $0.id == created.id })
    }

    // MARK: - User roles

    func test_promoteThenDemote_returnsUpdatedRole() async throws {
        let client = try makeClient()
        guard let user = try await client.fetchUsers().first else {
            throw XCTSkip("Implementation has no seed users to promote/demote")
        }

        let promoted = try await client.promoteUser(id: user.id)
        XCTAssertEqual(promoted.role, "admin")

        let demoted = try await client.demoteUser(id: user.id)
        XCTAssertEqual(demoted.role, "member")
    }

    // MARK: - Fixtures

    private func choreRequest(name: String) -> ChoreRequest {
        ChoreRequest(name: name, description: "", assignees: [])
    }

    private func recipeRequest(title: String) -> RecipeRequest {
        RecipeRequest(
            title: title,
            steps: ["Step one"],
            ingredients: [IngredientGroup(name: "Main", items: ["Item"])],
            mealType: "dinner",
            servings: 2,
            prepTime: "5 min",
            cookTime: "10 min",
            sourceURL: nil,
            imageData: nil
        )
    }

    private func assertThrows(
        _ operation: () async throws -> Void,
        file: StaticString = #filePath,
        line: UInt = #line
    ) async {
        do {
            try await operation()
            XCTFail("Expected an error to be thrown", file: file, line: line)
        } catch {
            // Expected.
        }
    }
}

/// Runs the contract against the in-app fake used for demo mode and SwiftUI
/// previews, proving it stays behaviourally faithful to the real backend.
final class DemoAPIClientContractTests: APIClientContractTests {
    override func makeClient() throws -> any APIClientProtocol {
        DemoAPIClient()
    }
}

/// Seam for running the same contract against the real `APIClient`.
///
/// TODO: provide an in-memory `URLProtocol` stub server that emulates the REST
/// endpoints (capitalised JSON keys per the models' `CodingKeys`) plus a fake
/// `KeychainStore` returning a token, then return
/// `APIClient(baseURL:session:authManager:)` here. Until then this suite skips,
/// keeping the wiring point visible without standing up the stub server.
final class LiveAPIClientContractTests: APIClientContractTests {
    override func makeClient() throws -> any APIClientProtocol {
        throw XCTSkip("Pending: APIClient over a URLProtocol stub server — see plans/ios-architecture-cleanup.md")
    }
}
