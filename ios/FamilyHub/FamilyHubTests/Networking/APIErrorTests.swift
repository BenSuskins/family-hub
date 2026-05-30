import XCTest
@testable import FamilyHub

final class APIErrorTests: XCTestCase {

    // MARK: - isRetryable

    func testTransientErrorsAreRetryable() {
        XCTAssertTrue(APIError.offline.isRetryable)
        XCTAssertTrue(APIError.timedOut.isRetryable)
        XCTAssertTrue(APIError.rateLimited(retryAfter: nil).isRetryable)
        XCTAssertTrue(APIError.server(status: 500, serverMessage: nil).isRetryable)
        XCTAssertTrue(APIError.server(status: 503, serverMessage: "down").isRetryable)
        XCTAssertTrue(APIError.network(URLError(.networkConnectionLost)).isRetryable)
    }

    func testPermanentErrorsAreNotRetryable() {
        XCTAssertFalse(APIError.unauthorized.isRetryable)
        XCTAssertFalse(APIError.forbidden.isRetryable)
        XCTAssertFalse(APIError.notFound.isRetryable)
        XCTAssertFalse(APIError.conflict.isRetryable)
        XCTAssertFalse(APIError.badRequest(serverMessage: "bad").isRetryable)
        XCTAssertFalse(APIError.decoding.isRetryable)
        XCTAssertFalse(APIError.server(status: 404, serverMessage: nil).isRetryable)
        XCTAssertFalse(APIError.network(URLError(.badURL)).isRetryable)
    }

    // MARK: - from(_:)

    func testFromMapsURLErrorCodes() {
        XCTAssertEqual(APIError.from(URLError(.notConnectedToInternet)), .offline)
        XCTAssertEqual(APIError.from(URLError(.dataNotAllowed)), .offline)
        XCTAssertEqual(APIError.from(URLError(.timedOut)), .timedOut)
        XCTAssertEqual(APIError.from(URLError(.cannotFindHost)), .network(URLError(.cannotFindHost)))
    }

    func testFromPassesThroughExistingAPIError() {
        let original = APIError.conflict
        XCTAssertEqual(APIError.from(original), .conflict)
    }

    func testFromWrapsUnknownErrors() {
        struct SomeError: Error {}
        XCTAssertEqual(APIError.from(SomeError()), .network(URLError(.unknown)))
    }

    // MARK: - Messaging

    func testEveryCaseHasUserFacingDescription() {
        let cases: [APIError] = [
            .offline, .timedOut, .network(URLError(.unknown)), .unauthorized,
            .forbidden, .notFound, .conflict, .badRequest(serverMessage: nil),
            .rateLimited(retryAfter: nil), .server(status: 500, serverMessage: nil), .decoding,
        ]
        for error in cases {
            XCTAssertFalse(error.errorDescription?.isEmpty ?? true, "missing copy for \(error)")
        }
    }

    func testServerMessageIsSurfacedWhenReasonable() {
        let error = APIError.badRequest(serverMessage: "name is required")
        XCTAssertEqual(error.errorDescription, "name is required")
    }

    func testOverlongServerMessageFallsBackToGenericCopy() {
        let huge = String(repeating: "x", count: 500)
        let error = APIError.server(status: 500, serverMessage: huge)
        XCTAssertEqual(error.errorDescription, "Something went wrong on our end. Please try again.")
    }

    func testEmptyServerMessageFallsBackToGenericCopy() {
        let error = APIError.badRequest(serverMessage: "   ")
        XCTAssertEqual(error.errorDescription, "That request couldn't be completed. Please check your input.")
    }
}
