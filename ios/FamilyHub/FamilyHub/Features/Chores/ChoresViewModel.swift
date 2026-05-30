// ios/FamilyHub/Features/Chores/ChoresViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class ChoresViewModel {
    var state: ViewState<[Chore]> = .idle
    var actionError: APIError?
    var users: [String: User] = [:]
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

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        async let choresTask = apiClient.fetchChores()
        async let usersTask = apiClient.fetchUsers()
        async let meTask = apiClient.fetchMe()
        do {
            let (chores, userList, me) = try await (choresTask, usersTask, meTask)
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
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
                chores[index] = Chore(
                    id: chores[index].id,
                    name: chores[index].name,
                    description: chores[index].description,
                    status: .completed,
                    dueDate: chores[index].dueDate,
                    assignedToUserID: chores[index].assignedToUserID
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
