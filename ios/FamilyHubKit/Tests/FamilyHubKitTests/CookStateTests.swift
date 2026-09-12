import XCTest
@testable import FamilyHubKit

final class CookStateTests: XCTestCase {

    private func makeRecipe(
        steps: [String]? = ["Chop the onions", "Simmer for 20 minutes", "Serve"],
        ingredients: [IngredientGroup]? = [
            IngredientGroup(name: "Main", items: ["2 onions", "1 tbsp olive oil"]),
            IngredientGroup(name: "To serve", items: ["1 tbsp olive oil"]),
        ],
        stepDurations: [Int?]? = [nil, 1200, nil]
    ) -> Recipe {
        Recipe(id: "r1", title: "Soup", steps: steps, ingredients: ingredients, stepDurations: stepDurations)
    }

    // MARK: - Paging

    func testPageCountIsIngredientsPlusSteps() {
        let state = CookState(recipe: makeRecipe())
        XCTAssertEqual(state.pageCount, 4)
        XCTAssertTrue(state.isOnIngredients)
    }

    func testAdvanceMovesThroughStepsAndStopsAtTheEnd() {
        var state = CookState(recipe: makeRecipe())

        state.advance()
        XCTAssertEqual(state.currentStepIndex, 0)
        XCTAssertEqual(state.currentStepText, "Chop the onions")

        state.advance()
        state.advance()
        XCTAssertTrue(state.isOnLastStep)

        // Past the end is a no-op, not a crash.
        state.advance()
        XCTAssertEqual(state.currentPage, 3)
    }

    func testGoBackStopsAtIngredients() {
        var state = CookState(recipe: makeRecipe(), currentPage: 1)
        state.goBack()
        XCTAssertTrue(state.isOnIngredients)
        state.goBack()
        XCTAssertEqual(state.currentPage, 0)
    }

    func testInitialPageIsClamped() {
        XCTAssertEqual(CookState(recipe: makeRecipe(), currentPage: 99).currentPage, 3)
        XCTAssertEqual(CookState(recipe: makeRecipe(), currentPage: -5).currentPage, 0)
    }

    func testRecipeWithNoStepsIsNotCookable() {
        let state = CookState(recipe: makeRecipe(steps: [], stepDurations: nil))
        XCTAssertFalse(state.isCookable)
        XCTAssertEqual(state.pageCount, 1)
    }

    // MARK: - Timers

    func testDurationIsReadPositionally() {
        let state = CookState(recipe: makeRecipe())
        XCTAssertNil(state.duration(forStep: 0))
        XCTAssertEqual(state.duration(forStep: 1), 1200)
        XCTAssertNil(state.duration(forStep: 2))
    }

    func testCurrentStepDurationFollowsThePage() {
        var state = CookState(recipe: makeRecipe())
        XCTAssertNil(state.currentStepDuration)   // ingredients page
        state.go(to: 2)
        XCTAssertEqual(state.currentStepDuration, 1200)
    }

    func testMissingOrShortDurationsAreTolerated() {
        let noDurations = CookState(recipe: makeRecipe(stepDurations: nil))
        XCTAssertNil(noDurations.duration(forStep: 1))

        // An older server might send fewer entries than there are steps.
        let short = CookState(recipe: makeRecipe(stepDurations: [nil]))
        XCTAssertNil(short.duration(forStep: 2))
    }

    // MARK: - Ingredient checklist

    func testDuplicateIngredientTextDoesNotCollide() {
        // "1 tbsp olive oil" appears in both groups. Keying by text — as the iOS
        // cook mode does — would tick both at once.
        var state = CookState(recipe: makeRecipe())
        let first = IngredientIndex(group: 0, item: 1)
        let second = IngredientIndex(group: 1, item: 0)

        state.toggle(first)

        XCTAssertTrue(state.isChecked(first))
        XCTAssertFalse(state.isChecked(second))
        XCTAssertEqual(state.checkedIngredientCount, 1)
    }

    func testToggleIsReversible() {
        var state = CookState(recipe: makeRecipe())
        let index = IngredientIndex(group: 0, item: 0)
        state.toggle(index)
        state.toggle(index)
        XCTAssertFalse(state.isChecked(index))
        XCTAssertEqual(state.checkedIngredientCount, 0)
    }

    func testAllIngredientsChecked() {
        var state = CookState(recipe: makeRecipe())
        XCTAssertEqual(state.totalIngredientCount, 3)
        XCTAssertFalse(state.allIngredientsChecked)

        state.toggle(IngredientIndex(group: 0, item: 0))
        state.toggle(IngredientIndex(group: 0, item: 1))
        state.toggle(IngredientIndex(group: 1, item: 0))
        XCTAssertTrue(state.allIngredientsChecked)
    }

    func testEmptyIngredientsAreNotConsideredAllChecked() {
        let state = CookState(recipe: makeRecipe(ingredients: []))
        XCTAssertEqual(state.totalIngredientCount, 0)
        XCTAssertFalse(state.allIngredientsChecked)
    }
}
