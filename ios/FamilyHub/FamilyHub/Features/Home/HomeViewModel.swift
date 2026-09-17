import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class HomeViewModel {
    var state: ViewState<DashboardStats> = .idle
    var users: [String: User] = [:]
    var currentUser: User?
    var todayEvents: [CalendarEvent] = []
    var completedChoreIDs: Set<String> = []

    let apiClient: any APIClientProtocol

    private static let currentUserKey = "cached_current_user"

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        currentUser = Self.loadCachedUser()
    }

    // Prevents an Xcode 26 crash where the Swift 6 runtime's isolated-deinit
    // mechanism tries to use task-local storage that doesn't exist when the
    // object is released outside an async context (e.g. sync unit tests).
    #if compiler(>=6.0)
    nonisolated deinit {}
    #endif

    func load() async {
        await load(silent: false)
    }

    /// Loads the dashboard, users, current user and today's events together.
    /// When `silent`, the existing UI is kept while refreshing (no spinner, and
    /// transient failures don't replace already-loaded content).
    private func load(silent: Bool) async {
        if !silent { state = .loading }
        async let statsTask = apiClient.fetchDashboardStats()
        async let usersTask = apiClient.fetchUsers()
        async let meTask = apiClient.fetchMe()
        async let calTask = apiClient.fetchCalendar(view: "day", date: Date())
        do {
            let (stats, userList, me, cal) = try await (statsTask, usersTask, meTask, calTask)
            users = userList.keyedByID
            currentUser = me
            Self.cacheUser(me)
            todayEvents = cal.events
            state = .loaded(stats)
            // The widget's own refresh budget is a few dozen a day, so every
            // load here is worth far more to it than one of its own — and this
            // is the only place that has the dashboard, the events and the
            // users together, which is exactly what a day needs.
            WidgetPublisher.publish(stats: stats, events: cal.events, users: users)
            if silent { completedChoreIDs = [] }
        } catch {
            if !silent { state = .failed(.from(error)) }
        }
    }

    private static func loadCachedUser() -> User? {
        guard let data = UserDefaults.standard.data(forKey: currentUserKey) else { return nil }
        return try? JSONDecoder().decode(User.self, from: data)
    }

    private static func cacheUser(_ user: User) {
        UserDefaults.standard.set(try? JSONEncoder().encode(user), forKey: currentUserKey)
    }

    func completeChore(id: String) async -> Bool {
        completedChoreIDs.insert(id)
        do {
            try await apiClient.completeChore(id: id)
            await load(silent: true)
            return true
        } catch {
            completedChoreIDs.remove(id)
            return false
        }
    }
}
