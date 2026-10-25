import Foundation
import XCTest
@testable import FamilyHubKit

@MainActor
final class MealWeekTests: XCTestCase {
    func testEachWeekdayStartsTheContainingWeek() async throws {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "Europe/London")!
        let date = try XCTUnwrap(ISO8601DateFormatter().date(from: "2026-10-25T12:00:00Z"))
        let expected = ["2026-10-19", "2026-10-20", "2026-10-21", "2026-10-22", "2026-10-23", "2026-10-24", "2026-10-25"]
        let formatter = DateFormatter()
        formatter.calendar = calendar
        formatter.timeZone = calendar.timeZone
        formatter.dateFormat = "yyyy-MM-dd"
        for (day, key) in zip(MealWeekday.allCases, expected) {
            let start = day.weekStart(containing: date, calendar: calendar)
            XCTAssertEqual(formatter.string(from: start), key)
            XCTAssertEqual(calendar.component(.hour, from: start), 0)
        }
        let january = try XCTUnwrap(ISO8601DateFormatter().date(from: "2027-01-01T12:00:00Z"))
        XCTAssertEqual(formatter.string(from: MealWeekday.monday.weekStart(containing: january, calendar: calendar)), "2026-12-28")
        let spring = try XCTUnwrap(ISO8601DateFormatter().date(from: "2026-03-29T12:00:00Z"))
        XCTAssertEqual(formatter.string(from: MealWeekday.friday.weekStart(containing: spring, calendar: calendar)), "2026-03-27")
    }

    func testOlderServersDefaultToFriday() async throws {
        let settings = try JSONDecoder().decode(AppSettings.self, from: Data(#"{"family_name":"Family"}"#.utf8))
        XCTAssertEqual(settings.mealWeekStart, .friday)
        let configured = try JSONDecoder().decode(AppSettings.self, from: Data(#"{"family_name":"Family","meal_week_start":"monday"}"#.utf8))
        XCTAssertEqual(configured.mealWeekStart, .monday)
    }
}
