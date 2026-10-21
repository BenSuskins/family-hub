import SwiftUI
import FamilyHubKit

@Observable
final class FamilyMembersViewModel {
    var users: [User] = []
    var isLoading = false
    var actionError: APIError?

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
            actionError = .from(error)
        }
    }

    func promote(user: User) async {
        do {
            let updated = try await apiClient.promoteUser(id: user.id)
            if let index = users.firstIndex(where: { $0.id == updated.id }) {
                users[index] = updated
            }
        } catch {
            actionError = .from(error)
        }
    }

    func demote(user: User) async {
        do {
            let updated = try await apiClient.demoteUser(id: user.id)
            if let index = users.firstIndex(where: { $0.id == updated.id }) {
                users[index] = updated
            }
        } catch {
            actionError = .from(error)
        }
    }
}

struct FamilyMembersView: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
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
            NavigationLink {
                FamilyMemberRoleView(user: user, currentUserID: currentUserID, model: viewModel)
            } label: {
                HStack(spacing: 12) {
                    if !dynamicTypeSize.isAccessibilitySize {
                        UserAvatar(user: user, size: 36, apiClient: apiClient).accessibilityHidden(true)
                    }
                    VStack(alignment: .leading, spacing: 4) {
                        Text(user.name).font(.body)
                        Text(user.email).font(.caption).foregroundStyle(AppTheme.secondary)
                        Text(user.isAdmin ? "Admin" : "Member").font(.caption.weight(.semibold)).foregroundStyle(AppTheme.accent)
                    }.fixedSize(horizontal: false, vertical: true)
                }.padding(.vertical, 4)
            }
            .accessibilityLabel(user.name)
            .accessibilityHint(user.isAdmin ? "Admin. View role details." : "Member. View role details.")
            .accessibilityIdentifier("member-" + user.id)
            .listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle("Family Members")
        .navigationBarTitleDisplayMode(.inline)
        .overlay {
            if viewModel.isLoading && viewModel.users.isEmpty {
                ProgressView()
            }
        }
        .errorAlert($viewModel.actionError)
        .task { await viewModel.load() }
    }
}

private struct FamilyMemberRoleView: View {
    let user: User
    let currentUserID: String
    @Bindable var model: FamilyMembersViewModel
    @State private var confirming = false
    @State private var isSaving = false
    private var current: User { model.users.first { $0.id == user.id } ?? user }

    var body: some View {
        Form {
            Section {
                LabeledContent("Name", value: current.name)
                LabeledContent("Email", value: current.email)
                LabeledContent("Role", value: current.isAdmin ? "Admin" : "Member")
            }.listRowBackground(AppTheme.surface)
            Section("Family roles") {
                Text("Members use family meals, chores, calendars, and household lists.")
                Text("Admins also manage family settings, roles, categories, and API tokens.")
                if user.id == currentUserID {
                    Text("You cannot change your own role here.").foregroundStyle(AppTheme.secondary)
                } else {
                    Button(current.isAdmin ? "Make member" : "Make admin") { confirming = true }
                        .disabled(isSaving)
                }
            }.listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle(current.name)
        .confirmationDialog("Change \(current.name)’s role?", isPresented: $confirming, titleVisibility: .visible) {
            Button(current.isAdmin ? "Make member" : "Make admin") {
                Task {
                    isSaving = true
                    if current.isAdmin { await model.demote(user: current) }
                    else { await model.promote(user: current) }
                    isSaving = false
                }
            }
        }
        .errorAlert($model.actionError)
    }
}
