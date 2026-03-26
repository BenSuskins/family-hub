import SwiftUI

struct ConfigurationFormView: View {
    let configStore: ConfigStore
    let onSave: () -> Void

    @State private var baseURL: String
    @State private var clientID: String
    @State private var authorizationEndpoint: String
    @State private var tokenEndpoint: String

    @Environment(\.dismiss) private var dismiss

    init(configStore: ConfigStore, onSave: @escaping () -> Void) {
        self.configStore = configStore
        self.onSave = onSave
        _baseURL = State(initialValue: configStore.baseURL)
        _clientID = State(initialValue: configStore.clientID)
        _authorizationEndpoint = State(initialValue: configStore.authorizationEndpoint)
        _tokenEndpoint = State(initialValue: configStore.tokenEndpoint)
    }

    var body: some View {
        List {
            Section("Server") {
                configField(label: "Base URL", placeholder: "https://hub.example.com", text: $baseURL, isURL: true)
            }
            .listRowBackground(Theme.surface)

            Section("OIDC") {
                configField(label: "Client ID", placeholder: "familyhub-ios", text: $clientID, isURL: false)
                configField(label: "Authorization Endpoint", placeholder: "https://auth.example.com/…/authorize", text: $authorizationEndpoint, isURL: true)
                configField(label: "Token Endpoint", placeholder: "https://auth.example.com/…/token", text: $tokenEndpoint, isURL: true)
            }
            .listRowBackground(Theme.surface)
        }
        .scrollContentBackground(.hidden)
        .listStyle(.insetGrouped)
        .background(Theme.background)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") {
                    configStore.baseURL = baseURL
                    configStore.clientID = clientID
                    configStore.authorizationEndpoint = authorizationEndpoint
                    configStore.tokenEndpoint = tokenEndpoint
                    onSave()
                    dismiss()
                }
                .foregroundStyle(Theme.accent)
            }
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
                    .foregroundStyle(Theme.accent)
            }
        }
    }

    @ViewBuilder
    private func configField(label: String, placeholder: String, text: Binding<String>, isURL: Bool) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(Theme.textSecondary)
            if isURL {
                TextField(placeholder, text: text)
                    .font(.system(size: 14))
                    .foregroundStyle(Theme.textPrimary)
                    .keyboardType(.URL)
                    .autocorrectionDisabled()
                    .textInputAutocapitalization(.never)
            } else {
                TextField(placeholder, text: text)
                    .font(.system(size: 14))
                    .foregroundStyle(Theme.textPrimary)
                    .autocorrectionDisabled()
                    .textInputAutocapitalization(.never)
            }
        }
        .padding(.vertical, 2)
    }
}
