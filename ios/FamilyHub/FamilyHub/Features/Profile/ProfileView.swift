import SwiftUI
import PhotosUI

struct ProfileView: View {
    let apiClient: any APIClientProtocol

    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @Environment(\.dismiss) private var dismiss

    @State private var currentUser: User?
    @State private var photoPickerItem: PhotosPickerItem?
    @State private var isUploadingAvatar = false
    @State private var avatarError: String?

    private var displayName: String { currentUser?.name ?? authManager.displayName }
    private var displayEmail: String { currentUser?.email ?? authManager.email }
    private var isAdmin: Bool { currentUser?.isAdmin ?? false }

    var body: some View {
        NavigationStack {
            List {
                profileSection
                appSection
                if isAdmin { adminSection }
                accountSection
            }
            .listStyle(.insetGrouped)
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .alert("Avatar Error", isPresented: Binding(
                get: { avatarError != nil },
                set: { if !$0 { avatarError = nil } }
            )) {
                Button("OK") { avatarError = nil }
            } message: {
                Text(avatarError ?? "")
            }
            .task { currentUser = try? await apiClient.fetchMe() }
            .onChange(of: photoPickerItem) { _, item in
                guard let item else { return }
                Task { await uploadSelectedPhoto(item) }
            }
        }
    }

    // MARK: - Sections

    private var profileSection: some View {
        Section {
            HStack(spacing: 14) {
                ZStack(alignment: .bottomTrailing) {
                    UserAvatar(user: currentUser, size: 60, apiClient: apiClient)
                    if isUploadingAvatar {
                        ProgressView()
                            .scaleEffect(0.7)
                            .padding(4)
                            .background(Color(.systemBackground))
                            .clipShape(Circle())
                    }
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(displayName)
                        .font(.body.weight(.semibold))
                    if !displayEmail.isEmpty {
                        Text(displayEmail)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .padding(.vertical, 4)

            PhotosPicker(selection: $photoPickerItem, matching: .images) {
                Label("Change Photo", systemImage: "camera")
            }
            .disabled(isUploadingAvatar)

            if currentUser?.avatarURL.hasPrefix("/avatar/") == true {
                Button(role: .destructive) {
                    Task { await removeAvatar() }
                } label: {
                    Label("Remove Photo", systemImage: "trash")
                }
                .disabled(isUploadingAvatar)
            }
        }
    }

    private var appSection: some View {
        Section("App") {
            NavigationLink {
                NavigationStack {
                    ConfigurationFormView(
                        configStore: configStore,
                        discoveryService: URLSessionOIDCDiscoveryService(),
                        onSave: { authManager.signOut() }
                    )
                    .navigationTitle("Server Configuration")
                    .navigationBarTitleDisplayMode(.inline)
                }
            } label: {
                LabeledContent("Server URL") {
                    Text(configStore.baseURL)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
            }

            LabeledContent("Version") {
                Text(appVersion)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private var adminSection: some View {
        Section("Admin") {
            NavigationLink("Family Name") {
                FamilyNameView(apiClient: apiClient)
            }
            NavigationLink("Family Members") {
                FamilyMembersView(apiClient: apiClient, currentUserID: currentUser?.id ?? "")
            }
            NavigationLink("Categories") {
                CategoriesView(apiClient: apiClient)
            }
            NavigationLink("API Tokens") {
                APITokensView(apiClient: apiClient)
            }
        }
    }

    private var accountSection: some View {
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

    // MARK: - Avatar actions

    private func uploadSelectedPhoto(_ item: PhotosPickerItem) async {
        isUploadingAvatar = true
        defer {
            isUploadingAvatar = false
            photoPickerItem = nil
        }
        do {
            guard let data = try await item.loadTransferable(type: Data.self) else { return }
            let mimeType = data.imagesMimeType ?? "image/jpeg"
            let updated = try await apiClient.uploadAvatar(imageData: data, mimeType: mimeType)
            currentUser = updated
        } catch {
            avatarError = "Failed to upload photo"
        }
    }

    private func removeAvatar() async {
        isUploadingAvatar = true
        defer { isUploadingAvatar = false }
        do {
            try await apiClient.deleteAvatar()
            currentUser = try await apiClient.fetchMe()
        } catch {
            avatarError = "Failed to remove photo"
        }
    }

    // MARK: - Helpers

    private var appVersion: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "—"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "—"
        return "\(version) (\(build))"
    }
}

private extension Data {
    var imagesMimeType: String? {
        var byte: UInt8 = 0
        copyBytes(to: &byte, count: 1)
        switch byte {
        case 0xFF: return "image/jpeg"
        case 0x89: return "image/png"
        case 0x47: return "image/gif"
        case 0x49, 0x4D: return "image/tiff"
        default: return nil
        }
    }
}
