import Foundation
import Observation

@Observable
@MainActor
final class DashboardViewModel {
    var state: ViewState<DashboardStats> = .idle

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        do {
            let stats = try await apiClient.fetchDashboardStats()
            state = .loaded(stats)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }
}
