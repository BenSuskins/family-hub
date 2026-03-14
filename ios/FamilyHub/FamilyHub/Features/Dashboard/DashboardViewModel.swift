import Foundation
import Observation

@Observable
@MainActor
final class DashboardViewModel {
    var state: ViewState<DashboardStats> = .idle
    var users: [String: User] = [:]

    let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        async let statsTask = apiClient.fetchDashboardStats()
        async let usersTask = apiClient.fetchUsers()
        do {
            let (stats, userList) = try await (statsTask, usersTask)
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
            state = .loaded(stats)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }
}
