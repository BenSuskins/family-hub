import XCTest
@testable import FamilyHub

final class RetryPolicyTests: XCTestCase {

    // Near-zero delays so tests don't actually sleep.
    private let fastPolicy = RetryPolicy(maxAttempts: 3, baseDelay: 0, maxDelay: 0, jitter: 0)

    // MARK: - delay(forAttempt:)

    func testRetryAfterTakesPrecedenceAndIsCapped() {
        let policy = RetryPolicy(baseDelay: 0.5, maxDelay: 8)
        XCTAssertEqual(policy.delay(forAttempt: 1, retryAfter: 3), 3, accuracy: 0.001)
        XCTAssertEqual(policy.delay(forAttempt: 1, retryAfter: 100), 8, accuracy: 0.001)
    }

    func testExponentialBackoffGrowsAndCaps() {
        let policy = RetryPolicy(baseDelay: 1, maxDelay: 8, jitter: 0)
        XCTAssertEqual(policy.delay(forAttempt: 1), 1, accuracy: 0.001)
        XCTAssertEqual(policy.delay(forAttempt: 2), 2, accuracy: 0.001)
        XCTAssertEqual(policy.delay(forAttempt: 3), 4, accuracy: 0.001)
        XCTAssertEqual(policy.delay(forAttempt: 10), 8, accuracy: 0.001) // capped
    }

    // MARK: - withRetry

    func testSucceedsAfterTransientFailure() async throws {
        var attempts = 0
        let result = try await withRetry(policy: fastPolicy, shouldRetry: { $0.isRetryable }) {
            attempts += 1
            if attempts < 2 { throw APIError.server(status: 500, serverMessage: nil) }
            return "ok"
        }
        XCTAssertEqual(result, "ok")
        XCTAssertEqual(attempts, 2)
    }

    func testStopsAtMaxAttempts() async {
        var attempts = 0
        do {
            _ = try await withRetry(policy: fastPolicy, shouldRetry: { $0.isRetryable }) {
                attempts += 1
                throw APIError.timedOut
            }
            XCTFail("expected to throw")
        } catch {
            XCTAssertEqual(APIError.from(error), .timedOut)
        }
        XCTAssertEqual(attempts, 3) // maxAttempts
    }

    func testDoesNotRetryWhenShouldRetryIsFalse() async {
        var attempts = 0
        // Simulates a non-idempotent (POST) request: never retried even though the error is transient.
        do {
            _ = try await withRetry(policy: fastPolicy, shouldRetry: { _ in false }) {
                attempts += 1
                throw APIError.server(status: 500, serverMessage: nil)
            }
            XCTFail("expected to throw")
        } catch {
            XCTAssertEqual(APIError.from(error), .server(status: 500, serverMessage: nil))
        }
        XCTAssertEqual(attempts, 1)
    }

    func testNonRetryableErrorIsNotRetried() async {
        var attempts = 0
        do {
            _ = try await withRetry(policy: fastPolicy, shouldRetry: { $0.isRetryable }) {
                attempts += 1
                throw APIError.badRequest(serverMessage: "nope")
            }
            XCTFail("expected to throw")
        } catch {
            XCTAssertEqual(APIError.from(error), .badRequest(serverMessage: "nope"))
        }
        XCTAssertEqual(attempts, 1)
    }

    func testNonePolicyDisablesRetries() async {
        var attempts = 0
        do {
            _ = try await withRetry(policy: .none, shouldRetry: { $0.isRetryable }) {
                attempts += 1
                throw APIError.timedOut
            }
            XCTFail("expected to throw")
        } catch {
            XCTAssertEqual(APIError.from(error), .timedOut)
        }
        XCTAssertEqual(attempts, 1)
    }
}
