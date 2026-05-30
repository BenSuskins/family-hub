import Foundation

/// Unified error type for every API call in the app.
///
/// Cases classify failures so the UI can show friendly, non-technical copy and
/// the networking layer can decide whether a request is worth retrying. Use
/// ``APIError/from(_:)`` to normalize any thrown `Error` into an `APIError`.
enum APIError: Error, LocalizedError, Equatable {
    /// Device is offline or the host is unreachable.
    case offline
    /// The request exceeded its timeout.
    case timedOut
    /// Other transport-level failure (carries the underlying `URLError`).
    case network(URLError)
    /// 401 — token missing/expired. Triggers a global sign-out.
    case unauthorized
    /// 403 — authenticated but not allowed.
    case forbidden
    /// 404 — resource not found.
    case notFound
    /// 409 — conflicting state.
    case conflict
    /// 400 / 422 — invalid request. Carries the server's plain-text message.
    case badRequest(serverMessage: String?)
    /// 429 — rate limited. Carries the parsed `Retry-After` delay if present.
    case rateLimited(retryAfter: TimeInterval?)
    /// 5xx and any other unexpected status. Carries status + server message.
    case server(status: Int, serverMessage: String?)
    /// The response body could not be decoded into the expected type.
    case decoding

    /// `URLError` codes that represent transient connectivity blips worth retrying.
    private static let transientURLCodes: Set<URLError.Code> = [
        .networkConnectionLost,
        .cannotConnectToHost,
        .cannotFindHost,
        .dnsLookupFailed,
        .resourceUnavailable,
    ]

    /// Whether automatically retrying the request could plausibly succeed.
    /// Only meaningful for idempotent requests — the caller gates on the HTTP method.
    var isRetryable: Bool {
        switch self {
        case .offline, .timedOut, .rateLimited:
            return true
        case .server(let status, _):
            return status >= 500
        case .network(let error):
            return Self.transientURLCodes.contains(error.code)
        case .unauthorized, .forbidden, .notFound, .conflict, .badRequest, .decoding:
            return false
        }
    }

    var errorDescription: String? {
        switch self {
        case .offline:
            return "You're offline. Check your connection and try again."
        case .timedOut:
            return "The request timed out. Please try again."
        case .network:
            return "Couldn't reach the server. Please try again."
        case .unauthorized:
            return "Your session has expired. Please sign in again."
        case .forbidden:
            return "You don't have permission to do that."
        case .notFound:
            return "We couldn't find what you were looking for."
        case .conflict:
            return "That change conflicts with the current data. Refresh and try again."
        case .badRequest(let serverMessage):
            return Self.cleaned(serverMessage) ?? "That request couldn't be completed. Please check your input."
        case .rateLimited:
            return "Too many requests. Please wait a moment and try again."
        case .server(_, let serverMessage):
            return Self.cleaned(serverMessage) ?? "Something went wrong on our end. Please try again."
        case .decoding:
            return "We received an unexpected response. Please try again."
        }
    }

    var recoverySuggestion: String? {
        switch self {
        case .offline:
            return "Make sure Wi-Fi or mobile data is turned on."
        case .unauthorized:
            return "You'll be returned to the sign-in screen."
        default:
            return nil
        }
    }

    /// Normalize any thrown error into an `APIError`, mapping `URLError` codes to
    /// the appropriate transport case. Already-`APIError` values pass through.
    static func from(_ error: Error) -> APIError {
        if let apiError = error as? APIError {
            return apiError
        }
        if let urlError = error as? URLError {
            switch urlError.code {
            case .notConnectedToInternet, .dataNotAllowed, .internationalRoamingOff:
                return .offline
            case .timedOut:
                return .timedOut
            default:
                return .network(urlError)
            }
        }
        return .network(URLError(.unknown))
    }

    /// Trim a server-provided message and reject anything empty or implausibly
    /// long (so we never dump a stray HTML page or stack trace at the user).
    private static func cleaned(_ message: String?) -> String? {
        guard let trimmed = message?.trimmingCharacters(in: .whitespacesAndNewlines),
              !trimmed.isEmpty,
              trimmed.count <= 200 else {
            return nil
        }
        return trimmed
    }

    // MARK: - Equatable

    static func == (lhs: APIError, rhs: APIError) -> Bool {
        switch (lhs, rhs) {
        case (.offline, .offline),
             (.timedOut, .timedOut),
             (.unauthorized, .unauthorized),
             (.forbidden, .forbidden),
             (.notFound, .notFound),
             (.conflict, .conflict),
             (.decoding, .decoding):
            return true
        case let (.network(a), .network(b)):
            return a.code == b.code
        case let (.badRequest(a), .badRequest(b)):
            return a == b
        case let (.rateLimited(a), .rateLimited(b)):
            return a == b
        case let (.server(sa, ma), .server(sb, mb)):
            return sa == sb && ma == mb
        default:
            return false
        }
    }
}
