import Foundation

/// Self-contained error messaging for the Share Extension.
///
/// The extension cannot import the main app module, so this mirrors the
/// user-facing copy of the app's `APIError` to keep error messaging consistent
/// across the whole product. Maps both HTTP responses and thrown transport
/// errors to friendly, non-technical strings.
///
/// NOTE: The status→message mapping and the `cleaned`/`bodyText` helpers
/// duplicate `APIError` by necessity — an app-extension target can't depend on
/// the host app. To remove this duplication we'd extract the shared messaging
/// into a `SharedKit` framework target embedded in both the app and the
/// extension. That's a project-structure change (new target + membership) and
/// is deliberately deferred; until then, keep the copy here in sync with
/// `APIError.errorDescription`.
enum ShareAPIError {

    /// Friendly message for a non-2xx HTTP response, capturing the server's
    /// plain-text body for 4xx/5xx where it's reasonable to show. Returns `nil`
    /// for success (2xx).
    static func message(forStatus statusCode: Int, body: Data) -> String? {
        switch statusCode {
        case 200...299:
            return nil
        case 400, 422:
            return cleaned(bodyText(body)) ?? "That request couldn't be completed. Please check your input."
        case 401:
            return "Your session has expired. Open Family Hub and sign in again."
        case 403:
            return "You don't have permission to do that."
        case 404:
            return "We couldn't find what you were looking for."
        case 409:
            return "That change conflicts with the current data. Please try again."
        case 429:
            return "Too many requests. Please wait a moment and try again."
        default:
            return cleaned(bodyText(body)) ?? "Something went wrong on our end. Please try again."
        }
    }

    /// Friendly message for a thrown transport error (offline, timeout, etc.).
    static func message(for error: Error) -> String {
        guard let urlError = error as? URLError else {
            return "Couldn't reach the server. Please try again."
        }
        switch urlError.code {
        case .notConnectedToInternet, .dataNotAllowed, .internationalRoamingOff:
            return "You're offline. Check your connection and try again."
        case .timedOut:
            return "The request timed out. Please try again."
        default:
            return "Couldn't reach the server. Please try again."
        }
    }

    private static func bodyText(_ data: Data) -> String? {
        guard let text = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines),
              !text.isEmpty else { return nil }
        return text
    }

    private static func cleaned(_ message: String?) -> String? {
        guard let trimmed = message?.trimmingCharacters(in: .whitespacesAndNewlines),
              !trimmed.isEmpty,
              trimmed.count <= 200 else {
            return nil
        }
        return trimmed
    }
}
