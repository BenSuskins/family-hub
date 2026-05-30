import Foundation

/// Bounded exponential-backoff retry configuration for transient API failures.
///
/// Delays follow `baseDelay * 2^(attempt-1)` capped at `maxDelay`, with a small
/// random jitter to avoid thundering-herd retries. Injected into ``APIClient``
/// so tests can use near-zero delays or disable retries entirely.
struct RetryPolicy {
    var maxAttempts: Int = 3
    var baseDelay: TimeInterval = 0.5
    var maxDelay: TimeInterval = 8
    /// Fractional jitter (0...1) applied to each computed delay.
    var jitter: Double = 0.2

    static let `default` = RetryPolicy()
    static let none = RetryPolicy(maxAttempts: 1)

    /// Delay before the next attempt. `attempt` is 1-based (the attempt that just failed).
    /// When `retryAfter` is supplied (e.g. from a 429 `Retry-After` header) it takes
    /// precedence, capped at `maxDelay`.
    func delay(forAttempt attempt: Int, retryAfter: TimeInterval? = nil) -> TimeInterval {
        if let retryAfter {
            return min(retryAfter, maxDelay)
        }
        let exponential = baseDelay * pow(2, Double(attempt - 1))
        let capped = min(exponential, maxDelay)
        let jitterRange = capped * jitter
        return capped + Double.random(in: -jitterRange...jitterRange)
    }
}

/// Run `operation`, retrying transient failures per `policy`. Only retries when
/// `shouldRetry(error)` is true (the caller gates on idempotency + `isRetryable`).
/// Honors cooperative cancellation between attempts.
func withRetry<T>(
    policy: RetryPolicy,
    shouldRetry: (APIError) -> Bool,
    operation: () async throws -> T
) async throws -> T {
    var attempt = 0
    while true {
        attempt += 1
        do {
            return try await operation()
        } catch {
            let apiError = APIError.from(error)
            guard attempt < policy.maxAttempts, shouldRetry(apiError) else {
                throw apiError
            }
            var retryAfter: TimeInterval?
            if case .rateLimited(let after) = apiError { retryAfter = after }
            let seconds = policy.delay(forAttempt: attempt, retryAfter: retryAfter)
            try await Task.sleep(nanoseconds: UInt64(max(0, seconds) * 1_000_000_000))
        }
    }
}
