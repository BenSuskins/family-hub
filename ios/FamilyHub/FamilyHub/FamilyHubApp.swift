import SwiftUI

@main
struct FamilyHubApp: App {
    @State private var configStore = ConfigStore()
    @State private var authManager = AuthManager()

    var body: some Scene {
        WindowGroup {
            if authManager.isDemoMode {
                ContentView(apiClient: DemoAPIClient())
                    .environment(configStore)
                    .environment(authManager)
            } else if !configStore.isConfigured || !authManager.isAuthenticated {
                LoginView(isDemoMode: $authManager.isDemoMode)
                    .environment(configStore)
                    .environment(authManager)
            } else if let baseURL = URL(string: configStore.baseURL) {
                let client = APIClient(baseURL: baseURL, authManager: authManager)
                ContentView(apiClient: client)
                    .environment(configStore)
                    .environment(authManager)
            } else {
                LoginView(isDemoMode: $authManager.isDemoMode)
                    .environment(configStore)
                    .environment(authManager)
            }
        }
    }
}
