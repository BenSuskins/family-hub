import SwiftUI
import FamilyHubKit
import UserNotifications

struct ChoreRemindersView: View {
    let apiClient: any APIClientProtocol
    let isAdmin: Bool
    @State private var notifications = ChoreNotificationManager.shared
    @State private var settings = ChoreReminders()
    @State private var isLoaded = false
    @State private var isLoading = false
    @State private var isSaving = false
    @State private var actionError: APIError?
    @State private var permissionDescription = "Checking permission…"
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        Form {
            Section {
                Toggle("Receive notifications", isOn: Binding(
                    get: { notifications.isEnabled },
                    set: { enabled in Task { await setDeviceEnabled(enabled) } }
                ))
                .disabled(!isLoaded || isSaving || apiClient.serverIdentifier == nil)
                LabeledContent("Permission", value: permissionDescription)
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                if notifications.isEnabled && !notifications.isRegistered {
                    Text("Connecting this iPhone for notifications…")
                        .foregroundStyle(AppTheme.secondary)
                }
                if let error = notifications.actionError {
                    Text(error.localizedDescription).foregroundStyle(AppTheme.destructive)
                    Button("Retry Connection") { Task { await setDeviceEnabled(true) } }
                }
            } header: { Text("This iPhone") } footer: {
                Text(apiClient.serverIdentifier == nil ? "Demo mode cannot register this iPhone for notifications." : "Receive family messages and reminders for unfinished chores due today or overdue.")
            }.listRowBackground(AppTheme.surface)
            Section {
                Button("Open iPhone Settings") {
                    guard let url = URL(string: UIApplication.openNotificationSettingsURLString) else { return }
                    UIApplication.shared.open(url)
                }
                .buttonStyle(AppSecondaryButtonStyle())
                .accessibilityIdentifier("Open iPhone Notification Settings")
            }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
            Section {
                Toggle("Send chore reminders", isOn: $settings.enabled)
                    .accessibilityIdentifier("Family Reminders")
                    .disabled(!isAdmin || !settings.pushAvailable || isSaving)
                DatePicker("First reminder", selection: timeBinding(0), displayedComponents: .hourAndMinute)
                DatePicker("Second reminder", selection: timeBinding(1), displayedComponents: .hourAndMinute)
                LabeledContent("Time zone", value: settings.timezone)
            } header: {
                Text("Family reminder schedule")
            } footer: {
                Text(settings.pushAvailable
                     ? "The schedule applies to the whole family. An admin can change both times."
                     : "The server needs Apple push credentials before family reminders can start.")
            }.listRowBackground(AppTheme.surface)
            .disabled(!isLoaded || !isAdmin || isSaving)
        }
        .environment(\.timeZone, TimeZone(secondsFromGMT: 0)!)
        .appListStyle()
        .navigationTitle("Notifications")
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            if isAdmin {
                Button("Save") { Task { await save() } }.buttonStyle(AppCommitButtonStyle())
                    .disabled(!isLoaded || isSaving || settings.times[0] == settings.times[1])
            }
        }
        .overlay {
            if isLoading { ProgressView() }
            else if !isLoaded {
                ContentUnavailableView {
                    Label("Reminders Unavailable", systemImage: "bell.slash")
                } actions: {
                    Button("Retry") { Task { await load() } }
                }
            }
        }
        .errorAlert($actionError)
        .task { await load(); await refreshPermission() }
        .onChange(of: scenePhase) { _, phase in if phase == .active { Task { await refreshPermission() } } }
    }

    private func refreshPermission() async {
        let status = await UNUserNotificationCenter.current().notificationSettings().authorizationStatus
        switch status {
        case .authorized, .provisional, .ephemeral: permissionDescription = "Allowed in iPhone Settings"
        case .denied: permissionDescription = "Off in iPhone Settings"
        case .notDetermined: permissionDescription = "Not requested"
        @unknown default: permissionDescription = "Check iPhone Settings"
        }
    }

    private func timeBinding(_ index: Int) -> Binding<Date> {
        Binding(get: {
            ISO8601DateFormatter().date(from: "2001-01-01T\(settings.times[index]):00Z") ?? .now
        }, set: { date in
            let formatter = DateFormatter()
            formatter.locale = Locale(identifier: "en_US_POSIX")
            formatter.timeZone = TimeZone(secondsFromGMT: 0)
            formatter.dateFormat = "HH:mm"
            settings.times[index] = formatter.string(from: date)
        })
    }

    private func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            settings = try await apiClient.fetchChoreReminders()
            try await notifications.configure(apiClient: apiClient)
            isLoaded = true
        } catch { actionError = .from(error) }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            try await apiClient.updateChoreReminders(settings)
            settings = try await apiClient.fetchChoreReminders()
        } catch { actionError = .from(error) }
    }

    private func setDeviceEnabled(_ enabled: Bool) async {
        isSaving = true
        defer { isSaving = false }
        do {
            if enabled { try await notifications.enable() }
            else { try await notifications.disable() }
        } catch { actionError = .from(error) }
    }
}
