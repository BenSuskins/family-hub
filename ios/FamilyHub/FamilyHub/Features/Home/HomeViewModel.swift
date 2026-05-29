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

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
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
            todayEvents = cal.events
            state = .loaded(stats)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
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
