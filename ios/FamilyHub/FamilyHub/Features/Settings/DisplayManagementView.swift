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

    private let apiClient: any APIClientProtocol

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

    func remove(_ display: PairedDisplay) async {
        do {
            try await apiClient.deletePairedDisplay(id: display.id)
            displays.removeAll { $0.id == display.id }
            message = "\(display.name) has been removed."
        } catch { message = "\(display.name) could not be removed. Try again." }
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
                            Button("Remove", role: .destructive) {
                                displayToRemove = display
                                showingRemovalConfirmation = true
                            }
                            .buttonStyle(.borderless)
                            .accessibilityLabel("Remove \(display.name)")
                            .frame(minHeight: 44)
                        }
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
                guard let pairingCode, !displayName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
                Task { await model.approve(pairingCode, name: displayName.trimmingCharacters(in: .whitespacesAndNewlines)) }
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
        guard let data = value.data(using: .utf8),
              let code = try? JSONDecoder().decode(DisplayPairingCode.self, from: data),
              sameServer(code.apiBaseURL, configuredServer) else {
            model.message = "This code is for a different Family Hub server. Switch servers in settings, then scan again."
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
