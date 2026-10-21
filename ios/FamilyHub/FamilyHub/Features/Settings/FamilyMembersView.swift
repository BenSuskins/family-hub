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
        List {
            ForEach(viewModel.users) { user in
            NavigationLink {
                FamilyMemberRoleView(user: user, currentUserID: currentUserID, model: viewModel)
            } label: {
                let layout = dynamicTypeSize.isAccessibilitySize
                    ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
                    : AnyLayout(HStackLayout(spacing: 12))
                layout {
                    if !dynamicTypeSize.isAccessibilitySize {
                        if user.avatarURL.isEmpty {
                            WarmIcon(symbol: "person")
                        } else {
                            UserAvatar(user: user, size: 42, apiClient: apiClient).accessibilityHidden(true)
                        }
                    }
                    VStack(alignment: .leading, spacing: 4) {
                        Text(user.name).font(.body)
                        Text(user.email + (user.id == currentUserID ? " · You" : "")).font(.caption).foregroundStyle(AppTheme.secondary)
                    }.fixedSize(horizontal: false, vertical: true)
                    if user.isAdmin {
                        Text("Admin").font(.caption.weight(.semibold)).foregroundStyle(AppTheme.accent)
                            .padding(6).background(AppTheme.sage, in: Capsule())
                    }
                }.padding(.vertical, 4)
            }
            .accessibilityLabel(user.name)
            .accessibilityValue(user.isAdmin ? "Admin" : "Member")
            .accessibilityHint("View role details.")
            .accessibilityIdentifier("member-" + user.id)
            .listRowBackground(AppTheme.surface)
            }
            Section {
                Text("Select a person to review their role. Admins can change family settings.")
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)
        }
        .appListStyle()
        .navigationTitle("Family members")
        .navigationBarTitleDisplayMode(.large)
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
    @Environment(\.dismiss) private var dismiss
    let user: User
    let currentUserID: String
    @Bindable var model: FamilyMembersViewModel
    @State private var selectedIsAdmin: Bool
    @State private var confirming = false
    @State private var isSaving = false

    init(user: User, currentUserID: String, model: FamilyMembersViewModel) {
        self.user = user
        self.currentUserID = currentUserID
        self.model = model
        _selectedIsAdmin = State(initialValue: user.isAdmin)
    }

    private var current: User { model.users.first { $0.id == user.id } ?? user }
    private var isOwnRole: Bool { user.id == currentUserID }

    var body: some View {
        Form {
            Section {
                Text(current.email).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)
            Section {
                roleOption("Member", description: "Use the family’s shared features.", isAdmin: false)
                roleOption("Admin", description: "Also manage people, settings and API tokens.", isAdmin: true)
            } header: { Text("Family role") } footer: {
                Text(isOwnRole ? "You cannot change your own role here." : "A role change affects access to family settings.")
            }.listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle(current.name)
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") { confirming = true }
                    .buttonStyle(AppCommitButtonStyle())
                    .disabled(isOwnRole || isSaving || selectedIsAdmin == current.isAdmin)
            }
        }
        .confirmationDialog("Change \(current.name)’s role?", isPresented: $confirming, titleVisibility: .visible) {
            Button(selectedIsAdmin ? "Make admin" : "Make member") {
                Task {
                    isSaving = true
                    model.actionError = nil
                    if selectedIsAdmin { await model.promote(user: current) }
                    else { await model.demote(user: current) }
                    isSaving = false
                    if model.actionError == nil { dismiss() }
                }
            }
        }
        .errorAlert($model.actionError)
    }

    private func roleOption(_ title: String, description: String, isAdmin: Bool) -> some View {
        Button {
            selectedIsAdmin = isAdmin
        } label: {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(title).foregroundStyle(AppTheme.ink)
                    Text(description).font(.subheadline).foregroundStyle(AppTheme.secondary)
                }.fixedSize(horizontal: false, vertical: true)
                Spacer(minLength: 4)
                if selectedIsAdmin == isAdmin {
                    Image(systemName: "checkmark").foregroundStyle(AppTheme.accent)
                }
            }.padding(.vertical, 6)
        }
        .accessibilityIdentifier(isAdmin ? "Choose admin role" : "Choose member role")
        .accessibilityAddTraits(selectedIsAdmin == isAdmin ? .isSelected : [])
        .disabled(isOwnRole || isSaving)
    }
}
