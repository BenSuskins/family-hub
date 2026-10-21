import SwiftUI

struct ConfigurationFormView: View {
    let configStore: ConfigStore
    let discoveryService: OIDCDiscoveryService
    let onSave: () -> Void
    let beforeSave: () async throws -> Void

    @State private var baseURL: String
    @State private var isDiscovering = false
    @State private var discoveryError: String?
    @FocusState private var addressIsFocused: Bool

    @Environment(\.dismiss) private var dismiss

    init(configStore: ConfigStore, discoveryService: OIDCDiscoveryService, beforeSave: @escaping () async throws -> Void = {}, onSave: @escaping () -> Void) {
        self.configStore = configStore
        self.discoveryService = discoveryService
        self.onSave = onSave
        self.beforeSave = beforeSave
        _baseURL = State(initialValue: configStore.baseURL)
    }

    var body: some View {
        List {
            Section {
                Text("Change the Family Hub server for this iPhone.")
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)
            Section("Server address") {
                VStack(alignment: .leading, spacing: 4) {
                    TextField("https://hub.example.com", text: $baseURL)
                        .font(.body).foregroundStyle(AppTheme.ink)
                        .focused($addressIsFocused)
                        .submitLabel(.go).onSubmit { Task { await connect() } }
                        .keyboardType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }
                .padding(.vertical, 2)

                if let error = discoveryError {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(AppTheme.destructive)
                }
            }.listRowBackground(AppTheme.surface)
            Section {
                Text("Connecting to another server signs you out. Your current server stays active if the connection fails.")
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(AppTheme.sage)
            Section {
                Button(isDiscovering ? "Connecting…" : "Connect") { Task { await connect() } }
                    .buttonStyle(AppPrimaryButtonStyle())
                    .disabled(isDiscovering || baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
            }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
        }
        .listStyle(.insetGrouped)
        .appListStyle()
        .scrollDismissesKeyboard(.interactively)
    }

    @MainActor
    private func connect() async {
        guard !isDiscovering else { return }
        addressIsFocused = false
        discoveryError = nil
        isDiscovering = true
        defer { isDiscovering = false }

        guard let url = URL(string: baseURL.trimmingCharacters(in: .whitespaces)),
              url.scheme == "http" || url.scheme == "https" else {
            discoveryError = "Enter a valid http or https URL"
            return
        }

        do {
            let result = try await discoveryService.discover(baseURL: url)
            try await beforeSave()
            configStore.baseURL = url.absoluteString
            configStore.applyDiscovery(result)
            configStore.save()
            onSave()
            dismiss()
        } catch {
            discoveryError = error.localizedDescription
        }
    }
}
