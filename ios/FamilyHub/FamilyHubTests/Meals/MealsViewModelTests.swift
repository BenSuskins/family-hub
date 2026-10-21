import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class MealsViewModelTests: XCTestCase {
    func testEarlierWeekResponseCannotReplaceTheSelectedWeek() async {
        let client = FakeAPIClient()
        client.mealsByWeek = [
            "2026-10-16": [makeMeal(date: "2026-10-18", mealType: "dinner", name: "Pasta")],
            "2026-10-23": [makeMeal(date: "2026-10-24", mealType: "dinner", name: "Soup")],
        ]
        let started = expectation(description: "Earlier week request starts")
        var finishEarlierWeek: CheckedContinuation<Void, Never>?
        client.beforeFetchMeals = { week in
            if APIDate.dayString(week) == "2026-10-16" {
                await withCheckedContinuation { continuation in
                    finishEarlierWeek = continuation
                    started.fulfill()
                }
            }
        }
        let model = MealsViewModel(apiClient: client)
        model.currentWeek = APIDate.parse("2026-10-16T00:00:00Z")!
        let loading = Task { await model.load() }
        await fulfillment(of: [started], timeout: 2)

        await model.selectDay(APIDate.parse("2026-10-24T12:00:00Z")!)
        finishEarlierWeek?.resume()
        await loading.value

        guard case .loaded(let meals) = model.state else {
            return XCTFail("The selected week stays loaded.")
        }
        XCTAssertEqual(meals.first?.name, "Soup")
        XCTAssertEqual(APIDate.dayString(model.selectedDay), "2026-10-24")
    }

    func testTodaySelectsTodayAfterAnotherWeek() async {
        let model = MealsViewModel(apiClient: FakeAPIClient())
        await model.selectDay(APIDate.parse("2020-01-01T12:00:00Z")!)

        model.goToCurrentWeek()

        XCTAssertEqual(APIDate.dayString(model.selectedDay), APIDate.dayString(Date()))
        XCTAssertEqual(model.weekDays.first.map { Calendar.current.component(.weekday, from: $0) }, 6)
        XCTAssertTrue(model.weekDays.contains { Calendar.current.isDateInToday($0) })
    }

    func testPreviousWeekSelectsThursdayAcrossTheMonthBoundary() async {
        let model = MealsViewModel(apiClient: FakeAPIClient())
        await model.selectDay(APIDate.parse("2026-11-02T12:00:00Z")!)

        model.previousWeek()

        XCTAssertEqual(APIDate.dayString(model.currentWeek), "2026-10-23")
        XCTAssertEqual(APIDate.dayString(model.selectedDay), "2026-10-29")
    }

    func testNextWeekSelectsFridayAcrossTheYearBoundary() async {
        let model = MealsViewModel(apiClient: FakeAPIClient())
        await model.selectDay(APIDate.parse("2026-12-30T12:00:00Z")!)

        model.nextWeek()

        XCTAssertEqual(APIDate.dayString(model.currentWeek), "2027-01-01")
        XCTAssertEqual(APIDate.dayString(model.selectedDay), "2027-01-01")
    }

    func testSelectingADayLoadsItsFridayWeek() async {
        let client = FakeAPIClient()
        client.mealsByWeek = [
            "2026-10-16": [makeMeal(date: "2026-10-18", mealType: "dinner", name: "Pasta")],
            "2026-10-23": [makeMeal(date: "2026-10-24", mealType: "dinner", name: "Soup")],
        ]
        let model = MealsViewModel(apiClient: client)
        model.currentWeek = APIDate.parse("2026-10-16T12:00:00Z")!
        await model.load()

        await model.selectDay(APIDate.parse("2026-10-24T12:00:00Z")!)

        XCTAssertEqual(APIDate.dayString(model.currentWeek), "2026-10-23")
        XCTAssertEqual(APIDate.dayString(model.selectedDay), "2026-10-24")
        guard case .loaded(let meals) = model.state else {
            return XCTFail("The selected week must load.")
        }
        XCTAssertEqual(meals.first?.name, "Soup")
    }

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
    func testFailedSaveKeepsLoadedMealsAndReportsError() async {
        let fake = FakeAPIClient()
        fake.mealsResult = .success([makeMeal(date: "2026-10-18", mealType: "dinner", name: "Pasta")])
        fake.saveMealResult = .failure(APIError.server(status: 500, serverMessage: nil))
        let viewModel = MealsViewModel(apiClient: fake)
        await viewModel.load()

        let saved = await viewModel.saveMeal(date: "2026-10-18", mealType: "dinner", name: "Soup")

        XCTAssertFalse(saved)
        XCTAssertNotNil(viewModel.actionError)
        guard case .loaded(let meals) = viewModel.state else {
            return XCTFail("The existing meal plan stays available after a failed save.")
        }
        XCTAssertEqual(meals.first?.name, "Pasta")
    }

}

final class MealDraftTests: XCTestCase {
    func testTypingPreservesTheRecipeUntilTheUserChoosesNameOnly() {
        var draft = MealDraft(name: "Chicken Stir Fry", recipeID: "chicken")
        draft.name = "Chicken for dinner"
        XCTAssertEqual(draft.recipeID, "chicken")
        draft.useNameOnly()
        XCTAssertNil(draft.recipeID)
        XCTAssertEqual(draft.name, "Chicken for dinner")
        draft.select(Recipe(id: "pasta", title: "Pasta"))
        XCTAssertEqual(draft.name, "Pasta")
        XCTAssertEqual(draft.recipeID, "pasta")
    }
}
