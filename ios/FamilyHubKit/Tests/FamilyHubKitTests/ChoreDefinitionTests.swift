import XCTest
@testable import FamilyHubKit

final class ChoreDefinitionTests: XCTestCase {
    func testDefinitionAndHistoryDecodeTheServerContract() throws {
        let definition = Data(#"{"ID":"bins","Chore":{"ID":"bins","Name":"Bins","Description":"","Status":"pending","EligibleAssignees":["ben"],"RecurrenceType":"daily"},"CurrentTask":{"ID":"task","Name":"Bins","Description":"","Status":"overdue","AssignedToUserID":"ben"},"Active":true,"DeletedAt":null}"#.utf8)
        let decoded = try JSONDecoder().decode(ChoreDefinition.self, from: definition)
        XCTAssertEqual(decoded.id, "bins")
        XCTAssertEqual(decoded.currentTask?.id, "task")
        XCTAssertEqual(decoded.chore.eligibleAssignees, ["ben"])
        let history = Data(#"{"Items":[{"ID":"old","Name":"Bins","Description":"","Status":"skipped","ResolvedAt":"2026-10-18T10:00:00Z","ResolvedByUserID":"alex"}],"NextCursor":"old"}"#.utf8)
        let page = try JSONDecoder().decode(ChoreHistoryPage.self, from: history)
        XCTAssertEqual(page.items.first?.status, .skipped)
        XCTAssertEqual(page.items.first?.resolvedByUserID, "alex")
        XCTAssertEqual(page.nextCursor, "old")
    }
}
