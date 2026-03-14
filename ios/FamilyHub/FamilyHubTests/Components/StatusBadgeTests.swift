import XCTest
@testable import FamilyHub

final class StatusBadgeTests: XCTestCase {
    func testOverdueVariantLabel() {
        XCTAssertEqual(StatusBadge.Variant.overdue.label, "Overdue")
    }

    func testDueTodayVariantLabel() {
        XCTAssertEqual(StatusBadge.Variant.dueToday.label, "Today")
    }

    func testDueSoonVariantLabel() {
        XCTAssertEqual(StatusBadge.Variant.dueSoon.label, "Due Soon")
    }

    func testChoreOverdueStatusMapsToOverdueVariant() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .overdue,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertEqual(chore.badgeVariant, .overdue)
    }

    func testChoreCompletedStatusHasNoBadge() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .completed,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertNil(chore.badgeVariant)
    }

    func testChorePendingDueTodayMapsToTodayVariant() {
        let todayString = ISO8601DateFormatter().string(from: Date())
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: todayString, assignedToUserID: nil)
        XCTAssertEqual(chore.badgeVariant, .dueToday)
    }

    func testChorePendingFutureDateMapsToDueSoonVariant() {
        let futureDate = Calendar.current.date(byAdding: .day, value: 3, to: Date())!
        let futureString = ISO8601DateFormatter().string(from: futureDate)
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: futureString, assignedToUserID: nil)
        XCTAssertEqual(chore.badgeVariant, .dueSoon)
    }
}
