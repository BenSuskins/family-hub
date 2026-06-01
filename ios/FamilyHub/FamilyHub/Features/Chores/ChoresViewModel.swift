// ios/FamilyHub/Features/Chores/ChoresViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class ChoresViewModel {
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
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
            categories = (try? await categoriesTask) ?? []
            currentUserID = me.id
            state = .loaded(chores)
        } catch {
            state = .failed(.from(error))
        }
    }

    func createChore(_ request: ChoreRequest) async -> Chore? {
        do {
            let created = try await apiClient.createChore(request)
            if case .loaded(var chores) = state {
                chores.append(created)
                state = .loaded(chores)
            }
            return created
        } catch {
            actionError = .from(error)
            return nil
        }
    }

    func updateChore(id: String, _ request: ChoreRequest) async -> Chore? {
        do {
            let updated = try await apiClient.updateChore(id: id, request)
            if case .loaded(var chores) = state {
                if let i = chores.firstIndex(where: { $0.id == id }) {
                    chores[i] = updated
                }
                state = .loaded(chores)
            }
            return updated
        } catch {
            actionError = .from(error)
            return nil
        }
    }

    func deleteChore(id: String) async -> Bool {
        do {
            try await apiClient.deleteChore(id: id)
            if case .loaded(var chores) = state {
                chores.removeAll { $0.id == id }
                state = .loaded(chores)
            }
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }

    func complete(choreID: String) async -> Bool {
        do {
            try await apiClient.completeChore(id: choreID)
            // Update local state immediately without re-fetching
            guard case .loaded(var chores) = state else { return true }
            if let index = chores.firstIndex(where: { $0.id == choreID }) {
                let existing = chores[index]
                chores[index] = Chore(
                    id: existing.id,
                    name: existing.name,
                    description: existing.description,
                    status: .completed,
                    dueDate: existing.dueDate,
                    assignedToUserID: existing.assignedToUserID,
                    categoryID: existing.categoryID,
                    dueTime: existing.dueTime,
                    eligibleAssignees: existing.eligibleAssignees,
                    recurrenceType: existing.recurrenceType,
                    recurrenceValue: existing.recurrenceValue,
                    recurOnComplete: existing.recurOnComplete,
                    seriesID: existing.seriesID,
                    recurrenceUntil: existing.recurrenceUntil,
                    recurrenceCount: existing.recurrenceCount
                )
                state = .loaded(chores)
            }
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }
}
