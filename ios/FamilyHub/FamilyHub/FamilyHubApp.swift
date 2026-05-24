import SwiftUI

@main
struct FamilyHubApp: App {
    @State private var configStore = ConfigStore()
    @State private var authManager = AuthManager()
    @State private var isDemoMode = false

    var body: some Scene {
        WindowGroup {
            if isDemoMode {
                ContentView(apiClient: DemoAPIClient(), isDemoMode: true)
                    .environment(configStore)
                    .environment(authManager)
            } else if !configStore.isConfigured {
                SetupView(isDemoMode: $isDemoMode)
                    .environment(configStore)
                    .environment(authManager)
            } else if !authManager.isAuthenticated {
                LoginView(isDemoMode: $isDemoMode)
                    .environment(configStore)
                    .environment(authManager)
            } else if let baseURL = URL(string: configStore.baseURL) {
                let client = APIClient(baseURL: baseURL, authManager: authManager)
                ContentView(apiClient: client, isDemoMode: false)
                    .environment(configStore)
                    .environment(authManager)
            } else {
                SetupView(isDemoMode: $isDemoMode)
                    .environment(configStore)
                    .environment(authManager)
            }
        }
    }
}
