import Foundation
import Observation

@Observable
@MainActor
final class HomeViewModel {
    var state: ViewState<DashboardStats> = .idle
    var users: [String: User] = [:]
    var currentUser: User?

    let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        async let statsTask = apiClient.fetchDashboardStats()
        async let usersTask = apiClient.fetchUsers()
        async let meTask = apiClient.fetchMe()
        do {
            let (stats, userList, me) = try await (statsTask, usersTask, meTask)
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
            currentUser = me
            state = .loaded(stats)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }

    func completeChore(id: String) async -> Bool {
        do {
            try await apiClient.completeChore(id: id)
            await load()
            return true
        } catch {
            return false
        }
    }
}
