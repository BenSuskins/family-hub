import Foundation
import WidgetKit
import FamilyHubKit

/// Keeps the Today's Chores widget in step with the app.
///
/// WidgetKit budgets a widget to a few dozen timeline refreshes a day — nowhere
/// near enough to track chores being ticked off through the day. The app closes
/// that gap. It publishes a snapshot every time it loads the dashboard, and
/// tells WidgetKit to redraw whenever a chore changes, so the widget is usually
/// rendering data the app fetched rather than spending its own budget.
enum WidgetPublisher {

    /// Publish what the dashboard just returned. Called from the one place in
    /// the app that has both halves the widget needs.
    static func publish(stats: DashboardStats, users: [String: User]) {
        TodayWidgetCache.save(TodayWidgetData.from(stats: stats, users: users))
        reload()
    }

    /// A chore was completed in the app. Drop it from the snapshot so the
    /// widget matches what the user just saw, rather than waiting for the next
    /// dashboard load to correct it.
    static func completed(choreID: String) {
        if let cached = TodayWidgetCache.load() {
            TodayWidgetCache.save(cached.removing(choreID: choreID))
        }
        reload()
    }

    /// A chore was added, edited or deleted. What's due today has changed in a
    /// way the app can't patch into the snapshot, so mark it stale and let the
    /// widget refetch.
    static func invalidate() {
        TodayWidgetCache.invalidate()
        reload()
    }

    /// Sign-out. The widget must not carry on showing the family's chores to
    /// whoever picks the phone up next.
    static func clear() {
        TodayWidgetCache.clear()
        reload()
    }

    private static func reload() {
        WidgetCenter.shared.reloadTimelines(ofKind: WidgetKind.todayChores)
    }
}
