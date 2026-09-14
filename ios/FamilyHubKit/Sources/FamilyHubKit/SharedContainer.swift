import Foundation

/// The app group shared by the iOS app, the share extension and the widget
/// extension.
///
/// The app is the only process that can sign a user in, so it mirrors the
/// server URL and the long-lived API token into this group for the extensions
/// to read. Those keys used to be spelled out at each call site — in
/// `KeychainStore`, in `ConfigStore`, and soon in the widget — which is exactly
/// the kind of string that drifts silently and leaves an extension reading a
/// key nobody writes. They live here instead.
///
/// Everything is `nonisolated`: this is plumbing over `UserDefaults` with no
/// mutable state of its own, and a widget timeline provider must be able to
/// read it without hopping to the main actor.
public enum SharedContainer {
    public nonisolated static let appGroupID = "group.uk.co.suskins.familyhub"

    public enum Key {
        /// Long-lived API token from `POST /api/auth/exchange`.
        public nonisolated static let apiToken = "api_token"
        /// Family Hub server root, e.g. `https://hub.example.com`.
        public nonisolated static let baseURL = "baseURL"
        /// JSON-encoded ``TodayWidgetData`` last published by the app.
        public nonisolated static let todaySnapshot = "today_widget_snapshot"
    }

    /// The shared defaults, or `nil` when the app group is unavailable — a
    /// missing entitlement, or a Linux CI run where suites aren't backed by a
    /// real container.
    public nonisolated static var defaults: UserDefaults? {
        UserDefaults(suiteName: appGroupID)
    }

    public nonisolated static var apiToken: String? {
        nonEmpty(defaults?.string(forKey: Key.apiToken))
    }

    /// The configured server root, or `nil` when unset or unparseable.
    public nonisolated static var baseURL: URL? {
        guard let raw = nonEmpty(defaults?.string(forKey: Key.baseURL)) else { return nil }
        return URL(string: raw)
    }

    /// Whether an extension has everything it needs to make an API call.
    /// Extensions check this to tell "not signed in yet" apart from "the
    /// request failed", which are very different things to put on a widget.
    public nonisolated static var isConfigured: Bool {
        baseURL != nil && apiToken != nil
    }

    private nonisolated static func nonEmpty(_ value: String?) -> String? {
        guard let value, !value.isEmpty else { return nil }
        return value
    }
}
