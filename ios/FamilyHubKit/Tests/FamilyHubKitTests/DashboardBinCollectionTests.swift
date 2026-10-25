import XCTest
@testable import FamilyHubKit

@MainActor
final class DashboardBinCollectionTests: XCTestCase {
    func testDashboardDecodesTomorrowCollections() throws {
        let data = Data(#"{"chores_due_today":0,"chores_overdue":0,"chores_due_today_list":[],"chores_overdue_list":[],"meals_this_week":0,"today_meals":[],"bin_collections_tomorrow":[{"date":"2026-10-28","type":"food_caddy","name":"Food caddy"}]}"#.utf8)

        let stats = try JSONDecoder().decode(DashboardStats.self, from: data)

        XCTAssertEqual(stats.binCollectionsTomorrow, [BinCollection(date: "2026-10-28", type: "food_caddy", name: "Food caddy")])
        XCTAssertEqual(stats.binCollectionsTomorrow.first?.imageName, "BinFoodCaddy")
    }

    func testDashboardDefaultsMissingCollectionsToEmpty() throws {
        let data = Data(#"{"chores_due_today":0,"chores_overdue":0,"chores_due_today_list":[],"chores_overdue_list":[],"meals_this_week":0,"today_meals":[]}"#.utf8)

        let stats = try JSONDecoder().decode(DashboardStats.self, from: data)

        XCTAssertTrue(stats.binCollectionsTomorrow.isEmpty)
    }
}
