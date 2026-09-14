import XCTest
@testable import FamilyHubKit

final class StaticTokenProviderTests: XCTestCase {

    func testReturnsTheStoredToken() async throws {
        let provider = StaticTokenProvider(token: "tok-123")

        let token = try await provider.validAPIToken()

        XCTAssertEqual(token, "tok-123")
    }

    func testMissingTokenIsUnauthorizedRatherThanAnEmptyString() async {
        let provider = StaticTokenProvider(read: { nil })

        do {
            _ = try await provider.validAPIToken()
            XCTFail("expected an unauthorized error")
        } catch {
            XCTAssertEqual(error as? APIError, .unauthorized)
        }
    }

    // A cleared keychain writes "" rather than removing the key on some paths;
    // an empty bearer would reach the server as `Authorization: Bearer ` and
    // come back as an opaque 401, so it is rejected here instead.
    func testEmptyTokenIsTreatedAsMissing() async {
        let provider = StaticTokenProvider(read: { "" })

        do {
            _ = try await provider.validAPIToken()
            XCTFail("expected an unauthorized error")
        } catch {
            XCTAssertEqual(error as? APIError, .unauthorized)
        }
    }

    // The provider outlives any single request, so it must not capture the
    // token at init: signing out and back in on the phone has to be picked up
    // without rebuilding the API client.
    func testTokenIsReadFreshOnEveryRequest() async throws {
        final class Box { var value: String? = "first" }
        let box = Box()
        let provider = StaticTokenProvider(read: { box.value })

        // `try await` cannot live inside XCTAssertEqual's autoclosure.
        let first = try await provider.validAPIToken()
        XCTAssertEqual(first, "first")

        box.value = "second"
        let second = try await provider.validAPIToken()
        XCTAssertEqual(second, "second")
    }
}
