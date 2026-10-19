import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class ChoresViewModelTests: XCTestCase {
    func testTasksLoadsDefinitionsWithoutLoadingOccurrences() async {
        let fake = FakeAPIClient()
        let chore = Chore(id: "bins", name: "Bins", description: "", status: .pending, eligibleAssignees: ["ben"], recurrenceType: "daily")
        fake.choreDefinitions = [ChoreDefinition(id: "bins", chore: chore, currentTask: nil, active: true)]
        fake.choresResult = .failure(APIError.offline)
        let model = ChoresViewModel(apiClient: fake)
        await model.load()
        guard case .loaded(let definitions) = model.state else { return XCTFail("Definitions must load independently of tasks") }
        XCTAssertEqual(definitions.map(\.id), ["bins"])
        XCTAssertEqual(definitions.first?.chore.recurrenceSummary, "Daily")
    }

    func testInactiveDefinitionsRequireTheFilter() async {
        let fake = FakeAPIClient()
        let chore = Chore(id: "done", name: "Done", description: "", status: .completed)
        fake.choreDefinitions = [ChoreDefinition(id: "done", chore: chore, currentTask: nil, active: false)]
        let model = ChoresViewModel(apiClient: fake)
        await model.load()
        guard case .loaded(let active) = model.state else { return XCTFail("Expected loaded state") }
        XCTAssertTrue(active.isEmpty)
        model.showInactive = true
        await model.load()
        guard case .loaded(let all) = model.state else { return XCTFail("Expected loaded state") }
        XCTAssertEqual(all.count, 1)
    }

    func testDeleteFailureKeepsTheDefinition() async {
        let fake = FakeAPIClient()
        let chore = Chore(id: "bins", name: "Bins", description: "", status: .pending)
        fake.choreDefinitions = [ChoreDefinition(id: "bins", chore: chore, currentTask: nil, active: true)]
        fake.deleteChoreResult = .failure(APIError.offline)
        let model = ChoresViewModel(apiClient: fake)
        await model.load()
        let deleted = await model.deleteChore(id: "bins")
        XCTAssertFalse(deleted)
        XCTAssertNotNil(model.actionError)
        guard case .loaded(let definitions) = model.state else { return XCTFail("Expected loaded state") }
        XCTAssertEqual(definitions.count, 1)
    }
}
