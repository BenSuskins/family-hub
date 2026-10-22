import XCTest
import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
@testable import FamilyHubKit

private final class CommuteURLProtocol: URLProtocol, @unchecked Sendable {
    override class func canInit(with request: URLRequest) -> Bool { request.url?.host == "commute.example" }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let query = URLComponents(url: request.url!, resolvingAgainstBaseURL: false)?.queryItems ?? []
        let direction = query.first { $0.name == "direction" }?.value ?? ""
        let count = query.first { $0.name == "count" }?.value
        let authorized = request.value(forHTTPHeaderField: "Authorization") == "Bearer commute-fixture"
        let valid = request.httpMethod == "GET" && request.url?.path == "/api/commute/trains" && count == "3"
        let status = !authorized ? 401 : valid ? 200 : 503
        let home = direction == "to_home"
        let response = Self.fixture.replacingOccurrences(of: "DIRECTION", with: direction)
            .replacingOccurrences(of: "ORIGIN_CODE", with: home ? "LST" : "KEL")
            .replacingOccurrences(of: "ORIGIN_NAME", with: home ? "London Liverpool Street" : "Kelvedon")
            .replacingOccurrences(of: "DESTINATION_CODE", with: home ? "KEL" : "LST")
            .replacingOccurrences(of: "DESTINATION_NAME", with: home ? "Kelvedon" : "London Liverpool Street")
        let data = Data(response.utf8)
        client?.urlProtocol(self, didReceive: HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: nil, headerFields: ["Content-Type": "application/json"])!, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}

    private static let fixture = #"""
    {"direction":"DIRECTION","origin":{"code":"ORIGIN_CODE","name":"ORIGIN_NAME"},"destination":{"code":"DESTINATION_CODE","name":"DESTINATION_NAME"},"generatedAt":"2026-10-21T07:30:00+01:00","fetchedAt":"2026-10-21T06:30:01.123456Z","timezone":"Europe/London","trains":[
      {"serviceId":"late","operator":"Greater Anglia","platform":"1","status":"delayed","scheduledDeparture":"08:03","expectedDeparture":"08:08","scheduledArrival":"08:58","expectedArrival":"09:03","cancellationReason":null,"delayReason":"A signal fault"},
      {"serviceId":"cancelled","operator":"Greater Anglia","platform":null,"status":"cancelled","scheduledDeparture":"08:21","expectedDeparture":null,"scheduledArrival":"09:15","expectedArrival":null,"cancellationReason":"A train fault","delayReason":null},
      {"serviceId":"unknown","operator":"Greater Anglia","platform":null,"status":"unknown","scheduledDeparture":null,"expectedDeparture":null,"scheduledArrival":null,"expectedArrival":null,"cancellationReason":null,"delayReason":null}
    ]}
    """#
}

final class CommuteAPIClientTests: XCTestCase {
    @MainActor
    func testBothDirectionsUseTheAuthenticatedTrainContract() async throws {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [CommuteURLProtocol.self]
        let session = URLSession(configuration: configuration)
        defer { session.invalidateAndCancel() }
        let provider = StaticTokenProvider(token: "commute-fixture")
        let client = APIClient(baseURL: URL(string: "https://commute.example")!, session: session, tokenProvider: provider, retryPolicy: .none)
        for direction in CommuteDirection.allCases {
            let board = try await client.fetchCommuteTrains(direction: direction, count: 3)
            XCTAssertEqual(board.direction, direction)
            XCTAssertEqual(board.origin.code, direction == .toWork ? "KEL" : "LST")
            XCTAssertEqual(board.destination.code, direction == .toWork ? "LST" : "KEL")
            XCTAssertEqual(board.trains.count, 3)
            XCTAssertEqual(board.trains[0].departure.original, "08:03")
            XCTAssertEqual(board.trains[1].statusLabel, "Cancelled")
            XCTAssertNil(board.trains[1].platform)
            XCTAssertEqual(board.trains[2].departure.display, "Time unavailable")
            XCTAssertNotNil(APIDate.parse(board.fetchedAt))
            XCTAssertEqual(board.fetchedDate?.timeIntervalSince1970 ?? 0, 1_792_564_201.123456, accuracy: 0.001)
        }
        do {
            _ = try await client.fetchCommuteTrains(direction: .toWork, count: 1)
            XCTFail("Expected the provider failure to reach the caller")
        } catch {
            guard case .server(let status, _) = APIError.from(error) else { return XCTFail("Expected a server error") }
            XCTAssertEqual(status, 503)
        }
    }

    @MainActor
    func testLiveCommuteContractWhenConfigured() async throws {
        let environment = ProcessInfo.processInfo.environment
        guard let base = environment["FAMILY_HUB_COMMUTE_TEST_URL"],
              let token = environment["FAMILY_HUB_COMMUTE_TEST_TOKEN"], let url = URL(string: base) else {
            throw XCTSkip("Set the commute test URL and token to check the live API.")
        }
        let provider = StaticTokenProvider(token: token)
        let client = APIClient(baseURL: url, tokenProvider: provider, retryPolicy: .none)
        for direction in CommuteDirection.allCases {
            let board = try await client.fetchCommuteTrains(direction: direction, count: 3)
            XCTAssertEqual(board.direction, direction)
            XCTAssertLessThanOrEqual(board.trains.count, 3)
            XCTAssertEqual(board.timezone, "Europe/London")
            XCTAssertNotNil(APIDate.parse(board.fetchedAt))
        }
    }
}
