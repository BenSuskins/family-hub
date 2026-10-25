import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
import XCTest
@testable import FamilyHubKit

private final class MealSettingsFixture: @unchecked Sendable {
    private let lock = NSLock()
    private var online = true
    private var weekday = "monday"
    private var lastPatch: [String: String] = [:]

    func setOnline(_ value: Bool) { lock.lock(); defer { lock.unlock() }; online = value }
    func patch() -> [String: String] { lock.lock(); defer { lock.unlock() }; return lastPatch }

    func respond(_ request: URLRequest) -> (Int, Data) {
        lock.lock()
        defer { lock.unlock() }
        guard online else { return (503, Data()) }
        switch (request.httpMethod, request.url!.path) {
        case ("GET", "/api/settings"):
            return (200, try! JSONSerialization.data(withJSONObject: ["family_name": "Family", "meal_week_start": weekday]))
        case ("PATCH", "/api/settings"):
            let data: Data
            if let body = request.httpBody { data = body }
            else if let stream = request.httpBodyStream {
                stream.open()
                defer { stream.close() }
                var body = Data()
                var buffer = [UInt8](repeating: 0, count: 512)
                while stream.hasBytesAvailable {
                    let count = stream.read(&buffer, maxLength: buffer.count)
                    if count <= 0 { break }
                    body.append(buffer, count: count)
                }
                data = body
            } else { data = Data() }
            lastPatch = (try? JSONSerialization.jsonObject(with: data) as? [String: String]) ?? [:]
            weekday = lastPatch["meal_week_start"] ?? weekday
            return (204, Data())
        case ("GET", "/api/meals"):
            return (200, Data("[]".utf8))
        case ("POST", "/api/meals"):
            return (200, Data(#"{"Date":"2026-10-25","MealType":"dinner","Name":"Soup","Notes":"","RecipeID":null}"#.utf8))
        case ("DELETE", "/api/meals"):
            return (204, Data())
        default:
            return (404, Data())
        }
    }
}

private final class MealSettingsURLProtocol: URLProtocol, @unchecked Sendable {
    static let fixture = MealSettingsFixture()
    override class func canInit(with request: URLRequest) -> Bool { request.url?.host == "meal-settings.example" || request.url?.host == "other-meal-settings.example" }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        let (status, data) = Self.fixture.respond(request)
        client?.urlProtocol(self, didReceive: HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: "HTTP/1.1", headerFields: ["Content-Type": "application/json"])!, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}
}

@MainActor
final class MealSettingsAPIClientTests: XCTestCase {
    func testWeekdayPatchAndOfflineSettingsStayWithTheirServerAndAccount() async throws {
        let provider = StaticTokenProvider(token: UUID().uuidString)
        let client = makeClient(provider: provider)
        MealSettingsURLProtocol.fixture.setOnline(true)
        let initial = try await client.fetchSettings()
        XCTAssertEqual(initial.mealWeekStart, .monday)
        try await client.updateMealWeekStart(.saturday)
        XCTAssertEqual(MealSettingsURLProtocol.fixture.patch(), ["meal_week_start": "saturday"])
        MealSettingsURLProtocol.fixture.setOnline(false)
        defer { MealSettingsURLProtocol.fixture.setOnline(true) }
        let relaunched = makeClient(provider: provider)
        let saved = try await relaunched.fetchSettings()
        XCTAssertEqual(saved.mealWeekStart, .saturday)
        let otherProvider = StaticTokenProvider(token: UUID().uuidString)
        for other in [makeClient(provider: otherProvider), makeClient(provider: provider, host: "other-meal-settings.example")] {
            do { _ = try await other.fetchSettings(); XCTFail("Another account or server must not use saved settings") }
            catch { }
        }
    }

    func testSaveAndDeleteUpdateAllCachedWeekRangesContainingTheDate() async throws {
        let provider = StaticTokenProvider(token: UUID().uuidString)
        let client = makeClient(provider: provider)
        MealSettingsURLProtocol.fixture.setOnline(true)
        let starts = ["2026-10-19", "2026-10-23", "2026-10-25"].map { APIDate.parse($0)! }
        for start in starts { _ = try await client.fetchMeals(week: start) }
        _ = try await client.saveMeal(date: "2026-10-25", mealType: "dinner", name: "Soup", recipeID: nil)
        MealSettingsURLProtocol.fixture.setOnline(false)
        defer { MealSettingsURLProtocol.fixture.setOnline(true) }
        for start in starts {
            let meals = try await client.fetchMeals(week: start)
            XCTAssertEqual(meals.first?.name, "Soup")
        }
        MealSettingsURLProtocol.fixture.setOnline(true)
        try await client.deleteMeal(date: "2026-10-25", mealType: "dinner")
        MealSettingsURLProtocol.fixture.setOnline(false)
        for start in starts {
            let meals = try await client.fetchMeals(week: start)
            XCTAssertTrue(meals.isEmpty)
        }
    }

    private func makeClient(provider: StaticTokenProvider, host: String = "meal-settings.example") -> APIClient {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [MealSettingsURLProtocol.self]
        return APIClient(baseURL: URL(string: "https://\(host)")!, session: URLSession(configuration: configuration), tokenProvider: provider, retryPolicy: .none)
    }
}
