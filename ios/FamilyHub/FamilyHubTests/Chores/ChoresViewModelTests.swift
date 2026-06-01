import XCTest
@testable import FamilyHub

@MainActor
final class ChoresViewModelTests: XCTestCase {
    private func makeChore(id: String, status: ChoreStatus) -> Chore {
        Chore(id: id, name: "Chore \(id)", description: "", status: status, dueDate: nil, assignedToUserID: nil)
    }

    private func makeSeriesChore(id: String, name: String, status: ChoreStatus, dueDate: String?, seriesID: String?) -> Chore {
        Chore(id: id, name: name, description: "", status: status, dueDate: dueDate, assignedToUserID: nil,
              recurrenceType: seriesID == nil ? "none" : "weekly", seriesID: seriesID)
    }

    func testLoadFiltersChoresByStatus() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([
            makeChore(id: "1", status: .pending),
            makeChore(id: "2", status: .completed),
            makeChore(id: "3", status: .overdue),
        ])
        let viewModel = ChoresViewModel(apiClient: fake)

        await viewModel.load()

        XCTAssertEqual(viewModel.pendingChores.count, 2) // pending + overdue
        XCTAssertEqual(viewModel.completedChores.count, 1)
    }

    func testCompleteChoreUpdatesLocalState() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([makeChore(id: "1", status: .pending)])
        fake.completeChoreResult = .success(())
        let viewModel = ChoresViewModel(apiClient: fake)
        await viewModel.load()

        await viewModel.complete(choreID: "1")

        XCTAssertEqual(viewModel.pendingChores.count, 0)
        XCTAssertEqual(viewModel.completedChores.count, 1)
    }

    func testCompleteChoreFailureLeavesStateUnchanged() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([makeChore(id: "1", status: .pending)])
        fake.completeChoreResult = .failure(APIError.server(status: 500, serverMessage: nil))
        let viewModel = ChoresViewModel(apiClient: fake)
        await viewModel.load()

        await viewModel.complete(choreID: "1")

        XCTAssertEqual(viewModel.pendingChores.count, 1)
        XCTAssertNotNil(viewModel.actionError)
    }

    func testOverdueChoresAreGroupedSeparately() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([
            makeChore(id: "1", status: .overdue),
            makeChore(id: "2", status: .pending),
        ])
        let viewModel = ChoresViewModel(apiClient: fake)

        await viewModel.load()

        XCTAssertEqual(viewModel.overdueChores.count, 1)
        XCTAssertEqual(viewModel.dueSoonChores.count, 1)
    }

    func testCompletedChoreDoesNotAppearInOverdueOrDueSoon() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([makeChore(id: "1", status: .completed)])
        let viewModel = ChoresViewModel(apiClient: fake)

        await viewModel.load()

        XCTAssertEqual(viewModel.overdueChores.count, 0)
        XCTAssertEqual(viewModel.dueSoonChores.count, 0)
        XCTAssertEqual(viewModel.completedChores.count, 1)
    }

    func testSeriesDeduplicatesBySeriesIDAndKeepsOneOffs() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([
            // Two occurrences of the same series — should collapse to one.
            makeSeriesChore(id: "a1", name: "Dishes", status: .completed, dueDate: "2026-05-01", seriesID: "s-1"),
            makeSeriesChore(id: "a2", name: "Dishes", status: .pending, dueDate: "2026-05-08", seriesID: "s-1"),
            // A different series.
            makeSeriesChore(id: "b1", name: "Trash", status: .pending, dueDate: "2026-05-03", seriesID: "s-2"),
            // A one-off chore (no series).
            makeSeriesChore(id: "c1", name: "Wash car", status: .pending, dueDate: nil, seriesID: nil),
        ])
        let viewModel = ChoresViewModel(apiClient: fake)

        await viewModel.load()

        let series = viewModel.series
        XCTAssertEqual(series.count, 3) // s-1 (one row), s-2, one-off
        // The s-1 representative should prefer the earliest still-pending occurrence.
        let dishes = series.first { $0.name == "Dishes" }
        XCTAssertEqual(dishes?.id, "a2")
        // Sorted by name: Dishes, Trash, Wash car
        XCTAssertEqual(series.map(\.name), ["Dishes", "Trash", "Wash car"])
    }

    func testLoadPopulatesCategories() async {
        let fake = FakeAPIClient()
        fake.choresResult = .success([])
        fake.categoriesResult = .success([
            FamilyHub.Category(id: "cat-1", name: "Kitchen"),
            FamilyHub.Category(id: "cat-2", name: "Outdoor"),
        ])
        let viewModel = ChoresViewModel(apiClient: fake)

        await viewModel.load()

        XCTAssertEqual(viewModel.categories.count, 2)
        XCTAssertEqual(viewModel.categoryName("cat-1"), "Kitchen")
        XCTAssertNil(viewModel.categoryName(nil))
    }
}
