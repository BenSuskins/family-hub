import XCTest
@testable import FamilyHub

@MainActor
final class HomeViewModelTests: XCTestCase {
    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(DashboardStats(
            choresDueToday: 2,
            choresOverdue: 1,
            choresDueTodayList: [],
            choresOverdueList: [],
            mealsThisWeek: 0,
            todayMeals: []
        ))
        let viewModel = HomeViewModel(apiClient: fake)

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
        let viewModel = HomeViewModel(apiClient: fake)

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
        let viewModel = HomeViewModel(apiClient: FakeAPIClient())
        guard case .idle = viewModel.state else {
            XCTFail("expected idle initial state")
            return
        }
    }

    func testLoadFetchesUsers() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(
            DashboardStats(choresDueToday: 1, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [], mealsThisWeek: 0, todayMeals: [])
        )
        fake.usersResult = .success([User(id: "u1", name: "Ben Suskins", email: "", avatarURL: "", role: "member")])
        let viewModel = HomeViewModel(apiClient: fake)

        await viewModel.load()

        XCTAssertEqual(viewModel.users["u1"]?.name, "Ben Suskins")
    }

    func testCompleteChoreSuccess() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(
            DashboardStats(choresDueToday: 1, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [], mealsThisWeek: 0, todayMeals: [])
        )
        fake.completeChoreResult = .success(())
        let viewModel = HomeViewModel(apiClient: fake)
        await viewModel.load()

        let result = await viewModel.completeChore(id: "c1")

        XCTAssertTrue(result)
    }

    func testCompleteChoreFailure() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(
            DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [], mealsThisWeek: 0, todayMeals: [])
        )
        fake.completeChoreResult = .failure(APIError.server(500))
        let viewModel = HomeViewModel(apiClient: fake)
        await viewModel.load()

        let result = await viewModel.completeChore(id: "c1")

        XCTAssertFalse(result)
    }
}
