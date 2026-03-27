import SwiftUI

struct ProfileView: View {
    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @Environment(\.dismiss) private var dismiss

    @State private var showingEditConfigConfirmation = false
    @State private var showingEditConfig = false

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                List {
                    Section {
                        HStack(spacing: 14) {
                            UserAvatar(user: nil, size: 52)
                            VStack(alignment: .leading, spacing: 4) {
                                Text(authManager.displayName)
                                    .font(.system(size: 16, weight: .semibold))
                                    .foregroundStyle(Theme.textPrimary)
                                if !authManager.email.isEmpty {
                                    Text(authManager.email)
                                        .font(.system(size: 13))
                                        .foregroundStyle(Theme.textSecondary)
                                }
                            }
                        }
                        .padding(.vertical, 4)
                        .listRowBackground(Theme.surface)
                    }

                    Section {
                        Button {
                            showingEditConfigConfirmation = true
                        } label: {
                            Text("Edit Configuration")
                                .font(.system(size: 15, weight: .medium))
                                .foregroundStyle(Theme.textPrimary)
                        }
                        .listRowBackground(Theme.surface)
                    }

                    Section {
                        Button(role: .destructive) {
                            authManager.signOut()
                            dismiss()
                        } label: {
                            Text("Sign Out")
                                .font(.system(size: 15, weight: .medium))
                                .foregroundStyle(Theme.statusRed)
                        }
                        .listRowBackground(Theme.surface)
                    }
                }
                .scrollContentBackground(.hidden)
                .listStyle(.insetGrouped)
            }
            .navigationTitle("Profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(Theme.accent)
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
                    .toolbarBackground(Theme.background, for: .navigationBar)
                }
            }
        }
    }
}
