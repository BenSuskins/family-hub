import XCTest
@testable import FamilyHub

@MainActor
final class ChoresViewModelTests: XCTestCase {
    private func makeChore(id: String, status: ChoreStatus) -> Chore {
        Chore(id: id, name: "Chore \(id)", description: "", status: status, dueDate: nil, assignedToUserID: nil)
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
        fake.completeChoreResult = .failure(APIError.server(500))
        let viewModel = ChoresViewModel(apiClient: fake)
        await viewModel.load()

        await viewModel.complete(choreID: "1")

        XCTAssertEqual(viewModel.pendingChores.count, 1)
        XCTAssertNotNil(viewModel.errorMessage)
    }
}
