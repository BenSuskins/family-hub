import Foundation
import FamilyHubKit

/// Everything the widget needs to reach the server, assembled from the app
/// group the main app writes to.
///
/// The widget can never sign anyone in — OIDC needs a browser sheet and there
/// is no UI here to present one — so an unconfigured extension is a normal
/// state to render, not an error to report.
enum WidgetEnvironment {

    /// `APIClient` holds its token provider **weakly**: it normally points at
    /// the app's `AuthManager`, where a strong reference would be a retain
    /// cycle. The widget's provider has no other owner, so it is held here for
    /// the life of the extension process. Without this the provider would be
    /// deallocated the moment ``client()`` returned and every request would
    /// fail unauthorized.
    private static let tokenProvider = StaticTokenProvider()

    /// One client per server URL, kept so its caches survive across the
    /// timeline reloads that happen within a single extension launch.
    private static var cached: (baseURL: URL, client: APIClient)?

    /// The API client, or `nil` when the app has not been configured and
    /// signed in yet.
    static func client() -> APIClient? {
        guard SharedContainer.isConfigured, let baseURL = SharedContainer.baseURL else {
            return nil
        }
        if let cached, cached.baseURL == baseURL {
            return cached.client
        }
        let client = APIClient(baseURL: baseURL, tokenProvider: tokenProvider)
        cached = (baseURL, client)
        return client
    }

    /// Fetch a fresh day, or `nil` when unconfigured or the server could not be
    /// reached. Callers fall back to the cached snapshot; a self-hosted hub is
    /// often unreachable away from home, so that is an ordinary outcome rather
    /// than a failure worth surfacing.
    ///
    /// Three requests in parallel. Only the dashboard is load-bearing: it
    /// carries the outstanding chores and today's meals, and there is no
    /// honest day to draw without it. The calendar and the user list each
    /// degrade a part of the widget rather than the whole of it, so a failure
    /// in either is swallowed — a day with its events missing still beats
    /// "can't reach the hub".
    static func fetchSnapshot() async -> TodayWidgetData? {
        guard let client = client() else { return nil }

        async let statsTask = client.fetchDashboardStats()
        async let calendarTask = client.fetchCalendar(view: "day", date: Date())
        async let usersTask = client.fetchUsers()
        do {
            let stats = try await statsTask
            let events = (try? await calendarTask)?.events ?? []
            let users = (try? await usersTask) ?? []
            return TodayWidgetData.from(stats: stats, events: events, users: users.keyedByID)
        } catch {
            return nil
        }
    }
}
