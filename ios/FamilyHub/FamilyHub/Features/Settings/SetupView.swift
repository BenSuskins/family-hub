import SwiftUI

struct SetupView: View {
    @Environment(ConfigStore.self) private var configStore

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                VStack(spacing: 0) {
                    VStack(spacing: 8) {
                        Text("Welcome to Family Hub")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundStyle(Theme.textPrimary)
                        Text("Enter your server URL to get started.")
                            .font(.system(size: 15))
                            .foregroundStyle(Theme.textSecondary)
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
            }
            .navigationTitle("Setup")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
        }
    }
}
