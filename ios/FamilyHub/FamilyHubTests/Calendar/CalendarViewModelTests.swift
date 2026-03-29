import XCTest
@testable import FamilyHub

@MainActor
final class CalendarViewModelTests: XCTestCase {
    private func makeChore(id: String, name: String, dueDate: String) -> Chore {
        Chore(id: id, name: name, description: "", status: .pending,
              dueDate: dueDate + "T00:00:00Z", assignedToUserID: nil)
    }

    private func makeResponse(chores: [Chore] = [], events: [CalendarEvent] = [], meals: [MealPlan] = []) -> CalendarResponse {
        CalendarResponse(chores: chores, events: events, meals: meals)
    }

    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.calendarResult = .success(makeResponse(chores: [makeChore(id: "1", name: "Clean", dueDate: "2026-03-15")]))
        let viewModel = CalendarViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let response) = viewModel.state else {
            XCTFail("expected loaded state")
            return
        }
        XCTAssertEqual(response.chores.count, 1)
    }

    func testChoresForDayFiltersCorrectly() async {
        let fake = FakeAPIClient()
        fake.calendarResult = .success(makeResponse(chores: [
            makeChore(id: "1", name: "A", dueDate: "2026-03-15"),
            makeChore(id: "2", name: "B", dueDate: "2026-03-20"),
        ]))
        let viewModel = CalendarViewModel(apiClient: fake)
        await viewModel.load()

        let march15 = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        let chores = viewModel.chores(for: march15)

        XCTAssertEqual(chores.count, 1)
        XCTAssertEqual(chores.first?.name, "A")
    }
}
