import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class DatabaseBackupTests: XCTestCase {
    func testSelectingDatabaseArchiveRequiresConfirmation() {
        let selection = DatabaseRestoreSelection()
        let archive = URL(fileURLWithPath: "/tmp/family-hub.db.gz")

        XCTAssertTrue(selection.select(archive))
        XCTAssertTrue(selection.requiresConfirmation)
        XCTAssertEqual(selection.selectedFileURL, archive)

        selection.clear()
        XCTAssertFalse(selection.requiresConfirmation)
    }

    func testSelectingOtherFileDoesNotAllowRestore() {
        let selection = DatabaseRestoreSelection()

        XCTAssertFalse(selection.select(URL(fileURLWithPath: "/tmp/family-hub.db")))
        XCTAssertFalse(selection.requiresConfirmation)
        XCTAssertEqual(selection.actionError, .badRequest(serverMessage: "Select a .db.gz backup file."))
    }
}
