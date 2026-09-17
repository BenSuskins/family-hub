import Foundation

/// WidgetKit kind identifiers, shared so the app can reload a widget by name
/// without duplicating the string the extension registers under. Getting these
/// out of step fails silently — the reload request simply matches nothing.
public enum WidgetKind {
    /// The combined day view: agenda, chores and the meal plan.
    ///
    /// Renamed from the chores-only widget's kind, so anyone who added that one
    /// during its single release has to add this instead. Reusing the old
    /// string would have been worse: an installed widget would keep its
    /// "Today's Chores" gallery description while drawing a different thing.
    public nonisolated static let today = "FamilyHubTodayWidget"
}
