import Foundation

/// WidgetKit kind identifiers, shared so the app can reload a widget by name
/// without duplicating the string the extension registers under. Getting these
/// out of step fails silently — the reload request simply matches nothing.
public enum WidgetKind {
    public nonisolated static let todayChores = "TodayChoresWidget"
}
