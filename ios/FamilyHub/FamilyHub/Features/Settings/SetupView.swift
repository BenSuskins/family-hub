import SwiftUI

struct SetupView: View {
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
            }
            .navigationTitle("Setup")
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}
