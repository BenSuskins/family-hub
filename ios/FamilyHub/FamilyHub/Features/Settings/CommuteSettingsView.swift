import SwiftUI
import FamilyHubKit

struct CommuteSettingsView: View {
    let apiClient: any APIClientProtocol
    private let store = CommutePreferencesStore()
    @State private var state: ViewState<String> = .idle
    @State private var preferences = CommutePreferences()
    @State private var savedPreferences = CommutePreferences()
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    private let weekdays = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]

    var body: some View {
        StateContentView(state: state, retry: load) { _ in
            Form {
                stationSection
                countSection
                ForEach(CommuteDirection.allCases, id: \.self) { direction in
                    scheduleSection(direction)
                }
                Section {
                    Text("Times use Europe/London. Your settings apply to your account on this device only.")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.secondary)
                }.listRowBackground(Color.clear)
            }
            .appListStyle()
        }
        .groupedBackground()
        .navigationTitle("Commute")
        .navigationBarTitleDisplayMode(.large)
        .environment(\.timeZone, TimeZone(secondsFromGMT: 0)!)
        .task { await load() }
    }

    private var stationSection: some View {
        Section {
            TextField("Home station code", text: stationCodeBinding(\.homeStationCode))
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .accessibilityIdentifier("commute-home-station-code")
            TextField("Work station code", text: stationCodeBinding(\.workStationCode))
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .accessibilityIdentifier("commute-work-station-code")
            if !preferences.hasValidStations {
                Text("Enter a three-letter station code for Home and Work before enabling commute trains.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
                    .accessibilityIdentifier("commute-station-validation")
            }
        } header: {
            Text("Stations")
        } footer: {
            Text("Enter National Rail station codes. These settings are saved on this device for your account.")
        }
        .listRowBackground(AppTheme.surface)
    }

    private var countSection: some View {
        Section("Train count") {
            Stepper("Trains per direction: \(preferences.trainCount)", value: trainCountBinding, in: 1...10)
                .accessibilityIdentifier("commute-train-count")
        }
        .listRowBackground(AppTheme.surface)
    }

    private func scheduleSection(_ direction: CommuteDirection) -> some View {
        Section {
            Toggle("Show trains \(direction == .toWork ? "to work" : "to home")", isOn: scheduleBinding(direction, \.enabled))
                .accessibilityIdentifier("commute-\(direction.rawValue)-enabled")
                .disabled(!preferences.hasValidStations)
            Text(direction == .toWork ? "Home → Work" : "Work → Home")
                .font(.subheadline)
                .foregroundStyle(AppTheme.secondary)
            Group {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Days").font(.subheadline)
                    if dynamicTypeSize.isAccessibilitySize {
                        ForEach([2, 3, 4, 5, 6, 7, 1], id: \.self) { weekday in
                            Toggle(weekdays[weekday - 1], isOn: weekdayBinding(direction, weekday))
                        }
                    } else {
                        LazyVGrid(columns: [GridItem(.adaptive(minimum: 44), spacing: 0)], spacing: 4) {
                            ForEach([2, 3, 4, 5, 6, 7, 1], id: \.self) { weekday in
                                Toggle(isOn: weekdayBinding(direction, weekday)) {
                                    Text(String(weekdays[weekday - 1].prefix(1)))
                                        .frame(maxWidth: .infinity, minHeight: 44)
                                }
                                    .toggleStyle(.button)
                                    .buttonStyle(.plain)
                                    .background(preferences[direction].weekdays.contains(weekday) ? AppTheme.sage : Color.clear, in: Capsule())
                                    .accessibilityLabel(weekdays[weekday - 1])
                            }
                        }
                    }
                }
                DatePicker("Show from", selection: timeBinding(direction, \.start), displayedComponents: .hourAndMinute)
                    .accessibilityIdentifier("commute-\(direction.rawValue)-start")
                DatePicker("Hide at", selection: timeBinding(direction, \.end), displayedComponents: .hourAndMinute)
                    .accessibilityIdentifier("commute-\(direction.rawValue)-end")
                if let message = preferences[direction].validationMessage {
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.destructive)
                        .accessibilityIdentifier("commute-validation")
                }
            }
            .disabled(!preferences[direction].enabled)
        } header: {
            Text(direction.title)
        } footer: {
            if preferences[direction].start > preferences[direction].end {
                Text("This range ends the next day.")
            }
        }
        .listRowBackground(AppTheme.surface)
    }

    private func stationCodeBinding(_ keyPath: WritableKeyPath<CommutePreferences, String>) -> Binding<String> {
        Binding(get: { preferences[keyPath: keyPath] }, set: { value in
            let normalized = CommutePreferences.normalizedStationCode(value)
            preferences[keyPath: keyPath] = normalized
            savedPreferences[keyPath: keyPath] = normalized
            if !preferences.hasValidStations {
                preferences.toWork.enabled = false
                preferences.toHome.enabled = false
                savedPreferences.toWork.enabled = false
                savedPreferences.toHome.enabled = false
            }
            savePreferences()
        })
    }

    private var trainCountBinding: Binding<Int> {
        Binding(get: { preferences.trainCount }, set: { value in
            preferences.trainCount = value
            savedPreferences.trainCount = value
            savePreferences()
        })
    }

    private func scheduleBinding<Value>(_ direction: CommuteDirection, _ keyPath: WritableKeyPath<CommuteSchedule, Value>) -> Binding<Value> {
        Binding(get: { preferences[direction][keyPath: keyPath] }, set: { value in
            updateSchedule(direction) { $0[keyPath: keyPath] = value }
        })
    }

    private func weekdayBinding(_ direction: CommuteDirection, _ weekday: Int) -> Binding<Bool> {
        Binding(get: { preferences[direction].weekdays.contains(weekday) }, set: { selected in
            updateSchedule(direction) {
                if selected { $0.weekdays.insert(weekday) }
                else { $0.weekdays.remove(weekday) }
            }
        })
    }

    private func timeBinding(_ direction: CommuteDirection, _ keyPath: WritableKeyPath<CommuteSchedule, String>) -> Binding<Date> {
        Binding(get: {
            Date(timeIntervalSinceReferenceDate: Double(CommuteSchedule.minutes(preferences[direction][keyPath: keyPath]) ?? 0) * 60)
        }, set: { date in
            var calendar = Calendar(identifier: .gregorian)
            calendar.timeZone = TimeZone(secondsFromGMT: 0)!
            let components = calendar.dateComponents([.hour, .minute], from: date)
            updateSchedule(direction) {
                $0[keyPath: keyPath] = String(format: "%02d:%02d", components.hour!, components.minute!)
            }
        })
    }

    private func updateSchedule(_ direction: CommuteDirection, change: (inout CommuteSchedule) -> Void) {
        change(&preferences[direction])
        guard case .loaded(let userID) = state else { return }
        let schedule = preferences[direction]
        if !schedule.enabled {
            savedPreferences[direction].enabled = false
        } else if schedule.validationMessage == nil {
            savedPreferences[direction] = schedule
        }
        store.save(savedPreferences, server: apiClient.serverIdentifier ?? "demo", userID: userID)
    }

    private func savePreferences() {
        guard case .loaded(let userID) = state else { return }
        store.save(savedPreferences, server: apiClient.serverIdentifier ?? "demo", userID: userID)
    }

    private func load() async {
        state = .loading
        do {
            let user = try await apiClient.fetchMe()
            guard !Task.isCancelled else { return }
            savedPreferences = store.load(server: apiClient.serverIdentifier ?? "demo", userID: user.id)
            preferences = savedPreferences
            state = .loaded(user.id)
        } catch {
            guard !Task.isCancelled else { return }
            state = .failed(.from(error))
        }
    }
}
