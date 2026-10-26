import XCTest
import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
@testable import FamilyHubKit

private final class AdminHTTPFixture: @unchecked Sendable {
    private let lock = NSLock()
    private var storedRequests: [URLRequest] = []
    private var storedBodies: [[UInt8]] = []

    var requests: [URLRequest] {
        lock.lock()
        defer { lock.unlock() }
        return storedRequests
    }

    func reset() {
        lock.lock()
        storedRequests.removeAll()
        storedBodies.removeAll()
        lock.unlock()
    }

    func body(at index: Int) -> [UInt8] {
        lock.lock()
        defer { lock.unlock() }
        return storedBodies[index]
    }

    func respond(_ request: URLRequest) -> (Int, Data, [String: String]) {
        lock.lock()
        storedRequests.append(request)
        storedBodies.append(Array(Self.bodyData(request)))
        lock.unlock()
        let path = request.url?.path ?? ""
        switch (request.httpMethod, path) {
        case ("GET", "/api/calendars"):
            return (200, Data(#"[{"id":"school","name":"School","url":"https://example.com/school.ics","color":"indigo","lastFetchedAt":null,"createdAt":"2026-10-22T12:00:00Z"}]"#.utf8), [:])
        case ("POST", "/api/calendars"):
            return (201, Data(#"{"id":"new","name":"School","url":"https://example.com/school.ics","color":"teal","lastFetchedAt":null,"createdAt":"2026-10-22T12:00:00Z"}"#.utf8), [:])
        case ("PATCH", "/api/calendars/school/color"):
            return (200, Data(#"{"id":"school","name":"School","url":"https://example.com/school.ics","color":"rose","lastFetchedAt":null,"createdAt":"2026-10-22T12:00:00Z"}"#.utf8), [:])
        case ("POST", "/api/calendars/school/refresh"), ("DELETE", "/api/calendars/school"), ("POST", "/api/admin/restore"):
            return (204, Data(), [:])
        case ("GET", "/api/bin-collections"), ("POST", "/api/bin-collections"):
            return (200, Data(#"{"enabled":true,"uploaded_at":"2026-10-25T12:00:00Z","collection_count":12,"push_available":true}"#.utf8), [:])
        case ("PATCH", "/api/bin-collections"):
            return (200, Data(#"{"enabled":false,"uploaded_at":"2026-10-25T12:00:00Z","collection_count":12,"push_available":true}"#.utf8), [:])
        case ("GET", "/api/displays"):
            return (200, Data(#"[{"id":"display-1","name":"Kitchen","pairedByUserId":null,"createdAt":"2026-10-25T12:00:00Z","enabledSections":["calendar"],"commutePreferences":{"toWork":{"enabled":true,"weekdays":[2,3,4,5,6],"start":"06:30","end":"09:30"},"toHome":{"enabled":false,"weekdays":[2,3,4,5,6],"start":"16:00","end":"19:30"},"homeStationCode":"KGX","workStationCode":"CBG","trainCount":4}}]"#.utf8), [:])
        case ("PATCH", "/api/displays/display-1/name"):
            return (200, Data(#"{"name":"Hall"}"#.utf8), [:])
        case ("GET", "/api/displays/display-1/commute"), ("PUT", "/api/displays/display-1/commute"):
            return (200, Data(#"{"toWork":{"enabled":true,"weekdays":[2,3,4,5,6],"start":"06:30","end":"09:30"},"toHome":{"enabled":false,"weekdays":[2,3,4,5,6],"start":"16:00","end":"19:30"},"homeStationCode":"KGX","workStationCode":"CBG","trainCount":4}"#.utf8), [:])
        case ("POST", "/api/display-pairing/challenges/challenge-1/approve"):
            return (200, Data(#"{"status":"approved"}"#.utf8), [:])
        case ("DELETE", "/api/displays/display-1"):
            return (204, Data(), [:])
        case ("GET", "/api/admin/backup"):
            return (200, Data("backup archive".utf8), ["Content-Type": "application/gzip", "Content-Disposition": "attachment; filename=family-hub-backup-2026-10-22.db.gz"])
        default:
            return (404, Data(), [:])
        }
    }

    private static func bodyData(_ request: URLRequest) -> Data {
        if let body = request.httpBody { return body }
        guard let stream = request.httpBodyStream else { return Data() }
        stream.open()
        defer { stream.close() }
        var result = Data()
        var buffer = [UInt8](repeating: 0, count: 1024)
        while stream.hasBytesAvailable {
            let count = stream.read(&buffer, maxLength: buffer.count)
            if count <= 0 { break }
            result.append(buffer, count: count)
        }
        return result
    }
}

private final class AdminURLProtocol: URLProtocol, @unchecked Sendable {
    static let fixture = AdminHTTPFixture()
    override class func canInit(with request: URLRequest) -> Bool { request.url?.host == "admin.example" }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let (status, data, headers) = Self.fixture.respond(request)
        let response = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: "HTTP/1.1", headerFields: ["Content-Type": "application/json"].merging(headers) { _, new in new })!
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}
}

final class AdminAPIClientTests: XCTestCase {
    private let provider = StaticTokenProvider(token: "fixture-token")

    func testCalendarFeedActionsUseAdminAPIContract() async throws {
        AdminURLProtocol.fixture.reset()
        let client = makeClient()
        let calendars = try await client.fetchCalendarSubscriptions()
        XCTAssertEqual(calendars.map(\.name), ["School"])
        XCTAssertEqual(calendars.first?.color, "indigo")

        let created = try await client.createCalendarSubscription(name: "School", url: "https://example.com/school.ics", color: "teal")
        XCTAssertEqual(created.id, "new")
        let recolored = try await client.updateCalendarSubscriptionColor(id: "school", color: "rose")
        XCTAssertEqual(recolored.color, "rose")
        try await client.refreshCalendarSubscription(id: "school")
        try await client.deleteCalendarSubscription(id: "school")

        let requests = AdminURLProtocol.fixture.requests
        XCTAssertEqual(requests.map(\.url!.path), ["/api/calendars", "/api/calendars", "/api/calendars/school/color", "/api/calendars/school/refresh", "/api/calendars/school"])
        let createBody = Data(AdminURLProtocol.fixture.body(at: 1))
        let payload = try XCTUnwrap(JSONSerialization.jsonObject(with: createBody) as? [String: String])
        XCTAssertEqual(payload, ["name": "School", "url": "https://example.com/school.ics", "color": "teal"])
        XCTAssertEqual(requests[2].httpMethod, "PATCH")
    }

    func testBackupDownloadsToNamedFileAndRestoreUploadsFromFile() async throws {
        AdminURLProtocol.fixture.reset()
        let client = makeClient()
        let backupURL = try await client.downloadDatabaseBackup()
        defer { try? FileManager.default.removeItem(at: backupURL) }
        XCTAssertEqual(backupURL.lastPathComponent, "family-hub-backup-2026-10-22.db.gz")
        XCTAssertEqual(try Data(contentsOf: backupURL), Data("backup archive".utf8))

        let sourceURL = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString + ".db.gz")
        try Data("restore archive".utf8).write(to: sourceURL)
        defer { try? FileManager.default.removeItem(at: sourceURL) }
        try await client.restoreDatabaseBackup(from: sourceURL)
        let request = try XCTUnwrap(AdminURLProtocol.fixture.requests.last)
        XCTAssertEqual(request.url?.path, "/api/admin/restore")
        XCTAssertTrue(request.value(forHTTPHeaderField: "Content-Type")?.contains("multipart/form-data; boundary=") == true)
        XCTAssertTrue(String(decoding: AdminURLProtocol.fixture.body(at: 1), as: UTF8.self).contains("name=\"backup\"; filename=\"" + sourceURL.lastPathComponent + "\""))
    }

    func testBinScheduleSettingsAndUploadUseAdminAPIContract() async throws {
        AdminURLProtocol.fixture.reset()
        let client = makeClient()
        let loaded = try await client.fetchBinSchedule()
        XCTAssertEqual(loaded.collectionCount, 12)
        XCTAssertTrue(loaded.enabled)

        let sourceURL = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString + ".ics")
        try Data("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n".utf8).write(to: sourceURL)
        defer { try? FileManager.default.removeItem(at: sourceURL) }
        let uploaded = try await client.uploadBinSchedule(from: sourceURL)
        XCTAssertEqual(uploaded.collectionCount, 12)
        let updated = try await client.updateBinReminders(enabled: false)
        XCTAssertFalse(updated.enabled)

        let requests = AdminURLProtocol.fixture.requests
        XCTAssertEqual(requests.map(\.httpMethod), ["GET", "POST", "PATCH"])
        XCTAssertTrue(String(decoding: AdminURLProtocol.fixture.body(at: 1), as: UTF8.self).contains("name=\"schedule\""))
        XCTAssertTrue(String(decoding: AdminURLProtocol.fixture.body(at: 1), as: UTF8.self).contains("BEGIN:VCALENDAR"))
        XCTAssertEqual(try JSONSerialization.jsonObject(with: Data(AdminURLProtocol.fixture.body(at: 2))) as? [String: Bool], ["enabled": false])
    }

    func testDisplayPairingAndRevocationUseServerContracts() async throws {
        AdminURLProtocol.fixture.reset()
        let client = makeClient()
        let displays = try await client.fetchPairedDisplays()
        XCTAssertEqual(displays.map(\.name), ["Kitchen"])
        XCTAssertEqual(displays[0].pairedByUserID, "")
        XCTAssertEqual(displays[0].commutePreferences.trainCount, 4)
        try await client.approveDisplayPairing(id: "challenge-1", pairingCode: String(repeating: "a", count: 64), name: "Hall")
        try await client.updatePairedDisplayName(id: "display-1", name: "Hall")
        let commute = try await client.fetchPairedDisplayCommute(id: "display-1")
        XCTAssertEqual(commute.homeStationCode, "KGX")
        try await client.updatePairedDisplayCommute(id: "display-1", preferences: commute)
        try await client.deletePairedDisplay(id: "display-1")

        let requests = AdminURLProtocol.fixture.requests
        XCTAssertEqual(requests.map(\.httpMethod), ["GET", "POST", "PATCH", "GET", "PUT", "DELETE"])
        let body = try XCTUnwrap(JSONSerialization.jsonObject(with: Data(AdminURLProtocol.fixture.body(at: 1))) as? [String: String])
        XCTAssertEqual(body, ["pairingCode": String(repeating: "a", count: 64), "name": "Hall"])
        XCTAssertEqual(requests[2].url?.path, "/api/displays/display-1/name")
        XCTAssertEqual(requests[4].url?.path, "/api/displays/display-1/commute")
    }

    private func makeClient() -> APIClient {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [AdminURLProtocol.self]
        let session = URLSession(configuration: configuration)
        return APIClient(baseURL: URL(string: "https://admin.example")!, session: session, tokenProvider: provider, retryPolicy: .none)
    }
}
