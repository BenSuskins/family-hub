import XCTest
@testable import FamilyHub

@MainActor
final class DashboardViewModelTests: XCTestCase {
    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(DashboardStats(
            choresDueToday: 2,
            choresOverdue: 1,
            choresDueTodayList: [],
            choresOverdueList: []
        ))
        let viewModel = DashboardViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let stats) = viewModel.state else {
            XCTFail("expected loaded state, got \(viewModel.state)")
            return
        }
        XCTAssertEqual(stats.choresDueToday, 2)
        XCTAssertEqual(stats.choresOverdue, 1)
    }

    func testLoadFailure() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .failure(APIError.server(500))
        let viewModel = DashboardViewModel(apiClient: fake)

        await viewModel.load()

        guard case .failed(let error) = viewModel.state else {
            XCTFail("expected failed state")
            return
        }
        if case .server(let code) = error {
            XCTAssertEqual(code, 500)
        } else {
            XCTFail("expected server error, got \(error)")
        }
    }

    func testInitialStateIsIdle() {
        let viewModel = DashboardViewModel(apiClient: FakeAPIClient())
        guard case .idle = viewModel.state else {
            XCTFail("expected idle initial state")
            return
        }
    }
}
