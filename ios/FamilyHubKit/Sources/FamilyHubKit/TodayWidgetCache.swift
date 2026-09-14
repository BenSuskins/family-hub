import Foundation

/// The ``TodayWidgetData`` handoff between the app and the widget extension.
///
/// WidgetKit gives a widget only a few dozen timeline refreshes a day, so the
/// widget should not be the only thing keeping itself current. The app writes a
/// snapshot here every time it loads the dashboard — which it does far more
/// often than the widget's budget allows — and the widget renders that
/// immediately, going to the network only when what it finds is stale. The
/// cache is also what lets the widget show real (if slightly old) data when the
/// server is unreachable, which matters for a self-hosted hub that may only be
/// on the home network.
///
/// `nonisolated` throughout: a timeline provider reads this off the main actor.
public enum TodayWidgetCache {
    /// How old a snapshot may be before the widget refetches rather than
    /// rendering it directly.
    public nonisolated static let freshFor: TimeInterval = 15 * 60

    public nonisolated static func load() -> TodayWidgetData? {
        load(from: SharedContainer.defaults)
    }

    public nonisolated static func load(from defaults: UserDefaults?) -> TodayWidgetData? {
        guard let data = defaults?.data(forKey: SharedContainer.Key.todaySnapshot) else { return nil }
        return try? JSONDecoder().decode(TodayWidgetData.self, from: data)
    }

    public nonisolated static func save(_ snapshot: TodayWidgetData) {
        save(snapshot, to: SharedContainer.defaults)
    }

    public nonisolated static func save(_ snapshot: TodayWidgetData, to defaults: UserDefaults?) {
        guard let defaults, let encoded = try? JSONEncoder().encode(snapshot) else { return }
        defaults.set(encoded, forKey: SharedContainer.Key.todaySnapshot)
    }

    /// Drop the snapshot — on sign-out, so a widget can't keep displaying the
    /// family's chores to whoever holds the phone next.
    public nonisolated static func clear() {
        clear(from: SharedContainer.defaults)
    }

    public nonisolated static func clear(from defaults: UserDefaults?) {
        defaults?.removeObject(forKey: SharedContainer.Key.todaySnapshot)
    }

    /// Stamp the cached snapshot stale without discarding it, so the next
    /// timeline pass refetches but can still fall back to real data when the
    /// hub is unreachable. Used after a chore is added, edited or deleted in
    /// the app, where the app knows the snapshot is wrong but hasn't rebuilt
    /// one.
    public nonisolated static func invalidate() {
        invalidate(in: SharedContainer.defaults)
    }

    public nonisolated static func invalidate(in defaults: UserDefaults?) {
        guard let cached = load(from: defaults) else { return }
        save(cached.markedStale(), to: defaults)
    }

    /// Whether `snapshot` is recent enough to render without refetching.
    public nonisolated static func isFresh(_ snapshot: TodayWidgetData, now: Date = Date()) -> Bool {
        let age = now.timeIntervalSince(snapshot.capturedAt)
        return age >= 0 && age < freshFor
    }
}
