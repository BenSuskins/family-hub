import SwiftUI
import PhotosUI
import UIKit
import FamilyHubKit

struct ProfileView: View {
    let apiClient: any APIClientProtocol

    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @Environment(\.dismiss) private var dismiss

    @State private var currentUser: User?
    @State private var photoPickerItem: PhotosPickerItem?
    @State private var isUploadingAvatar = false
    @State private var avatarError: APIError?
    @State private var confirmRemovePhoto = false
    @State private var avatarRefreshID = UUID()

    private var displayName: String { currentUser?.name ?? authManager.displayName }
    private var displayEmail: String { currentUser?.email ?? authManager.email }
    private var isAdmin: Bool { currentUser?.isAdmin ?? false }

    var body: some View {
        NavigationStack {
            List {
                profileSection
                appSection
                if isAdmin { adminSection }
                Section("About") {
                    serverEntry
                    LabeledContent("Version", value: appVersion)
                    Link("Privacy Policy", destination: URL(string: "https://github.com/bensuskins/family-hub/blob/main/PRIVACY.md")!)
                    Link("Terms of Use", destination: URL(string: "https://github.com/bensuskins/family-hub/blob/main/TERMS.md")!)
                }.listRowBackground(AppTheme.surface)
                accountSection
            }
            .listStyle(.insetGrouped)
        .appListStyle()
        .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .errorAlert($avatarError)
            .confirmationDialog("Remove your photo?", isPresented: $confirmRemovePhoto, titleVisibility: .visible) {
                Button("Remove Photo", role: .destructive) { Task { await removeAvatar() } }
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
                    UserAvatar(user: currentUser, size: 42, apiClient: apiClient)
                        .id(avatarRefreshID)
                    if isUploadingAvatar {
                        ProgressView()
                            .scaleEffect(0.7)
                            .padding(4)
                            .background(AppTheme.surface)
                            .clipShape(Circle())
                    }
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(displayName)
                        .font(.body.weight(.semibold))
                    if !displayEmail.isEmpty {
                        Text(displayEmail)
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.secondary)
                    }
                }
            }
            .padding(.vertical, 4)

        }.listRowBackground(AppTheme.surface)
    }

    private var appSection: some View {
        Section("This iPhone") {
            PhotosPicker(selection: $photoPickerItem, matching: .images) {
                FeatureEntryLabel(title: "Photo", subtitle: "Change your profile photo", systemImage: "person.crop.square")
            }
            .disabled(isUploadingAvatar)
            .accessibilityIdentifier("Change Photo")

            if currentUser?.avatarURL.hasPrefix("/avatar/") == true {
                Button(role: .destructive) {
                    confirmRemovePhoto = true
                } label: {
                    Label("Remove Photo", systemImage: "trash")
                }
                .disabled(isUploadingAvatar)
            }
            NavigationLink {
                ChoreRemindersView(apiClient: apiClient, isAdmin: isAdmin)
            } label: {
                FeatureEntryLabel(title: "Notifications", subtitle: "Device permission and reminder schedule", systemImage: "bell")
            }
            .accessibilityIdentifier("Chore Reminders")
            NavigationLink {
                CommuteSettingsView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Commute", subtitle: "Train times and display schedule", systemImage: "tram.fill")
            }
            .accessibilityIdentifier("Commute settings")
        }.listRowBackground(AppTheme.surface)
    }

    private var serverEntry: some View {
            NavigationLink {
                ConfigurationFormView(
                    configStore: configStore,
                    discoveryService: URLSessionOIDCDiscoveryService(),
                    beforeSave: { try await ChoreNotificationManager.shared.disconnect() },
                    onSave: { authManager.signOut() }
                )
                    .navigationTitle("Server")
                    .navigationBarTitleDisplayMode(.large)
            } label: {
                LabeledContent("Server") {
                    Text(configStore.baseURL)
                        .foregroundStyle(AppTheme.secondary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
            }.accessibilityIdentifier("Server configuration")

    }

    private var adminSection: some View {
        Section("Family administration") {
            NavigationLink {
                FamilyNameView(apiClient: apiClient)
            } label: {
                settingsEntry("Family name", detail: "Name shown to everyone")
            }.accessibilityIdentifier("Family Name")
            NavigationLink {
                FamilyMembersView(apiClient: apiClient, currentUserID: currentUser?.id ?? "")
            } label: {
                settingsEntry("Family members", detail: "People and admin roles")
            }.accessibilityIdentifier("Family Members")
            NavigationLink {
                HomeInfoView(apiClient: apiClient)
            } label: {
                settingsEntry("Home location", detail: "Weather and departure times")
            }.accessibilityLabel("Home location")
            NavigationLink {
                FamilyBroadcastView(apiClient: apiClient)
            } label: {
                settingsEntry("Send family message", detail: "Message all registered iPhones")
            }.accessibilityIdentifier("Family message")
            NavigationLink("Categories") {
                CategoriesView(apiClient: apiClient)
            }
            NavigationLink {
                CalendarSubscriptionsView(apiClient: apiClient)
            } label: {
                settingsEntry("Calendars", detail: "Manage external calendar feeds")
            }.accessibilityIdentifier("Calendar feeds settings")
            NavigationLink("API Tokens") {
                APITokensView(apiClient: apiClient)
            }
            NavigationLink {
                DatabaseBackupView(apiClient: apiClient)
            } label: {
                settingsEntry("Database backup", detail: "Save or restore family data")
            }.accessibilityIdentifier("Database backup settings")
        }.listRowBackground(AppTheme.surface)
    }

    private func settingsEntry(_ title: String, detail: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
            Text(detail).font(.subheadline).foregroundStyle(AppTheme.secondary)
        }
    }

    private var accountSection: some View {
        Section {
            Button(role: .destructive) {
                Task {
                    do {
                        try await ChoreNotificationManager.shared.disconnect()
                        authManager.signOut()
                        dismiss()
                    } catch { avatarError = .from(error) }
                }
            } label: {
                Text("Sign Out")
                    .font(.body.weight(.medium))
            }
        }.listRowBackground(AppTheme.surface)
    }

    // MARK: - Avatar actions

    private func uploadSelectedPhoto(_ item: PhotosPickerItem) async {
        isUploadingAvatar = true
        defer {
            isUploadingAvatar = false
            photoPickerItem = nil
        }
        do {
            guard let rawData = try await item.loadTransferable(type: Data.self) else { return }
            let (uploadData, mimeType) = normalizedImageData(rawData)
            let updated = try await apiClient.uploadAvatar(imageData: uploadData, mimeType: mimeType)
            currentUser = updated
            avatarRefreshID = UUID()
        } catch {
            avatarError = .from(error)
        }
    }

    private func normalizedImageData(_ data: Data) -> (Data, String) {
        let serverCompatibleMagicBytes: [UInt8: String] = [0xFF: "image/jpeg", 0x89: "image/png", 0x47: "image/gif", 0x52: "image/webp"]
        if let first = data.first, let mimeType = serverCompatibleMagicBytes[first] {
            return (data, mimeType)
        }
        if let image = UIImage(data: data), let jpeg = image.jpegData(compressionQuality: 0.85) {
            return (jpeg, "image/jpeg")
        }
        return (data, "image/jpeg")
    }

    private func removeAvatar() async {
        isUploadingAvatar = true
        defer { isUploadingAvatar = false }
        do {
            try await apiClient.deleteAvatar()
            currentUser = try await apiClient.fetchMe()
            avatarRefreshID = UUID()
        } catch {
            avatarError = .from(error)
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
