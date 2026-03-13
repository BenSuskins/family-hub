// ios/FamilyHub/Features/Chores/ChoresViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class ChoresViewModel {
    var state: ViewState<[Chore]> = .idle
    var errorMessage: String?

    // Derived from loaded chores
    var pendingChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .pending || $0.status == .overdue }
    }

    var completedChores: [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        return chores.filter { $0.status == .completed }
    }

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        do {
            let chores = try await apiClient.fetchChores()
            state = .loaded(chores)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
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
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            return false
        } catch {
            errorMessage = error.localizedDescription
            return false
        }
    }
}
