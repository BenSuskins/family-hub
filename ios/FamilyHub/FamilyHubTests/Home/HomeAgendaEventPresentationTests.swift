import XCTest
import FamilyHubKit
@testable import FamilyHub

final class HomeAgendaEventPresentationTests: XCTestCase {
    func testEventIsNotPastBeforeItsEnd() {
        let event = makeEvent(endTime: date(hour: 11))

        XCTAssertFalse(HomeAgendaEventPresentation.isPast(event: event, at: date(hour: 10, minute: 59)))
    }

    func testEventIsPastAtItsEnd() {
        let event = makeEvent(endTime: date(hour: 11))

        XCTAssertTrue(HomeAgendaEventPresentation.isPast(event: event, at: date(hour: 11)))
    }

    func testEventIsPastAfterItsEnd() {
        let event = makeEvent(endTime: date(hour: 11))

        XCTAssertTrue(HomeAgendaEventPresentation.isPast(event: event, at: date(hour: 11, minute: 1)))
    }

    func testEventWithoutEndTimeIsNotPast() {
        let event = makeEvent(endTime: nil)

        XCTAssertFalse(HomeAgendaEventPresentation.isPast(event: event, at: date(hour: 23, minute: 59)))
    }

    private func makeEvent(endTime: Date?) -> CalendarEvent {
        CalendarEvent(
            id: "event",
            title: "Family dinner",
            description: "",
            location: "Home",
            startTime: date(hour: 10),
            endTime: endTime,
            allDay: false,
            color: "#007AFF"
        )
    }

    private func date(hour: Int, minute: Int = 0) -> Date {
        Calendar(identifier: .gregorian).date(
            from: DateComponents(year: 2026, month: 10, day: 22, hour: hour, minute: minute)
        )!
    }
}
