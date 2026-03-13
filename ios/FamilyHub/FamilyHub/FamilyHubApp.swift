import SwiftUI

@main
struct FamilyHubApp: App {
    @State private var authManager = AuthManager()

    var body: some Scene {
        WindowGroup {
            if authManager.isAuthenticated {
                let config = authManager.config
                let client = APIClient(baseURL: config.baseURL, authManager: authManager)
                ContentView(apiClient: client)
                    .environment(authManager)
            } else {
                LoginView()
                    .environment(authManager)
            }
        }
    }
}

// Temporary — replaced in Task 18
struct ContentView: View {
    let apiClient: any APIClientProtocol
    var body: some View { Text("Loading...") }
}
