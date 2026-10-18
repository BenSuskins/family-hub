import XCTest
import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
@testable import FamilyHubKit

private final class ReminderHTTPFixture: @unchecked Sendable {
    private let lock = NSLock()
    private var settings = ChoreReminders(pushAvailable: true)
    private var devices: [String: String] = [:]

    func respond(_ request: URLRequest) throws -> (Int, Data) {
        lock.lock()
        defer { lock.unlock() }
        guard request.value(forHTTPHeaderField: "Authorization") == "Bearer fixture-token" else { return (401, Data()) }
        if request.url?.path == "/api/chore-reminders" {
            if request.httpMethod == "PATCH" {
                let body = try JSONSerialization.jsonObject(with: bodyData(request)) as! [String: Any]
                settings.enabled = body["enabled"] as! Bool
                settings.times = body["times"] as! [String]
            }
            return (200, try JSONEncoder().encode(settings))
        }
        let identifier = request.url!.lastPathComponent
        switch request.httpMethod {
        case "PUT":
            let body = try JSONSerialization.jsonObject(with: bodyData(request)) as! [String: Any]
            guard body["environment"] as? String == "sandbox", let token = body["token"] as? String else { return (400, Data()) }
            devices[identifier] = token
        case "DELETE": devices.removeValue(forKey: identifier)
        default: return (404, Data())
        }
        return (204, Data())
    }

    func token(for identifier: String) -> String? {
        lock.lock()
        defer { lock.unlock() }
        return devices[identifier]
    }

    private func bodyData(_ request: URLRequest) -> Data {
        if let data = request.httpBody { return data }
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

private final class ReminderURLProtocol: URLProtocol, @unchecked Sendable {
    static let fixture = ReminderHTTPFixture()
    override class func canInit(with request: URLRequest) -> Bool { request.url?.host == "reminders.example" }
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

final class ReminderAPIClientTests: XCTestCase {
    func testSettingsAndDeviceLifecycleUseTheAuthenticatedWireContract() async throws {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [ReminderURLProtocol.self]
        let session = URLSession(configuration: configuration)
        defer { session.invalidateAndCancel() }
        let provider = StaticTokenProvider(token: "fixture-token")
        let client = APIClient(baseURL: URL(string: "https://reminders.example")!, session: session, tokenProvider: provider, retryPolicy: .none)
        var settings = try await client.fetchChoreReminders()
        XCTAssertEqual(settings.times, ["10:00", "16:30"])
        settings.enabled = true
        settings.times = ["09:00", "18:00"]
        try await client.updateChoreReminders(settings)
        let updated = try await client.fetchChoreReminders()
        XCTAssertEqual(updated, settings)
        try await client.registerNotificationDevice(NotificationDevice(id: "phone", token: "abcd", environment: "sandbox"))
        XCTAssertEqual(ReminderURLProtocol.fixture.token(for: "phone"), "abcd")
        try await client.registerNotificationDevice(NotificationDevice(id: "phone", token: "ef01", environment: "sandbox"))
        XCTAssertEqual(ReminderURLProtocol.fixture.token(for: "phone"), "ef01")
        try await client.removeNotificationDevice(id: "phone")
        XCTAssertNil(ReminderURLProtocol.fixture.token(for: "phone"))
    }
}
