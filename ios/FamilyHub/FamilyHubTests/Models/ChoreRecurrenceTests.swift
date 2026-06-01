import XCTest
@testable import FamilyHub

final class ChoreRecurrenceTests: XCTestCase {
    private func chore(recurrenceType: String, recurrenceValue: String = "") -> Chore {
        Chore(id: "1", name: "Test", description: "", status: .pending,
              recurrenceType: recurrenceType, recurrenceValue: recurrenceValue)
    }

    func testRecurrenceConfigParsesWeeklyDays() {
        let config = RecurrenceConfig(json: "{\"interval\":2,\"days\":[\"monday\",\"wednesday\"]}")
        XCTAssertEqual(config.interval, 2)
        XCTAssertEqual(config.days, ["monday", "wednesday"])
    }

    func testRecurrenceConfigParsesMonthlyDayOfMonth() {
        let config = RecurrenceConfig(json: "{\"interval\":1,\"day_of_month\":15}")
        XCTAssertEqual(config.dayOfMonth, 15)
    }

    func testRecurrenceConfigDefaultsOnInvalidJSON() {
        let config = RecurrenceConfig(json: "not json")
        XCTAssertEqual(config.interval, 1)
        XCTAssertEqual(config.unit, "days")
        XCTAssertTrue(config.days.isEmpty)
    }

    func testRecurrenceSummaryForOneTimeChore() {
        XCTAssertEqual(chore(recurrenceType: "none").recurrenceSummary, "One-time")
        XCTAssertFalse(chore(recurrenceType: "none").isRecurring)
    }

    func testRecurrenceSummaryForDaily() {
        XCTAssertEqual(chore(recurrenceType: "daily").recurrenceSummary, "Daily")
    }

    func testRecurrenceSummaryForWeeklyWithDays() {
        let c = chore(recurrenceType: "weekly", recurrenceValue: "{\"interval\":1,\"days\":[\"monday\",\"wednesday\"]}")
        XCTAssertEqual(c.recurrenceSummary, "Weekly · Mon, Wed")
    }

    func testRecurrenceSummaryForMonthlyWithDayOfMonth() {
        let c = chore(recurrenceType: "monthly", recurrenceValue: "{\"interval\":1,\"day_of_month\":3}")
        XCTAssertEqual(c.recurrenceSummary, "Monthly · day 3")
    }

    func testRecurrenceSummaryForCustom() {
        let c = chore(recurrenceType: "custom", recurrenceValue: "{\"interval\":3,\"unit\":\"weeks\"}")
        XCTAssertEqual(c.recurrenceSummary, "Every 3 weeks")
    }
}
