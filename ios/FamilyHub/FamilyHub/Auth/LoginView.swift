import SwiftUI

struct LoginView: View {
    @Binding var isDemoMode: Bool
    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @State private var isLoading = false
    @State private var baseURL = ""
    @State private var discoveryError: String?
    @State private var showingSetupInfo = false

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Label("Family Hub", systemImage: "house.fill").font(.title.bold())
                    Text(configStore.isConfigured ? "Manage chores, meals, and more." : "Enter your server URL to connect.")
                        .foregroundStyle(.secondary)
                }
                Section("Server") {
                    if configStore.isConfigured {
                        LabeledContent("Server URL", value: configStore.baseURL)
                    } else {
                        TextField("Server URL", text: $baseURL, prompt: Text("https://hub.example.com"))
                            .keyboardType(.URL)
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.never)
                            .accessibilityLabel("Server URL")
                    }
                    if let error = discoveryError ?? authManager.loginError {
                        Text(error).foregroundStyle(.red)
                    }
                    if isLoading {
                        ProgressView(configStore.isConfigured ? "Signing in" : "Connecting")
                    } else {
                        Button(configStore.isConfigured ? "Sign In" : "Connect") {
                            Task {
                                if configStore.isConfigured {
                                    isLoading = true
                                    await authManager.login(configStore: configStore)
                                    isLoading = false
                                } else { await connect() }
                            }
                        }
                        .disabled(!configStore.isConfigured && baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                }
                Section {
                    Button("Try Demo") { isDemoMode = true }
                    Button("Setup Help", systemImage: "info.circle") { showingSetupInfo = true }
                }
            }
            .disabled(isLoading)
            .navigationTitle("Welcome")
            .sheet(isPresented: $showingSetupInfo) { SetupInfoSheet() }
        }
        .modifier(AccessibleMotionModifier())
    }

    @MainActor
    private func connect() async {
        discoveryError = nil
        isLoading = true
        defer { isLoading = false }

        guard let url = URL(string: baseURL.trimmingCharacters(in: .whitespaces)),
              url.scheme == "http" || url.scheme == "https" else {
            discoveryError = "Enter a valid http or https URL"
            return
        }

        do {
            let result = try await URLSessionOIDCDiscoveryService().discover(baseURL: url)
            configStore.baseURL = url.absoluteString
            configStore.applyDiscovery(result)
            configStore.save()
        } catch {
            discoveryError = error.localizedDescription
        }
    }
}

private struct SetupInfoSheet: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    Text("Family Hub is a self-hosted family organiser for chores, meals, recipes, and calendars. You'll need to deploy the server yourself before connecting.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)

                    Link(destination: URL(string: "https://github.com/bensuskins/family-hub")!) {
                        Label("github.com/bensuskins/family-hub", systemImage: "arrow.up.right.square")
                            .font(.subheadline.weight(.medium))
                    }

                    VStack(alignment: .leading, spacing: 16) {
                        SetupStep(number: 1, title: "Deploy with Docker", detail: "Run docker compose -f docker-compose.prod.yml up from the server/ directory in the repository.")
                        SetupStep(number: 2, title: "Configure an OIDC provider", detail: "Set up a public PKCE client in Authelia, Keycloak, or Auth0. Add your app's callback URL as a redirect URI.")
                        SetupStep(number: 3, title: "Set environment variables", detail: "Required: OIDC_ISSUER, OIDC_CLIENT_ID, OIDC_REDIRECT_URL, and SESSION_SECRET. A .env.example file is included in the repository.")
                        SetupStep(number: 4, title: "Enter your server URL here", detail: "Once the server is running, enter its public URL above and tap Connect. The first user to sign in becomes admin.")
                    }

                    Text("Full documentation is available in the GitHub repository.")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
                .padding()
            }
            .navigationTitle("Setting Up Family Hub")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
    }
}

private struct SetupStep: View {
    let number: Int
    let title: String
    let detail: String

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Text("\(number)")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.white)
                .frame(width: 22, height: 22)
                .background(Color.accentColor)
                .clipShape(Circle())

            VStack(alignment: .leading, spacing: 3) {
                Text(title)
                    .font(.subheadline.weight(.medium))
                Text(detail)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }
}
