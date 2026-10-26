import SwiftUI
import UIKit
import Vision
import VisionKit
import FamilyHubKit

@MainActor @Observable
private final class DisplayManagementModel {
    var displays: [PairedDisplay] = []
    var isLoading = true
    var isSaving = false
    var message: String?

    let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) { self.apiClient = apiClient }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do { displays = try await apiClient.fetchPairedDisplays() }
        catch { message = "Paired displays could not be loaded. Try again." }
    }

    func approve(_ code: DisplayPairingCode, name: String) async {
        isSaving = true
        defer { isSaving = false }
        do {
            try await apiClient.approveDisplayPairing(id: code.challengeID, pairingCode: code.pairingCode, name: name)
            message = "Display paired with this Family Hub server."
            await load()
        } catch { message = "This pairing code expired or could not be approved. Create a new code on the display." }
    }

    func remove(_ display: PairedDisplay) async -> Bool {
        do {
            try await apiClient.deletePairedDisplay(id: display.id)
            displays.removeAll { $0.id == display.id }
            message = "\(display.name) has been removed."
            return true
        } catch {
            message = "\(display.name) could not be removed. Try again."
            return false
        }
    }

    func updateSections(_ sections: [String], for display: PairedDisplay) {
        guard let index = displays.firstIndex(where: { $0.id == display.id }) else { return }
        let current = displays[index]
        displays[index] = PairedDisplay(id: current.id, name: current.name, pairedByUserID: current.pairedByUserID, createdAt: current.createdAt, enabledSections: sections, commutePreferences: current.commutePreferences)
    }

    func updateDisplay(_ display: PairedDisplay) {
        guard let index = displays.firstIndex(where: { $0.id == display.id }) else { return }
        displays[index] = display
    }
}

struct DisplayManagementView: View {
    @Environment(ConfigStore.self) private var configStore
    @State private var model: DisplayManagementModel
    @State private var isScanning = false
    @State private var pairingCode: DisplayPairingCode?
    @State private var displayName = "Kitchen display"
    @State private var isNamingDisplay = false
    @State private var displayToRemove: PairedDisplay?
    @State private var showingRemovalConfirmation = false

