import SwiftUI

@Observable
final class FamilyMembersViewModel {
    var users: [User] = []
    var isLoading = false
    var errorMessage: String?

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            users = try await apiClient.fetchUsers()
        } catch {
            errorMessage = "Failed to load members"
        }
    }

    func promote(user: User) async {
        do {
            let updated = try await apiClient.promoteUser(id: user.id)
            if let index = users.firstIndex(where: { $0.id == updated.id }) {
                users[index] = updated
            }
        } catch {
            errorMessage = "Failed to promote \(user.name)"
        }
    }

    func demote(user: User) async {
        do {
            let updated = try await apiClient.demoteUser(id: user.id)
            if let index = users.firstIndex(where: { $0.id == updated.id }) {
                users[index] = updated
            }
        } catch {
            errorMessage = "Failed to demote \(user.name)"
        }
    }
}

struct FamilyMembersView: View {
    let apiClient: any APIClientProtocol
    let currentUserID: String

    @State private var viewModel: FamilyMembersViewModel

    init(apiClient: any APIClientProtocol, currentUserID: String) {
        self.apiClient = apiClient
        self.currentUserID = currentUserID
        self._viewModel = State(initialValue: FamilyMembersViewModel(apiClient: apiClient))
    }

    var body: some View {
        List(viewModel.users) { user in
            HStack {
                UserAvatar(user: user, size: 36, apiClient: apiClient)
                VStack(alignment: .leading, spacing: 2) {
                    Text(user.name).font(.body)
                    Text(user.email).font(.caption).foregroundStyle(.secondary)
                }
                Spacer()
                if user.isAdmin {
                    Text("Admin")
                        .font(.caption.weight(.medium))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(Color.accentColor.opacity(0.15))
                        .foregroundStyle(Color.accentColor)
                        .clipShape(Capsule())
                }
            }
            .swipeActions(edge: .trailing) {
                if user.id != currentUserID {
                    if user.isAdmin {
                        Button("Demote") { Task { await viewModel.demote(user: user) } }
                            .tint(.orange)
                    } else {
                        Button("Promote") { Task { await viewModel.promote(user: user) } }
                            .tint(.blue)
                    }
                }
            }
        }
        .navigationTitle("Family Members")
        .navigationBarTitleDisplayMode(.inline)
        .overlay {
            if viewModel.isLoading && viewModel.users.isEmpty {
                ProgressView()
            }
        }
        .alert("Error", isPresented: Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )) {
            Button("OK") { viewModel.errorMessage = nil }
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
        .task { await viewModel.load() }
    }
}
