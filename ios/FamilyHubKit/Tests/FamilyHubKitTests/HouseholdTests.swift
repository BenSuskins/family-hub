import XCTest
@testable import FamilyHubKit

final class HouseholdTests: XCTestCase {
    func testShoppingCacheIsScopedToServerAndUser() async throws {
        try await MainActor.run {
            let defaults = UserDefaults(suiteName: UUID().uuidString)!
            let cache = ShoppingCache(defaults: defaults)
            let item = ShoppingItem(id: "soap", name: "Soap")
            cache.save([item], server: "https://one.example", userID: "one", at: Date(timeIntervalSince1970: 100))
            XCTAssertEqual(cache.load(server: "https://one.example", userID: "one")?.items.first?.name, "Soap")
            XCTAssertNil(cache.load(server: "https://two.example", userID: "one"))
            XCTAssertNil(cache.load(server: "https://one.example", userID: "two"))
            cache.clear()
            XCTAssertNil(cache.load(server: "https://one.example", userID: "one"))
        }
    }

    func testSignedOutSessionCannotRestoreTheCache() async {
        await MainActor.run {
            let defaults = UserDefaults(suiteName: UUID().uuidString)!
            let oldSession = ShoppingCache(defaults: defaults)
            ShoppingCache(defaults: defaults).clear()
            oldSession.save([ShoppingItem(id: "soap", name: "Soap")], server: "server", userID: "old")
            XCTAssertNil(ShoppingCache(defaults: defaults).load(server: "server", userID: "old"))
        }
    }

    func testDepartureStopsAtEventStart() async throws {
        try await MainActor.run {
            let data = Data(#"{"eventID":"next","title":"Dentist","destination":"10 High Street","startTime":"2026-10-18T14:00:00Z","leaveAt":"2026-10-18T13:30:00Z","drivingSeconds":1200,"fetchedAt":"2026-10-18T13:20:00Z"}"#.utf8)
            let departure = try JSONDecoder().decode(Departure.self, from: data)
            XCTAssertEqual(departure.countdown(at: APIDate.parse("2026-10-18T13:20:00Z")!), "Leave in 10 min")
            XCTAssertEqual(departure.countdown(at: APIDate.parse("2026-10-18T08:01:00Z")!), "Leave in 5 hr 29 min")
            XCTAssertEqual(departure.countdown(at: APIDate.parse("2026-10-18T09:00:00Z")!), "Leave in 4 hr 30 min")
            XCTAssertEqual(departure.countdown(at: APIDate.parse("2026-10-18T13:31:00Z")!), "Leave now")
            XCTAssertNil(departure.countdown(at: APIDate.parse("2026-10-18T14:00:00Z")!))
        }
    }
}
