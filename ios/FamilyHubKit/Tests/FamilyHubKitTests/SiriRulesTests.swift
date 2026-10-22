import XCTest
@testable import FamilyHubKit

final class SiriRulesTests: XCTestCase {
    @MainActor
    func testExplicitMealDateDoesNotShiftAcrossTimezones() async throws {
        let rules = try SiriDateRules(timezone: "Pacific/Auckland")
        let day = try rules.day(DateComponents(year: 2026, month: 10, day: 27), now: Date(timeIntervalSince1970: 0))
        XCTAssertEqual(day.key, "2026-10-27")
        XCTAssertEqual(day.spoken, "Tuesday, 27 October 2026")
    }
}

extension SiriRulesTests {
    @MainActor
    func testTodayUsesHouseholdTimezoneAtExecution() async throws {
        let rules = try SiriDateRules(timezone: "Pacific/Auckland")
        let before = APIDate.iso.date(from: "2026-10-26T10:30:00Z")!
        let after = APIDate.iso.date(from: "2026-10-26T11:30:00Z")!
        XCTAssertEqual(try rules.day(nil, now: before).key, "2026-10-26")
        XCTAssertEqual(try rules.day(nil, now: after).key, "2026-10-27")
    }

    @MainActor
    func testMyChoresExcludeFutureUndatedCompletedAndOtherPeople() async throws {
        let rules = try SiriDateRules(timezone: "Europe/London")
        let chores = [
            Chore(id: "today", name: "Dishes", description: "", status: .pending, dueDate: "2026-10-19T23:00:00Z", assignedToUserID: "me"),
            Chore(id: "past", name: "Bins", description: "", status: .pending, dueDate: "2026-10-18", assignedToUserID: "me"),
            Chore(id: "future", name: "Future", description: "", status: .pending, dueDate: "2026-10-21", assignedToUserID: "me"),
            Chore(id: "undated", name: "Undated", description: "", status: .pending, assignedToUserID: "me"),
            Chore(id: "done", name: "Done", description: "", status: .completed, dueDate: "2026-10-19", assignedToUserID: "me"),
            Chore(id: "other", name: "Other", description: "", status: .overdue, dueDate: "2026-10-18", assignedToUserID: "other"),
        ]
        let now = APIDate.iso.date(from: "2026-10-20T00:30:00Z")!
        XCTAssertEqual(rules.eligibleChores(chores, userID: "me", now: now).map(\.id), ["past", "today"])
    }

    @MainActor
    func testStockChangesPreserveDetailsAndClampCount() async throws {
        let item = InventoryItem(id: "soap", areaID: "bathroom", name: "Soap", quantity: 3, unit: "bars", lowAt: 1)
        let saved = try InventoryStockChange.quantityDelta(-5).applying(to: item)
        XCTAssertEqual(saved.quantity, 0)
        XCTAssertEqual(saved.unit, "bars")
        XCTAssertEqual(saved.lowAt, 1)
        XCTAssertThrowsError(try InventoryStockChange.level(50).applying(to: item))
    }

    @MainActor
    func testRecipeNamesMatchWithoutCaseOrExtraWhitespace() async {
        let recipes = [Recipe(id: "one", title: "Butter Chicken"), Recipe(id: "two", title: "Butter Chicken Curry")]
        XCTAssertEqual(SiriNames.recipes(named: " butter  CHICKEN ", in: recipes).map(\.id), ["one"])
    }
}

extension SiriRulesTests {
    @MainActor
    func testChoreDueDayMatchesServerWithoutTimezoneShift() async throws {
        let rules = try SiriDateRules(timezone: "America/Los_Angeles")
        let tomorrow = Chore(id: "tomorrow", name: "Bins", description: "", status: .pending, dueDate: "2026-10-20T00:00:00Z", assignedToUserID: "me")
        let today = Chore(id: "today", name: "Dishes", description: "", status: .pending, dueDate: "2026-10-19T00:00:00.000Z", assignedToUserID: "me")
        let now = APIDate.iso.date(from: "2026-10-19T18:00:00Z")!
        XCTAssertEqual(rules.eligibleChores([tomorrow, today], userID: "me", now: now).map(\.id), ["today"])
    }
}

extension SiriRulesTests {
    @MainActor
    func testCalendarDatesStayStableAcrossDaylightSavingAndYearBoundaries() async throws {
        let rules = try SiriDateRules(timezone: "Europe/London")
        let cases: [(DateComponents, String)] = [
            (DateComponents(year: 2026, month: 10, day: 25), "2026-10-25"),
            (DateComponents(year: 2026, month: 12, day: 31), "2026-12-31"),
            (DateComponents(year: 2027, month: 1, day: 1), "2027-01-01"),
        ]
        for (components, expected) in cases {
            let selected = try rules.day(components, now: Date(timeIntervalSince1970: 0))
            XCTAssertEqual(selected.key, expected)
            XCTAssertEqual(APIDate.dayString(selected.queryDate), expected)
        }
        XCTAssertThrowsError(try rules.day(DateComponents(year: 2026, month: 2, day: 30), now: Date()))
    }

    @MainActor
    func testDuplicateRecipeMatchesRemainAvailableForAChoice() async {
        let recipes = [Recipe(id: "one", title: "Butter Chicken"), Recipe(id: "two", title: "butter chicken")]
        XCTAssertEqual(SiriNames.recipes(named: "Butter chicken", in: recipes).map(\.id), ["one", "two"])
        XCTAssertEqual(SiriNames.recipes(named: "New meal", in: recipes), [])
    }

    @MainActor
    func testLongListSummaryDoesNotReadEveryItem() async {
        let summary = SiriNames.summary(["One", "Two", "Three", "Four", "Five", "Six"], empty: "Empty")
        XCTAssertTrue(summary.contains("6 items"))
        XCTAssertTrue(summary.contains("Five"))
        XCTAssertFalse(summary.contains("Six"))
    }
}

extension SiriRulesTests {
    @MainActor
    func testResolvedAndForecastChoresCannotBeCompletedThroughSiri() async throws {
        let rules = try SiriDateRules(timezone: "Europe/London")
        let chores = [
            Chore(id: "due", name: "Due", description: "", status: .pending, dueDate: "2026-10-19", assignedToUserID: "me"),
            Chore(id: "skipped", name: "Skipped", description: "", status: .skipped, dueDate: "2026-10-18", assignedToUserID: "me"),
            Chore(id: "superseded", name: "Superseded", description: "", status: .superseded, dueDate: "2026-10-18", assignedToUserID: "me"),
            Chore(id: "forecast", name: "Forecast", description: "", status: .pending, dueDate: "2026-10-19", assignedToUserID: "me", isForecast: true),
        ]
        XCTAssertEqual(rules.eligibleChores(chores, userID: "me", now: APIDate.iso.date(from: "2026-10-19T12:00:00Z")!).map(\.id), ["due"])
    }
}
