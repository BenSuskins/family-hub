import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class HomeViewModel {
    var actionError: APIError?
    var state: ViewState<DashboardStats> = .idle
    var users: [String: User] = [:]
    var currentUser: User?
    var todayEvents: [CalendarEvent] = []
    var completedChoreIDs: Set<String> = []
    private(set) var refreshErrors: [String: APIError] = [:]
    private(set) var snapshotMetadata: SnapshotMetadata?
    private var loadTask: Task<Void, Never>?

    let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    // Prevents an Xcode 26 crash where the Swift 6 runtime's isolated-deinit
    // mechanism tries to use task-local storage that doesn't exist when the
    // object is released outside an async context (e.g. sync unit tests).
    #if compiler(>=6.0)
    nonisolated deinit {}
    #endif

    func load(forceRefresh: Bool = false) async {
        if let loadTask {
            await loadTask.value
            return
        }
        let task = Task { await performLoad(forceRefresh: forceRefresh) }
        loadTask = task
        await task.value
        loadTask = nil
    }

    private func performLoad(forceRefresh: Bool) async {
        if case .loaded = state {} else { state = .loading }
        refreshErrors = [:]
        async let dashboard: Void = loadDashboard(forceRefresh: forceRefresh)
        async let users: Void = loadUsers(forceRefresh: forceRefresh)
        async let profile: Void = loadProfile(forceRefresh: forceRefresh)
        async let calendar: Void = loadCalendar(forceRefresh: forceRefresh)
        _ = await (dashboard, users, profile, calendar)
        if !Task.isCancelled { completedChoreIDs = [] }
    }

    private func loadDashboard(forceRefresh: Bool) async {
        do {
            let stats = try await apiClient.fetchDashboardStats(forceRefresh: forceRefresh)
            guard !Task.isCancelled else { return }
            snapshotMetadata = await apiClient.snapshotMetadata(for: "dashboard-\(APIDate.dayString(Date()))")
            state = .loaded(stats)
            WidgetPublisher.publish(stats: stats, events: todayEvents, users: users)
        } catch { record(error, section: "dashboard") }
    }

    private func loadUsers(forceRefresh: Bool) async {
        do {
            users = try await apiClient.fetchUsers(forceRefresh: forceRefresh).keyedByID
            guard !Task.isCancelled else { return }
            if case .loaded(let stats) = state { WidgetPublisher.publish(stats: stats, events: todayEvents, users: users) }
        } catch { record(error, section: "users") }
    }

    private func loadProfile(forceRefresh: Bool) async {
        do { currentUser = try await apiClient.fetchMe(forceRefresh: forceRefresh) }
        catch { record(error, section: "profile") }
    }

    private func loadCalendar(forceRefresh: Bool) async {
        do {
            let response = try await apiClient.fetchCalendar(view: "day", date: Date(), forceRefresh: forceRefresh)
            guard !Task.isCancelled else { return }
            todayEvents = response.events
            if case .loaded(let stats) = state { WidgetPublisher.publish(stats: stats, events: todayEvents, users: users) }
        } catch { record(error, section: "calendar") }
    }

    private func record(_ error: Error, section: String) {
        guard !Task.isCancelled else { return }
        let apiError = APIError.from(error)
        refreshErrors[section] = apiError
        if section == "dashboard", case .idle = state { state = .failed(apiError) }
        if section == "dashboard", case .loading = state { state = .failed(apiError) }
    }

    func completeChore(id: String) async -> Bool {
        await resolveChore(id: id) { try await self.apiClient.completeChore(id: id) }
    }

    func skipChore(id: String) async -> Bool {
        await resolveChore(id: id) { try await self.apiClient.skipChore(id: id) }
    }

    private func resolveChore(id: String, operation: () async throws -> Void) async -> Bool {
        guard !completedChoreIDs.contains(id) else { return false }
        actionError = nil
        completedChoreIDs.insert(id)
        do {
            try await operation()
            WidgetPublisher.invalidate()
            NotificationCenter.default.post(name: .familyHubChoresChanged, object: nil)
            await load(forceRefresh: true)
            return true
        } catch {
            actionError = .from(error)
            completedChoreIDs.remove(id)
            return false
        }
    }
}
