import XCTest
import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
@testable import FamilyHubKit

private final class ChoreHTTPFixture: @unchecked Sendable {
    private let lock = NSLock()
    private var calls: [String] = []
    private let definition = #"{"ID":"bins","Chore":{"ID":"bins","Name":"Bins","Description":"","Status":"pending"},"CurrentTask":null,"Active":true,"DeletedAt":null}"#

    func respond(_ request: URLRequest) throws -> (Int, Data) {
        lock.lock()
        defer { lock.unlock() }
        guard request.value(forHTTPHeaderField: "Authorization") == "Bearer chore-token" else { return (401, Data()) }
        let path = request.url!.path
        let method = request.httpMethod!
        calls.append("\(method) \(path)")
        let query = URLComponents(url: request.url!, resolvingAgainstBaseURL: false)?.queryItems ?? []
        if path == "/api/chore-history" {
            guard query.contains(URLQueryItem(name: "definition_id", value: "bins")),
                  query.contains(URLQueryItem(name: "cursor", value: "previous")) else { return (400, Data()) }
            return (200, Data(#"{"Items":[],"NextCursor":""}"#.utf8))
        }
        if method == "DELETE" || path.hasSuffix("/skip") { return (204, Data()) }
        if method == "GET" {
            guard query.contains(URLQueryItem(name: "include_inactive", value: "true")) else { return (400, Data()) }
            return (200, Data("[\(definition)]".utf8))
        }
        return (method == "POST" ? 201 : 200, Data(definition.utf8))
    }

    func requests() -> [String] {
        lock.lock()
        defer { lock.unlock() }
        return calls
    }
}

private final class ChoreURLProtocol: URLProtocol, @unchecked Sendable {
    static let fixture = ChoreHTTPFixture()
    override class func canInit(with request: URLRequest) -> Bool { request.url?.host == "chores.example" }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        do {
            let (status, data) = try Self.fixture.respond(request)
            let response = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: "HTTP/1.1", headerFields: ["Content-Type": "application/json"])!
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch { client?.urlProtocol(self, didFailWithError: error) }
    }
    override func stopLoading() {}
}

final class ChoreAPIClientTests: XCTestCase {
    func testDefinitionHistoryAndSkipUseTheAuthenticatedContract() async throws {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [ChoreURLProtocol.self]
        let session = URLSession(configuration: configuration)
        defer { session.invalidateAndCancel() }
        let provider = StaticTokenProvider(token: "chore-token")
        let client = APIClient(baseURL: URL(string: "https://chores.example")!, session: session, tokenProvider: provider, retryPolicy: .none)
        let definitions = try await client.fetchChoreDefinitions(includeInactive: true)
        XCTAssertEqual(definitions.first?.id, "bins")
        let request = ChoreRequest(name: "Bins", description: "", assignees: [])
        _ = try await client.createChoreDefinition(request)
        _ = try await client.updateChoreDefinition(id: "bins", request)
        try await client.skipChore(id: "task")
        let history = try await client.fetchChoreHistory(definitionID: "bins", cursor: "previous")
        XCTAssertTrue(history.items.isEmpty)
        try await client.deleteChoreDefinition(id: "bins")
        XCTAssertEqual(ChoreURLProtocol.fixture.requests(), ["GET /api/chore-definitions", "POST /api/chore-definitions", "PUT /api/chore-definitions/bins", "POST /api/chores/task/skip", "GET /api/chore-history", "DELETE /api/chore-definitions/bins"])
    }
}
