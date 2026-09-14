import Foundation

/// A ``TokenProviding`` for processes that cannot run the OIDC flow.
///
/// `AuthManager` is the app's provider, but it drives
/// `ASWebAuthenticationSession` and lives in the app target. An extension has
/// no UI to present a login sheet and no business starting one, so it reads the
/// long-lived API token the app already mirrored into the shared app group and
/// fails cleanly when it isn't there — leaving the widget to render a "sign in"
/// state rather than an error.
///
/// The token is read on every request rather than captured once, so signing out
/// or re-logging in on the phone is picked up without rebuilding the client.
public final class StaticTokenProvider: TokenProviding {
    private let read: () -> String?

    public init(read: @escaping () -> String?) {
        self.read = read
    }

    /// Fixed token. Useful in tests and previews.
    public convenience init(token: String) {
        self.init(read: { token })
    }

    /// Reads whatever the app last mirrored into the shared app group.
    public convenience init() {
        self.init(read: { SharedContainer.apiToken })
    }

    public func validAPIToken() async throws -> String {
        guard let token = read(), !token.isEmpty else {
            throw APIError.unauthorized
        }
        return token
    }
}
