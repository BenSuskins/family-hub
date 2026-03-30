import SwiftUI

struct ProfileView: View {
    let apiClient: any APIClientProtocol

    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @Environment(\.dismiss) private var dismiss

    @State private var currentUser: User?
    @State private var showingEditConfigConfirmation = false
    @State private var showingEditConfig = false

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack(spacing: 14) {
                        UserAvatar(user: currentUser, size: 52, apiClient: apiClient)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(currentUser?.name ?? authManager.displayName)
                                .font(.body.weight(.semibold))
                            let email = currentUser?.email ?? authManager.email
                            if !email.isEmpty {
                                Text(email)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }

                Section {
                    Button {
                        showingEditConfigConfirmation = true
                    } label: {
                        Text("Edit Configuration")
                            .font(.body.weight(.medium))
                    }
                }

                Section {
                    Button(role: .destructive) {
                        authManager.signOut()
                        dismiss()
                    } label: {
                        Text("Sign Out")
                            .font(.body.weight(.medium))
                    }
                }
            }
            .listStyle(.insetGrouped)
            .navigationTitle("Profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .confirmationDialog(
                "Editing your configuration will sign you out. Continue?",
                isPresented: $showingEditConfigConfirmation,
                titleVisibility: .visible
            ) {
                Button("Edit Configuration", role: .destructive) {
                    showingEditConfig = true
                }
                Button("Cancel", role: .cancel) {}
            }
            .sheet(isPresented: $showingEditConfig) {
                NavigationStack {
                    ConfigurationFormView(
                        configStore: configStore,
                        discoveryService: URLSessionOIDCDiscoveryService(),
                        onSave: { authManager.signOut() }
                    )
                    .navigationTitle("Edit Configuration")
                    .navigationBarTitleDisplayMode(.inline)
                }
            }
            .task {
                currentUser = try? await apiClient.fetchMe()
            }
        }
    }
}
