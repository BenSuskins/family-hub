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
                .task {
                    guard !ProcessInfo.processInfo.arguments.contains("-marketing-screenshots") else { return }
                    await timers.requestAuthorization()
                }
        }
    }
}
