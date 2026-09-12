import FamilyHubKit
import Foundation
import UserNotifications
import WatchKit

/// Runs the step timers.
///
/// Timers are scheduled as **local notifications** rather than an in-app
/// countdown, because watchOS suspends the app the moment your wrist drops —
/// which, while cooking, is most of the time. A notification fires with a haptic
/// whatever the app is doing; an in-process timer would simply not run.
///
/// There is no way to keep a watch app awake for this. `WKExtendedRuntimeSession`
/// exists, but its categories (self-care, mindfulness, physical therapy, alarm)
/// do not cover cooking, and claiming one dishonestly risks App Store rejection.
/// So: notifications for reliability, haptics in the foreground for immediacy.
@Observable @MainActor
final class CookTimerScheduler {
    /// The timer currently running, if any. One at a time — a cook is waiting
    /// for one thing at a time, and it keeps the UI unambiguous.
    private(set) var active: CookTimer?

    private let center: UNUserNotificationCenter

    init(center: UNUserNotificationCenter = .current()) {
        self.center = center
    }

    /// Ask once, up front. Declining only costs the background alert — the
    /// in-app countdown still works.
    func requestAuthorization() async {
        _ = try? await center.requestAuthorization(options: [.alert, .sound])
    }

    func start(stepIndex: Int, duration: Int, stepText: String) async {
        let timer = CookTimer(stepIndex: stepIndex, duration: duration)
        await cancel()
        active = timer

        let content = UNMutableNotificationContent()
        content.title = "Step \(stepIndex + 1) done"
        content.body = stepText
        content.sound = .default

        let request = UNNotificationRequest(
            identifier: timer.notificationID,
            content: content,
            trigger: UNTimeIntervalNotificationTrigger(timeInterval: TimeInterval(duration), repeats: false)
        )
        try? await center.add(request)

        WKInterfaceDevice.current().play(.start)
    }

    func cancel() async {
        guard let timer = active else { return }
        center.removePendingNotificationRequests(withIdentifiers: [timer.notificationID])
        active = nil
        WKInterfaceDevice.current().play(.stop)
    }

    /// Clear the timer once it has run its course, so the UI stops showing a
    /// countdown stuck at zero.
    func clearIfFired(now: Date = Date()) {
        guard let timer = active, timer.hasFired(at: now) else { return }
        active = nil
    }

    /// Whether the timer on screen belongs to the step being shown.
    func isRunning(forStep index: Int) -> Bool {
        active?.stepIndex == index
    }
}
