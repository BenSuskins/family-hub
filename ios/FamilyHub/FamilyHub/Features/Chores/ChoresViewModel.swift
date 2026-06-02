// ios/FamilyHub/Features/Chores/ChoresViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class ChoresViewModel: MutableListViewModel {
    var state: ViewState<[Chore]> = .idle
    var actionError: APIError?
    var users: [String: User] = [:]
    var categories: [Category] = []
    var currentUserID: String?

    // Derived from loaded chores
    var pendingChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .pending || $0.status == .overdue }
    }

    var overdueChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .overdue }
    }

    var dueSoonChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .pending }
    }

    var completedChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .completed }
    }

    var todayChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .pending && $0.badge == .dueToday }
    }

    var upcomingChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .pending && $0.badge == .dueSoon }
    }

    /// One representative chore per series (plus every non-recurring chore), for the
    /// Manage view. Editing the representative updates the whole series server-side.
    /// Recurring series prefer their earliest still-pending/overdue occurrence.
    var series: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        var bySeries: [String: Chore] = [:]
        var oneOffs: [Chore] = []
        for chore in chores {
            guard let sid = chore.seriesID, !sid.isEmpty else {
                oneOffs.append(chore)
                continue
            }
            if let existing = bySeries[sid] {
                if rank(chore) < rank(existing) {
                    bySeries[sid] = chore
                }
            } else {
                bySeries[sid] = chore
            }
        }
        let combined = Array(bySeries.values) + oneOffs
        return combined.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
    }

    /// Ordering used to pick a series' representative occurrence: not-completed
    /// first, then by due date ascending (nil due dates last).
    private func rank(_ chore: Chore) -> (Int, String) {
        let completed = chore.status == .completed ? 1 : 0
        return (completed, chore.dueDate ?? "9999-99-99")
    }

    func categoryName(_ id: String?) -> String? {
        guard let id, !id.isEmpty else { return nil }
        return categories.first { $0.id == id }?.name
    }

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        async let choresTask = apiClient.fetchChores()
        async let usersTask = apiClient.fetchUsers()
        async let meTask = apiClient.fetchMe()
        // Categories are optional context for the forms; tolerate failure.
        async let categoriesTask = apiClient.fetchCategories()
        do {
            let (chores, userList, me) = try await (choresTask, usersTask, meTask)
            users = userList.keyedByID
            categories = (try? await categoriesTask) ?? []
            currentUserID = me.id
            state = .loaded(chores)
        } catch {
            state = .failed(.from(error))
        }
    }

    func createChore(_ request: ChoreRequest) async -> Chore? {
        await performMutation { try await apiClient.createChore(request) } applying: { created, chores in
            chores.append(created)
        }
    }

    func updateChore(id: String, _ request: ChoreRequest) async -> Chore? {
        await performMutation { try await apiClient.updateChore(id: id, request) } applying: { updated, chores in
            if let index = chores.firstIndex(where: { $0.id == id }) {
                chores[index] = updated
            }
        }
    }

    func deleteChore(id: String) async -> Bool {
        await performDeletion { try await apiClient.deleteChore(id: id) } applying: { chores in
            chores.removeAll { $0.id == id }
        }
    }

    func complete(choreID: String) async -> Bool {
        do {
            try await apiClient.completeChore(id: choreID)
            // Update local state immediately without re-fetching.
            mutateLoaded { chores in
                if let index = chores.firstIndex(where: { $0.id == choreID }) {
                    chores[index] = chores[index].completed
                }
            }
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }
}
