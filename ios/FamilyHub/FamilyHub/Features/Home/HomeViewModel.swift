import Foundation
import Observation

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

    func load() async {
        state = .loading
        async let statsTask = apiClient.fetchDashboardStats()
        async let usersTask = apiClient.fetchUsers()
        async let meTask = apiClient.fetchMe()
        async let calTask = apiClient.fetchCalendar(view: "day", date: Date())
        do {
            let (stats, userList, me, cal) = try await (statsTask, usersTask, meTask, calTask)
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
            currentUser = me
            Self.cacheUser(me)
            todayEvents = cal.events
            state = .loaded(stats)
        } catch {
            state = .failed(.from(error))
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
            await loadSilently()
            return true
        } catch {
            completedChoreIDs.remove(id)
            return false
        }
    }

    private func loadSilently() async {
        async let statsTask = apiClient.fetchDashboardStats()
        async let usersTask = apiClient.fetchUsers()
        async let meTask = apiClient.fetchMe()
        async let calTask = apiClient.fetchCalendar(view: "day", date: Date())
        do {
            let (stats, userList, me, cal) = try await (statsTask, usersTask, meTask, calTask)
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
            currentUser = me
            todayEvents = cal.events
            state = .loaded(stats)
            completedChoreIDs = []
        } catch {}
    }
}
