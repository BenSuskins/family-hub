import XCTest
@testable import FamilyHubKit

@MainActor
final class CalendarScheduleTests: XCTestCase {
    private var calendar: Calendar {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "Europe/London")!
        calendar.firstWeekday = 2
        return calendar
    }

    private func date(_ value: String) -> Date { ISO8601DateFormatter().date(from: value)! }

    func testEventAppearsOnEachDayItOccupiesWithoutIncludingItsExclusiveEnd() {
        let schedule = CalendarSchedule(calendar: calendar)
        let event = CalendarEvent(id: "trip", title: "Trip", description: "", location: "",
                                  startTime: date("2026-04-01T00:00:00Z"),
                                  endTime: date("2026-04-03T00:00:00Z"), allDay: true, color: "")
        XCTAssertTrue(schedule.contains(event, on: date("2026-04-01T12:00:00Z")))
        XCTAssertTrue(schedule.contains(event, on: date("2026-04-02T12:00:00Z")))
        XCTAssertFalse(schedule.contains(event, on: date("2026-04-03T12:00:00Z")))
    }
    func testMonthGridUsesTheLocalWeekStartAndIncludesLeapDay() {
        let schedule = CalendarSchedule(calendar: calendar)
        let days = schedule.monthDays(containing: date("2024-02-10T12:00:00Z"))
        XCTAssertEqual(days.count, 35)
        XCTAssertEqual(days.compactMap { $0 }.count, 29)
        XCTAssertNil(days[0])
        XCTAssertEqual(schedule.dayKey(days[3]!), "2024-02-01")
        XCTAssertEqual(schedule.dayKey(days[31]!), "2024-02-29")
        XCTAssertEqual(schedule.dayKey(schedule.weekStart(containing: date("2024-02-10T12:00:00Z"))), "2024-02-05")
    }

    func testTimelinePlacesOverlapsInColumnsAndUsesWallTimeAcrossDaylightSaving() {
        let schedule = CalendarSchedule(calendar: calendar)
        let first = CalendarEvent(id: "first", title: "First", description: "", location: "",
                                  startTime: date("2026-03-29T08:00:00Z"), endTime: date("2026-03-29T09:00:00Z"),
                                  allDay: false, color: "")
        let second = CalendarEvent(id: "second", title: "Second", description: "", location: "",
                                   startTime: date("2026-03-29T08:30:00Z"), endTime: date("2026-03-29T10:00:00Z"),
                                   allDay: false, color: "")
        let placements = schedule.placements(for: [second, first], on: first.startTime)
        XCTAssertEqual(placements.count, 2)
        XCTAssertEqual(placements[0].startMinute, 540)
        XCTAssertEqual(placements[0].endMinute, 600)
        XCTAssertEqual(placements[0].column, 0)
        XCTAssertEqual(placements[1].column, 1)
        XCTAssertEqual(placements.map(\.columnCount), [2, 2])
    }

    func testAllDayDatesDoNotShiftWhenTheDeviceTimeZoneChanges() {
        var pacific = calendar
        pacific.timeZone = TimeZone(identifier: "America/Los_Angeles")!
        let schedule = CalendarSchedule(calendar: pacific)
        let event = CalendarEvent(id: "holiday", title: "Holiday", description: "", location: "",
                                  startTime: date("2026-04-01T00:00:00Z"),
                                  endTime: date("2026-04-02T00:00:00Z"), allDay: true, color: "")
        XCTAssertFalse(schedule.contains(event, on: date("2026-03-31T12:00:00Z")))
        XCTAssertTrue(schedule.contains(event, on: date("2026-04-01T12:00:00Z")))
        XCTAssertFalse(schedule.contains(event, on: date("2026-04-02T12:00:00Z")))
    }

    func testShortEventsKeepSeparateTouchTargetsAtSmallTimelineScales() {
        let schedule = CalendarSchedule(calendar: calendar)
        let first = CalendarEvent(id: "first", title: "First", description: "", location: "",
                                  startTime: date("2026-04-01T08:00:00Z"), endTime: date("2026-04-01T08:10:00Z"),
                                  allDay: false, color: "")
        let second = CalendarEvent(id: "second", title: "Second", description: "", location: "",
                                   startTime: date("2026-04-01T08:40:00Z"), endTime: date("2026-04-01T08:50:00Z"),
                                   allDay: false, color: "")
        let placements = schedule.placements(for: [first, second], on: first.startTime, minimumMinutes: 60)
        XCTAssertEqual(placements.map(\.columnCount), [2, 2])
        XCTAssertEqual(placements.map(\.column), [0, 1])
    }

}