    init(apiClient: any APIClientProtocol) {
        _model = State(initialValue: DisplayManagementModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            Section {
                Text("Pair a shared screen with this Family Hub server. The screen gets its own revocable access and does not use your personal sign-in.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }.listRowBackground(AppTheme.sage)

            Section("Paired displays") {
                if model.isLoading {
                    ProgressView("Loading displays…")
                } else if model.displays.isEmpty {
                    ContentUnavailableView("No displays yet", systemImage: "display", description: Text("Pair a screen to show the family day at a glance."))
                } else {
                    ForEach(model.displays) { display in
                        HStack(spacing: 12) {
                            NavigationLink {
                                PairedDisplaySettingsView(display: display, apiClient: model.apiClient, onSectionsSaved: { model.updateSections($0, for: display) }, onDisplaySaved: model.updateDisplay) {
                                    await model.remove(display)
                                }
                            } label: {
                                HStack(spacing: 12) {
                            Image(systemName: "display")
                                .foregroundStyle(AppTheme.accent)
                                .frame(width: 34, height: 34)
                                .background(AppTheme.sage, in: RoundedRectangle(cornerRadius: 9))
                            VStack(alignment: .leading, spacing: 3) {
                                Text(display.name).font(.body.weight(.medium))
                                Text("Paired \(display.createdAt.formatted(date: .abbreviated, time: .omitted))")
                                    .font(.caption).foregroundStyle(AppTheme.secondary)
                            }
                            Spacer(minLength: 4)
                                }
                            }
                            Button("Remove", role: .destructive) {
                                displayToRemove = display
                                showingRemovalConfirmation = true
                            }
                            .buttonStyle(.borderless)
                            .accessibilityLabel("Remove \(display.name)")
                            .frame(minHeight: 44)
                        }
                        .buttonStyle(.plain)
                        .listRowBackground(AppTheme.surface)
                    }
                }
            }

            if let message = model.message {
                Section { Text(message).font(.footnote).foregroundStyle(AppTheme.secondary) }
                    .listRowBackground(AppTheme.surface)
            }

            Section {
                Button {
                    isScanning = true
                } label: {
                    Label("Scan a display code", systemImage: "qrcode.viewfinder")
                        .frame(maxWidth: .infinity, minHeight: 48)
                }
                .buttonStyle(AppPrimaryButtonStyle())
                .disabled(model.isSaving)
            } footer: {
                Text("Open setup on the display first. Pairing codes expire after five minutes.")
            }
            .listRowBackground(Color.clear)
            .listRowInsets(EdgeInsets())
        }
        .listStyle(.insetGrouped)
        .appListStyle()
        .navigationTitle("Wall displays")
        .navigationBarTitleDisplayMode(.large)
        .task { await model.load() }
        .refreshable { await model.load() }
        .sheet(isPresented: $isScanning) {
            DisplayCodeScanner { value in
                isScanning = false
                receive(value)
            }
            .ignoresSafeArea()
        }
        .alert("Name this display", isPresented: $isNamingDisplay) {
            TextField("Display name", text: $displayName)
                .textInputAutocapitalization(.words)
            Button("Pair display") {
                let name = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
                guard let pairingCode, !name.isEmpty, name.utf8.count <= 60 else {
                    model.message = "Enter a display name from 1 to 60 bytes."
                    return
                }
                Task { await model.approve(pairingCode, name: name) }
            }
            Button("Cancel", role: .cancel) { pairingCode = nil }
        } message: {
            Text("Choose a name that helps everyone recognize this screen.")
        }
        .confirmationDialog("Remove this display?", isPresented: $showingRemovalConfirmation, titleVisibility: .visible) {
            Button("Remove display", role: .destructive) {
                guard let displayToRemove else { return }
                Task { await model.remove(displayToRemove) }
            }
        } message: {
            Text("It will lose access to this Family Hub server right away.")
        }
    }

    private func receive(_ value: String) {
        guard !value.isEmpty else {
            model.message = "The camera could not read that code. Scan again or create a new code on the display."
            return
        }
        guard let data = value.data(using: .utf8),
              let code = try? JSONDecoder().decode(DisplayPairingCode.self, from: data) else {
            model.message = "This QR code is not a Family Hub display code. Scan the code shown on the display."
            return
        }
        guard UUID(uuidString: code.challengeID) != nil,
              code.pairingCode.count == 64,
              code.pairingCode.allSatisfy(\.isHexDigit) else {
            model.message = "This Family Hub display code is invalid. Create a new code on the display and scan again."
            return
        }
        guard sameServer(code.apiBaseURL, configuredServer) else {
            model.message = "This code belongs to a different Family Hub server. Switch servers in settings, then scan again."
            return
        }
        pairingCode = code
        displayName = "Kitchen display"
        isNamingDisplay = true
    }

    private var configuredServer: URL? { URL(string: configStore.baseURL) }

    private func sameServer(_ first: URL, _ second: URL?) -> Bool {
        guard let second else { return false }
        return first.scheme?.lowercased() == second.scheme?.lowercased()
            && first.host?.lowercased() == second.host?.lowercased()
            && first.port == second.port
            && first.path.trimmingCharacters(in: CharacterSet(charactersIn: "/")) == second.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
            && first.user == nil && first.password == nil && first.query == nil && first.fragment == nil
    }
}

private struct PairedDisplaySettingsView: View {
    private struct SectionOption: Identifiable {
        let id: String
        let title: String
        let symbol: String
    }

    private static let sections = [
        SectionOption(id: "calendar", title: "Calendar", symbol: "calendar"),
        SectionOption(id: "meals", title: "Meals", symbol: "fork.knife"),
        SectionOption(id: "chores", title: "Chores", symbol: "checklist"),
        SectionOption(id: "weather", title: "Weather", symbol: "sun.max"),
        SectionOption(id: "departures", title: "Departures", symbol: "car"),
        SectionOption(id: "trains", title: "Trains", symbol: "tram"),
        SectionOption(id: "bins", title: "Bins", symbol: "trash"),
        SectionOption(id: "shopping", title: "Shopping", symbol: "basket"),
    ]

    let display: PairedDisplay
    let apiClient: any APIClientProtocol
    let onSectionsSaved: ([String]) -> Void
    let onDisplaySaved: (PairedDisplay) -> Void
    let onRemove: () async -> Bool
    @Environment(\.dismiss) private var dismiss
    @State private var enabledSections: Set<String>
    @State private var isSaving = false
    @State private var message = "Choose what appears on this display."
    @State private var showingRemovalConfirmation = false
    @State private var removalMessage: String?
    @State private var displayName: String
    @State private var savedDisplayName: String
    @State private var isSavingName = false
    @State private var displayNameMessage: String?

    init(display: PairedDisplay, apiClient: any APIClientProtocol, onSectionsSaved: @escaping ([String]) -> Void, onDisplaySaved: @escaping (PairedDisplay) -> Void, onRemove: @escaping () async -> Bool) {
        self.display = display
        self.apiClient = apiClient
        self.onSectionsSaved = onSectionsSaved
        self.onDisplaySaved = onDisplaySaved
        self.onRemove = onRemove
        _enabledSections = State(initialValue: Set(display.enabledSections))
        _displayName = State(initialValue: display.name)
        _savedDisplayName = State(initialValue: display.name)
    }

    var body: some View {
        List {
            Section("Display name") {
                TextField("Display name", text: $displayName)
                    .textInputAutocapitalization(.words)
                    .frame(minHeight: 44)
                if displayName.trimmingCharacters(in: .whitespacesAndNewlines).utf8.count > 60 {
                    Text("Use 60 bytes or fewer.").font(.footnote).foregroundStyle(AppTheme.destructive)
                }
                if let displayNameMessage { Text(displayNameMessage).font(.footnote).foregroundStyle(AppTheme.secondary) }
                Button(isSavingName ? "Saving…" : "Save name") { Task { await saveName() } }
                    .disabled(isSavingName || !isValidDisplayName || displayName == savedDisplayName)
                    .frame(minHeight: 44)
            }
            .listRowBackground(AppTheme.surface)

            if let removalMessage {
                Section { Text(removalMessage).font(.footnote).foregroundStyle(AppTheme.destructive) }
                    .listRowBackground(AppTheme.surface)
            }

            Section {
                HStack(spacing: 7) {
                    Image(systemName: "checkmark.circle.fill").foregroundStyle(AppTheme.accent)
                    Text(message).font(.footnote).foregroundStyle(AppTheme.secondary)
                    if isSaving { ProgressView().controlSize(.small) }
                }
                .listRowBackground(AppTheme.sage)
            }

            Section {
                ForEach(Self.sections) { section in
                    Toggle(isOn: Binding(
                        get: { enabledSections.contains(section.id) },
                        set: { setSection(section.id, enabled: $0) }
                    )) {
                        Label(section.title, systemImage: section.symbol)
                    }
                    .disabled(isSaving)
                    .frame(minHeight: 48)
                    .listRowBackground(AppTheme.surface)
                }
            } header: {
                Text("Available on this display")
            } footer: {
                Text("Choose what can appear. The display brings useful information forward as the day changes.")
            }

            Section {
                NavigationLink {
                    CommuteSettingsView(apiClient: apiClient, displayID: display.id)
                } label: {
                    HStack {
                        Label("Commute route", systemImage: "tram")
                        Spacer()
                        Text(display.commutePreferences.hasValidStations ? "Shared schedule" : "Choose stations")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.secondary)
                    }
                    .frame(minHeight: 44)
                }
                .listRowBackground(AppTheme.surface)
            }

            Section {
                Button(role: .destructive) { showingRemovalConfirmation = true } label: {
                    Label("Disconnect display", systemImage: "display.trianglebadge.exclamationmark")
                        .frame(maxWidth: .infinity, minHeight: 46, alignment: .leading)
                }
            }
        }
        .listStyle(.insetGrouped)
        .appListStyle()
        .navigationTitle(display.name)
        .navigationBarTitleDisplayMode(.large)
        .confirmationDialog("Disconnect this display?", isPresented: $showingRemovalConfirmation, titleVisibility: .visible) {
            Button("Disconnect display", role: .destructive) {
                Task {
                    if await onRemove() { dismiss() }
                    else { removalMessage = "The display is still connected. Try again." }
                }
            }
        } message: {
            Text("It will lose access to this Family Hub server right away.")
        }
    }

    private func setSection(_ section: String, enabled: Bool) {
        let previous = enabledSections
        if enabled { enabledSections.insert(section) } else { enabledSections.remove(section) }
        isSaving = true
        message = "Saving settings…"
        Task {
            do {
                try await apiClient.updatePairedDisplaySections(id: display.id, enabledSections: Self.sections.map(\.id).filter(enabledSections.contains))
                onSectionsSaved(Self.sections.map(\.id).filter(enabledSections.contains))
                message = "Settings saved to the server."
            } catch {
                enabledSections = previous
                message = "Settings could not be saved. Try again."
            }
            isSaving = false
        }
    }

    private func saveName() async {
        let name = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty, name.utf8.count <= 60 else { return }
        isSavingName = true
        defer { isSavingName = false }
        do {
            try await apiClient.updatePairedDisplayName(id: display.id, name: name)
            savedDisplayName = name
            displayNameMessage = "Name saved to this display."
            onDisplaySaved(PairedDisplay(id: display.id, name: name, pairedByUserID: display.pairedByUserID, createdAt: display.createdAt, enabledSections: display.enabledSections, commutePreferences: display.commutePreferences))
        } catch {
            displayNameMessage = "Name could not be saved. Try again."
        }
    }

    private var isValidDisplayName: Bool {
        let name = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        return !name.isEmpty && name.utf8.count <= 60
    }
}

private struct DisplayCodeScanner: UIViewControllerRepresentable {
    let onScan: (String) -> Void

    func makeCoordinator() -> Coordinator { Coordinator(onScan: onScan) }

    func makeUIViewController(context: Context) -> UIViewController {
        guard DataScannerViewController.isSupported, DataScannerViewController.isAvailable else {
            let controller = UIHostingController(rootView: Text("Camera scanning is unavailable on this iPhone."))
            return controller
        }
        let scanner = DataScannerViewController(
            recognizedDataTypes: [.barcode(symbologies: [.qr])],
            qualityLevel: .accurate,
            isHighlightingEnabled: true
        )
        scanner.delegate = context.coordinator
        do { try scanner.startScanning() }
        catch { context.coordinator.onScan("") }
        return scanner
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {}

    final class Coordinator: NSObject, DataScannerViewControllerDelegate {
        let onScan: (String) -> Void
        private var didScan = false

        init(onScan: @escaping (String) -> Void) { self.onScan = onScan }

        func dataScanner(_ dataScanner: DataScannerViewController, didAdd addedItems: [RecognizedItem], allItems: [RecognizedItem]) {
            guard !didScan else { return }
            for item in addedItems {
                if case .barcode(let barcode) = item, let value = barcode.payloadStringValue {
                    didScan = true
                    onScan(value)
                    return
                }
            }
        }
    }
}
