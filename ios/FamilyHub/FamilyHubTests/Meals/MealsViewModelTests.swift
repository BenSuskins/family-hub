import XCTest
@testable import FamilyHub

@MainActor
final class MealsViewModelTests: XCTestCase {
    private func makeMeal(date: String, mealType: String, name: String) -> MealPlan {
        MealPlan(date: date, mealType: mealType, name: name, notes: "", recipeID: nil)
    }

    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.mealsResult = .success([makeMeal(date: "2026-03-09", mealType: "dinner", name: "Pasta")])
        let viewModel = MealsViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let meals) = viewModel.state else {
            XCTFail("expected loaded state")
            return
        }
        XCTAssertEqual(meals.count, 1)
        XCTAssertEqual(meals.first?.name, "Pasta")
    }

    func testNavigateWeekChangesCurrentWeek() async {
        let fake = FakeAPIClient()
        let viewModel = MealsViewModel(apiClient: fake)
        let original = viewModel.currentWeek

        viewModel.nextWeek()

        XCTAssertNotEqual(viewModel.currentWeek, original)
        // Next week should be 7 days later
        let diff = Calendar.current.dateComponents([.day], from: original, to: viewModel.currentWeek).day
        XCTAssertEqual(diff, 7)
    }
}
