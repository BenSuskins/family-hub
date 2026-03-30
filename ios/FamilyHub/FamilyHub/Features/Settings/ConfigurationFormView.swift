import SwiftUI

struct ConfigurationFormView: View {
    let configStore: ConfigStore
    let discoveryService: OIDCDiscoveryService
    let onSave: () -> Void

    @State private var baseURL: String
    @State private var isDiscovering = false
    @State private var discoveryError: String?

    @Environment(\.dismiss) private var dismiss

    init(configStore: ConfigStore, discoveryService: OIDCDiscoveryService, onSave: @escaping () -> Void) {
        self.configStore = configStore
        self.discoveryService = discoveryService
        self.onSave = onSave
        _baseURL = State(initialValue: configStore.baseURL)
    }

    var body: some View {
        List {
            Section("Server") {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Base URL")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("https://hub.example.com", text: $baseURL)
                        .font(.subheadline)
                        .keyboardType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }
                .padding(.vertical, 2)

                if let error = discoveryError {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
            }
        }
        .listStyle(.insetGrouped)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                if isDiscovering {
                    ProgressView()
                } else {
                    Button("Connect") {
                        Task { await connect() }
                    }
                    .disabled(baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
            }
        }
    }

    @MainActor
    private func connect() async {
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
