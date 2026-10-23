import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class HomeViewModelTests: XCTestCase {
    func testDoesNotRestoreAnotherAccountsCachedProfile() async throws {
        let key = "cached_current_user"
        let previous = UserDefaults.standard.data(forKey: key)
        defer { UserDefaults.standard.set(previous, forKey: key) }
        let otherUser = User(id: "other", name: "Other User", email: "", avatarURL: "/avatar/other", role: "member")
        UserDefaults.standard.set(try JSONEncoder().encode(otherUser), forKey: key)
        let model = HomeViewModel(apiClient: FakeAPIClient())
        XCTAssertNil(model.currentUser)
    }

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
        fake.dashboardResult = .failure(APIError.server(status: 500, serverMessage: nil))
        let viewModel = HomeViewModel(apiClient: fake)

        await viewModel.load()

        guard case .failed(let error) = viewModel.state else {
            XCTFail("expected failed state")
            return
        }
        if case .server(let code, _) = error {
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

    func testIndependentHomeFailuresKeepDashboardVisible() async {
        let fake = FakeAPIClient()
        fake.usersResult = .failure(APIError.offline)
        fake.calendarResult = .failure(APIError.server(status: 500, serverMessage: nil))
        let viewModel = HomeViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let stats) = viewModel.state else { return XCTFail("Dashboard remains visible") }
        XCTAssertEqual(stats.choresDueToday, 0)
        XCTAssertEqual(Set(viewModel.refreshErrors.keys), ["users", "calendar"])
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
        fake.completeChoreResult = .failure(APIError.server(status: 500, serverMessage: nil))
        let viewModel = HomeViewModel(apiClient: fake)
        await viewModel.load()

        let result = await viewModel.completeChore(id: "c1")

        XCTAssertFalse(result)
        XCTAssertNotNil(viewModel.actionError)
        XCTAssertFalse(viewModel.completedChoreIDs.contains("c1"))
    }
}

@MainActor
final class HomeSkipTests: XCTestCase {
    func testSkipRefreshesTheDashboard() async {
        let fake = FakeAPIClient()
        let model = HomeViewModel(apiClient: fake)
        await model.load()
        fake.dashboardResult = .success(DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [], mealsThisWeek: 0, todayMeals: []))
        let success = await model.skipChore(id: "bins")
        XCTAssertTrue(success)
        guard case .loaded(let stats) = model.state else { return XCTFail("Expected refreshed dashboard") }
        XCTAssertEqual(stats.choresOverdue, 0)
    }

    func testSkipFailureKeepsTheTaskVisible() async {
        let fake = FakeAPIClient()
        fake.skipChoreResult = .failure(APIError.offline)
        let model = HomeViewModel(apiClient: fake)
        let success = await model.skipChore(id: "bins")
        XCTAssertFalse(success)
        XCTAssertFalse(model.completedChoreIDs.contains("bins"))
        XCTAssertNotNil(model.actionError)
    }

    func testReminderOpensHomeAndRequestsFreshData() {
        let navigation = AppNavigation()
        navigation.selectedTab = .tasks
        let previous = navigation.rootRefreshID(for: .today)
        navigation.open(URL(string: "familyhub://today")!)
        XCTAssertEqual(navigation.selectedTab, .today)
        XCTAssertNotEqual(navigation.rootRefreshID(for: .today), previous)
    }
}
