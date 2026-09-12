import SwiftUI

@main
struct FamilyHubWatchApp: App {
    @State private var session = WatchSessionStore.shared
    @State private var timers = CookTimerScheduler()

    init() {
        WatchSessionStore.shared.activate()
    }

    var body: some Scene {
        WindowGroup {
            CookRootView()
                .environment(session)
                .environment(timers)
                .task { await timers.requestAuthorization() }
        }
    }
}
