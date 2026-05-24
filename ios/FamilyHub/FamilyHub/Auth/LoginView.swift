import SwiftUI

struct LoginView: View {
    @Binding var isDemoMode: Bool
    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @State private var isLoading = false
    @State private var appeared = false
    @State private var baseURL: String = ""
    @State private var discoveryError: String?

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            VStack(spacing: 16) {
                Image(systemName: "house.fill")
                    .font(.system(size: 56))
                    .foregroundStyle(.tint)
                    .symbolEffect(.pulse, options: .repeating.speed(0.5), isActive: isLoading)

                Text("Family Hub")
                    .font(.largeTitle.bold())

                Text(configStore.isConfigured ? "Manage chores, meals, and more." : "Enter your server URL to get started.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .opacity(appeared ? 1 : 0)
            .offset(y: appeared ? 0 : 20)

            Spacer()

            VStack(spacing: 12) {
                if configStore.isConfigured {
                    if let error = authManager.loginError {
                        Text(error)
                            .foregroundStyle(.red)
                            .font(.caption)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)
                    }

                    Button {
                        Task {
                            isLoading = true
                            await authManager.login(configStore: configStore)
                            isLoading = false
                        }
                    } label: {
                        if isLoading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            Text("Sign In")
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    .disabled(isLoading)
                } else {
                    TextField("https://hub.example.com", text: $baseURL)
                        .textFieldStyle(.roundedBorder)
                        .keyboardType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)

                    if let error = discoveryError {
                        Text(error)
                            .foregroundStyle(.red)
                            .font(.caption)
                            .multilineTextAlignment(.center)
                    }

                    Button {
                        Task { await connect() }
                    } label: {
                        if isLoading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            Text("Connect")
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    .disabled(isLoading || baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
                }

                Button("Try Demo") {
                    isDemoMode = true
                }
                .buttonStyle(.bordered)
                .controlSize(.large)
                .foregroundStyle(.secondary)
                .disabled(isLoading)
            }
            .padding(.horizontal)
            .opacity(appeared ? 1 : 0)
        }
        .padding()
        .background(.regularMaterial)
        .onAppear {
            withAnimation(.spring(duration: 0.6)) {
                appeared = true
            }
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
