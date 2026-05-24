import SwiftUI

struct SetupView: View {
    @Binding var isDemoMode: Bool
    @Environment(ConfigStore.self) private var configStore

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                VStack(spacing: 8) {
                    Text("Welcome to Family Hub")
                        .font(.title2.bold())
                    Text("Enter your server URL to get started.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.top, 48)
                .padding(.horizontal)

                ConfigurationFormView(
                    configStore: configStore,
                    discoveryService: URLSessionOIDCDiscoveryService(),
                    onSave: {}
                )

                Spacer()

                Button("Try Demo") {
                    isDemoMode = true
                }
                .buttonStyle(.bordered)
                .controlSize(.large)
                .foregroundStyle(.secondary)
                .padding(.bottom, 32)
            }
            .navigationTitle("Setup")
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}
