import SwiftUI
import FamilyHubKit

@main
struct FamilyHubApp: App {
    @UIApplicationDelegateAdaptor(NotificationAppDelegate.self) private var appDelegate
    @State private var configStore = ConfigStore()
    @State private var authManager = AuthManager()
    @State private var clientStore = APIClientStore()
    @State private var demoClient = DemoAPIClient()

    init() {
        AppTheme.configureNavigationAppearance()
        // Activating early means the watch gets its credentials as soon as the
        // session is ready, without waiting for the user to visit a screen.
        PhoneWatchLink.shared.activate()
        SiriEnvironment.refreshSuggestions()
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if authManager.isDemoMode {
                    ContentView(apiClient: demoClient)
                        .environment(configStore)
                        .environment(authManager)
                } else if !configStore.isConfigured || !authManager.isAuthenticated {
                    LoginView(isDemoMode: $authManager.isDemoMode)
                        .environment(configStore)
                        .environment(authManager)
                } else if let baseURL = URL(string: configStore.baseURL) {
                    ContentView(apiClient: clientStore.client(for: baseURL, tokenProvider: authManager))
                        .environment(configStore)
                        .environment(authManager)
                } else {
                    LoginView(isDemoMode: $authManager.isDemoMode)
                        .environment(configStore)
                        .environment(authManager)
                }
            }
            .background(KeyboardDismissalTap())
            .appAppearance()
            .onChange(of: authManager.isAuthenticated) { _, isAuthenticated in
                if !isAuthenticated { Task { await clientStore.clear() } }
            }
            .onChange(of: authManager.isDemoMode) { _, enabled in
                if !enabled { demoClient = DemoAPIClient() }
            }
        }
    }
}

/// Keeps one APIClient alive across body re-evaluations.
///
/// The client owns a RecipeCache actor and a 50 MB image NSCache. Building it
/// inline in the body — as this app used to — discarded both every time SwiftUI
/// re-evaluated the scene, so nothing was ever really cached. A new client is
/// made only when the server URL actually changes.
///
/// Deliberately not @Observable: the cache is an implementation detail and
/// swapping it must not invalidate the view that just asked for it.
final class APIClientStore {
    private var cached: (baseURL: URL, client: APIClient)?

    func client(for baseURL: URL, tokenProvider: any TokenProviding) -> APIClient {
        if let cached, cached.baseURL == baseURL {
            return cached.client
        }
        if let cached { Task { await cached.client.clearAccountCache() } }
        let client = APIClient(baseURL: baseURL, tokenProvider: tokenProvider)
        cached = (baseURL, client)
        return client
    }

    func clear() async {
        await cached?.client.clearAccountCache()
        cached = nil
    }
}
