import SwiftUI
import FamilyHubKit

struct ChoreRemindersView: View {
    let apiClient: any APIClientProtocol
    let isAdmin: Bool
    @State private var notifications = ChoreNotificationManager.shared
    @State private var settings = ChoreReminders()
    @State private var isLoaded = false
    @State private var isLoading = false
    @State private var isSaving = false
    @State private var actionError: APIError?

    var body: some View {
        Form {
            Section {
                Toggle("Notifications on This iPhone", isOn: Binding(
                    get: { notifications.isEnabled },
                    set: { enabled in Task { await setDeviceEnabled(enabled) } }
                ))
                .disabled(!isLoaded || isSaving || apiClient.serverIdentifier == nil)
                if notifications.isEnabled && !notifications.isRegistered {
                    Text("Connecting this iPhone for notifications…")
                        .foregroundStyle(.secondary)
                }
                if let error = notifications.actionError {
                    Text(error.localizedDescription).foregroundStyle(.red)
                    Button("Retry Connection") { Task { await setDeviceEnabled(true) } }
                }
                Button("Open iPhone Notification Settings") {
                    guard let url = URL(string: UIApplication.openNotificationSettingsURLString) else { return }
                    UIApplication.shared.open(url)
                }
            } footer: {
                Text("Receive admin family messages and reminders for unfinished chores due today or overdue.")
            }
            Section {
                Toggle("Family Reminders", isOn: $settings.enabled)
                    .disabled(!isAdmin || !settings.pushAvailable || isSaving)
                DatePicker("First Reminder", selection: timeBinding(0), displayedComponents: .hourAndMinute)
                DatePicker("Second Reminder", selection: timeBinding(1), displayedComponents: .hourAndMinute)
                LabeledContent("Timezone", value: settings.timezone)
            } header: {
                Text("Family Schedule")
            } footer: {
                Text(settings.pushAvailable
                     ? "The schedule applies to the whole family. An admin can change both times."
                     : "The server needs Apple push credentials before family reminders can start.")
            }
            .disabled(!isLoaded || !isAdmin || isSaving)
        }
        .environment(\.timeZone, TimeZone(secondsFromGMT: 0)!)
        .navigationTitle("Chore Reminders")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if isAdmin {
                Button("Save") { Task { await save() } }
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
        .task { await load() }
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
