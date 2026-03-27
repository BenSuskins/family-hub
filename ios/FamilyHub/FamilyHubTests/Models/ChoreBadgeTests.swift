import XCTest
@testable import FamilyHub

final class ChoreBadgeTests: XCTestCase {
    func testOverdueLabel() {
        XCTAssertEqual(ChoreBadge.overdue.label, "Overdue")
    }

    func testDueTodayLabel() {
        XCTAssertEqual(ChoreBadge.dueToday.label, "Today")
    }

    func testDueSoonLabel() {
        XCTAssertEqual(ChoreBadge.dueSoon.label, "Due Soon")
    }

    func testChoreOverdueStatusMapsToOverdueBadge() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .overdue,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertEqual(chore.badge, .overdue)
    }

    func testChoreCompletedStatusHasNoBadge() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .completed,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertNil(chore.badge)
    }

    func testChorePendingDueTodayMapsToDueToday() {
        let todayString = ISO8601DateFormatter().string(from: Date())
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: todayString, assignedToUserID: nil)
        XCTAssertEqual(chore.badge, .dueToday)
    }

    func testChorePendingFutureDateMapsToDueSoon() {
        let futureDate = Calendar.current.date(byAdding: .day, value: 3, to: Date())!
        let futureString = ISO8601DateFormatter().string(from: futureDate)
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: futureString, assignedToUserID: nil)
        XCTAssertEqual(chore.badge, .dueSoon)
    }
}
