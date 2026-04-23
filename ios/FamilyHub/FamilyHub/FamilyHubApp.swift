import SwiftUI

@main
struct FamilyHubApp: App {
    @State private var configStore = ConfigStore()
    @State private var authManager = AuthManager()

    var body: some Scene {
        WindowGroup {
            if !configStore.isConfigured {
                SetupView()
                    .environment(configStore)
                    .environment(authManager)
            } else if !authManager.isAuthenticated {
                LoginView()
                    .environment(configStore)
                    .environment(authManager)
            } else if let baseURL = URL(string: configStore.baseURL) {
                let client = APIClient(baseURL: baseURL, authManager: authManager)
                ContentView(apiClient: client)
                    .environment(configStore)
                    .environment(authManager)
            } else {
                SetupView()
                    .environment(configStore)
                    .environment(authManager)
            }
        }
    }
}
