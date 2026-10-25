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
    @State private var photoToCrop: UIImage?
    @State private var isCropPresented = false
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
                }
                accountSection
            }
            .listStyle(.insetGrouped)
            .appListStyle()
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .errorAlert($avatarError)
            .confirmationDialog("Remove your photo?", isPresented: $confirmRemovePhoto, titleVisibility: .visible) {
                Button("Remove Photo", role: .destructive) { Task { await removeAvatar() } }
            }
            .sheet(isPresented: $isCropPresented, onDismiss: { photoToCrop = nil }) {
                if let photoToCrop {
                    AvatarCropView(image: photoToCrop) { croppedImage in
                        Task { await uploadCroppedPhoto(croppedImage) }
                    }
                }
            }
            .task { currentUser = try? await apiClient.fetchMe() }
            .onChange(of: photoPickerItem) { _, item in
                guard let item else { return }
                Task { await uploadSelectedPhoto(item) }
            }
        }
    }

    private var profileSection: some View {
        Section("Your account") {
            HStack(spacing: 14) {
                UserAvatar(user: currentUser, size: 56, apiClient: apiClient)
                    .id(avatarRefreshID)
                    .overlay {
                        if isUploadingAvatar {
                            ProgressView()
                                .scaleEffect(0.8)
                                .padding(6)
                                .background(AppTheme.surface, in: Circle())
                        }
                    }
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 4) {
                    Text(displayName)
                        .font(.body.weight(.semibold))
                    if !displayEmail.isEmpty {
                        Text(displayEmail)
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.secondary)
                    }
                    Label(isAdmin ? "Family admin" : "Family member", systemImage: isAdmin ? "person.2.fill" : "person.fill")
                        .font(.caption)
                        .foregroundStyle(AppTheme.secondary)
                        .padding(.top, 2)
                }
                Spacer(minLength: 0)
            }
            .padding(.vertical, 6)

            PhotosPicker(selection: $photoPickerItem, matching: .images) {
                FeatureEntryLabel(title: "Change Photo", subtitle: "Choose a profile photo", systemImage: "person.crop.square")
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
        }
        .listRowBackground(AppTheme.surface)
    }

    private var appSection: some View {
        Section("This iPhone") {
            NavigationLink {
                ChoreRemindersView(apiClient: apiClient, isAdmin: isAdmin)
            } label: {
                FeatureEntryLabel(title: "Notifications", subtitle: "Reminder schedule and permissions", systemImage: "bell")
            }
            .accessibilityIdentifier("Chore Reminders")

            NavigationLink {
                CommuteSettingsView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Commute", subtitle: "Train times and display schedule", systemImage: "tram.fill")
            }
            .accessibilityIdentifier("Commute settings")
        }
        .listRowBackground(AppTheme.surface)
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
        }
        .accessibilityIdentifier("Server configuration")
    }

    private var adminSection: some View {
        Section("Family administration") {
            NavigationLink {
                FamilyNameView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Family name", subtitle: "Name shown to everyone", systemImage: "house")
            }.accessibilityIdentifier("Family Name")

            NavigationLink {
                MealPlanningSettingsView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Meal planning", subtitle: "Choose the first day of the week", systemImage: "calendar")
            }.accessibilityIdentifier("Meal planning")

            NavigationLink {
                FamilyMembersView(apiClient: apiClient, currentUserID: currentUser?.id ?? "")
            } label: {
                FeatureEntryLabel(title: "Family members", subtitle: "People and admin roles", systemImage: "person.2")
            }.accessibilityIdentifier("Family Members")

            NavigationLink {
                HomeInfoView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Home location", subtitle: "Weather and departure times", systemImage: "mappin.and.ellipse")
            }.accessibilityLabel("Home location")

            NavigationLink {
                FamilyBroadcastView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Send family message", subtitle: "Message all registered iPhones", systemImage: "megaphone")
            }.accessibilityIdentifier("Family message")

            NavigationLink {
                CategoriesView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Categories", subtitle: "Organize meals and chores", systemImage: "square.grid.2x2")
            }

            NavigationLink {
                CalendarSubscriptionsView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Calendars", subtitle: "Manage external calendar feeds", systemImage: "calendar")
            }.accessibilityIdentifier("Calendar feeds settings")

            NavigationLink {
                BinCollectionsView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Bin collections", subtitle: "Upload schedule and set reminders", systemImage: "trash")
            }.accessibilityIdentifier("Bin Collections")

            NavigationLink {
                APITokensView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "API Tokens", subtitle: "Manage service access", systemImage: "key.horizontal")
            }

            NavigationLink {
                DatabaseBackupView(apiClient: apiClient)
            } label: {
                FeatureEntryLabel(title: "Database backup", subtitle: "Save or restore family data", systemImage: "externaldrive")
            }.accessibilityIdentifier("Database backup settings")
        }
        .listRowBackground(AppTheme.surface)
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
        }
        .listRowBackground(AppTheme.surface)
    }

    private func uploadSelectedPhoto(_ item: PhotosPickerItem) async {
        defer { photoPickerItem = nil }
        do {
            guard let rawData = try await item.loadTransferable(type: Data.self) else { return }
            guard let image = UIImage(data: rawData) else {
                avatarError = .from(ProfilePhotoError.invalidImage)
                return
            }
            photoToCrop = image
            isCropPresented = true
        } catch {
            avatarError = .from(error)
        }
    }

    private func uploadCroppedPhoto(_ image: UIImage) async {
        isUploadingAvatar = true
        defer { isUploadingAvatar = false }
        do {
            guard let uploadData = AvatarPhotoProcessor.jpegData(for: image) else {
                throw ProfilePhotoError.encodingFailed
            }
            let mimeType = "image/jpeg"
            let updated = try await apiClient.uploadAvatar(imageData: uploadData, mimeType: mimeType)
            currentUser = updated
            avatarRefreshID = UUID()
        } catch {
            avatarError = .from(error)
        }
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

    private var appVersion: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "—"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "—"
        return "\(version) (\(build))"
    }
}
