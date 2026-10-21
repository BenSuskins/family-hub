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
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    VStack(alignment: .leading, spacing: 16) {
                        Image(systemName: "house")
                            .font(.largeTitle)
                            .foregroundStyle(AppTheme.accent)
                            .accessibilityHidden(true)
                        Text("Your family,\nin one place.")
                            .font(.system(.largeTitle, design: .rounded, weight: .bold))
                            .accessibilityAddTraits(.isHeader)
                        Text("Plan meals. Share chores.\nKeep the day in view.")
                            .foregroundStyle(AppTheme.secondary)
                    }
                    .padding(.top, 32)

                    VStack(alignment: .leading, spacing: 12) {
                        Text("Family Hub server")
                            .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        if configStore.isConfigured {
                            Text(configStore.baseURL).textSelection(.enabled)
                        } else {
                            TextField("Server URL", text: $baseURL, prompt: Text("https://hub.example.com").foregroundStyle(AppTheme.secondary))
                                .keyboardType(.URL)
                                .autocorrectionDisabled()
                                .textInputAutocapitalization(.never)
                                .accessibilityLabel("Server URL")
                                .padding(16)
                                .groupedCard(radius: 16)
                                .submitLabel(.go)
                                .onSubmit { Task { await submit() } }
                        }
                        if let error = discoveryError ?? authManager.loginError {
                            Text(error).font(.subheadline).foregroundStyle(AppTheme.destructive)
                                .accessibilityIdentifier("Server connection error")
                        }
                        Button {
                            Task { await submit() }
                        } label: {
                            if isLoading {
                                ProgressView(configStore.isConfigured ? "Signing in" : "Connecting")
                            } else {
                                Text(configStore.isConfigured ? "Sign In" : "Connect")
                            }
                        }
                        .buttonStyle(AppPrimaryButtonStyle())
                        .disabled(!configStore.isConfigured && baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
                        Text("Ask your family admin for the server address.")
                            .font(.footnote).foregroundStyle(AppTheme.secondary)
                    }

                    VStack(spacing: 12) {
                        Button("Try Demo") { isDemoMode = true }
                            .frame(maxWidth: .infinity, minHeight: 52)
                            .background(AppTheme.sage, in: Capsule())
                            .foregroundStyle(AppTheme.accent)
                        Button("Need help connecting?") { showingSetupInfo = true }
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.accent)
                            .accessibilityIdentifier("Setup Help")
                    }
                }
                .frame(maxWidth: 520, alignment: .leading)
                .padding(20)
                .frame(maxWidth: .infinity)
            }
            .background(AppTheme.background)
            .disabled(isLoading)
            .sheet(isPresented: $showingSetupInfo) { SetupInfoSheet() }
        }
        .modifier(AccessibleMotionModifier())
    }

    @MainActor
    private func submit() async {
        guard !isLoading else { return }
        if configStore.isConfigured {
            isLoading = true
            await authManager.login(configStore: configStore)
            isLoading = false
        } else {
            await connect()
        }
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
                    Text("Join your family")
                        .font(.system(.largeTitle, design: .rounded, weight: .bold))
                        .accessibilityAddTraits(.isHeader)
                    Text("Your family admin provides the server address.")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.secondary)

                    VStack(alignment: .leading, spacing: 16) {
                        SetupStep(number: 1, title: "Ask for the address", detail: "It starts with https:// or http://.")
                        SetupStep(number: 2, title: "Connect", detail: "Enter the address on the Welcome screen and tap Connect.")
                        SetupStep(number: 3, title: "Sign in", detail: "Use the account your family admin provides.")
                    }
                    .padding(16)
                    .groupedCard()
                    Button("Back to Welcome") { dismiss() }
                        .buttonStyle(AppPrimaryButtonStyle())
                    DisclosureGroup("Set up a Family Hub server · For admins") {
                        VStack(alignment: .leading, spacing: 16) {
                            SetupStep(number: 1, title: "Deploy with Docker", detail: "Run docker compose -f docker-compose.prod.yml up from the server/ directory in the repository.")
                            SetupStep(number: 2, title: "Configure an OIDC provider", detail: "Set up a public PKCE client in Authelia, Keycloak, or Auth0. Add your app's callback URL as a redirect URI.")
                            SetupStep(number: 3, title: "Set environment variables", detail: "Required: OIDC_ISSUER, OIDC_CLIENT_ID, OIDC_REDIRECT_URL, and SESSION_SECRET. A .env.example file is included in the repository.")
                            SetupStep(number: 4, title: "Enter your server URL here", detail: "Once the server is running, enter its public URL above and tap Connect. The first user to sign in becomes admin.")
                        }

                    }
                    Link("Read the setup guide", destination: URL(string: "https://github.com/bensuskins/family-hub")!)
                    Text("The first person who signs in becomes the family admin.")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
                .frame(maxWidth: 520, alignment: .leading)
                .padding(20)
                .frame(maxWidth: .infinity)
            }
            .appListStyle()
            .navigationTitle("")
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
                    .foregroundStyle(AppTheme.secondary)
            }
        }
    }
}
