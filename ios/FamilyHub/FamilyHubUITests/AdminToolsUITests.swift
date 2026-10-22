import XCTest

final class AdminToolsUITests: XCTestCase {
    @MainActor
    func testAdminCanManageCalendarFeedsAndCreateBackup() {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 10))
        app.buttons["Open profile"].tap()

        let calendarSettings = app.buttons["Calendar feeds settings"]
        if !calendarSettings.isHittable { app.swipeUp() }
        XCTAssertTrue(calendarSettings.waitForExistence(timeout: 5))
        calendarSettings.tap()
        XCTAssertTrue(app.navigationBars["Calendars"].waitForExistence(timeout: 5))

        app.buttons["Add Calendar"].tap()
        let name = app.textFields["Calendar name"]
        XCTAssertTrue(name.waitForExistence(timeout: 5))
        name.tap()
        name.typeText("School")
        let url = app.textFields["Calendar URL"]
        url.tap()
        url.typeText("https://example.com/school.ics")
        app.buttons["Add"].tap()
        XCTAssertTrue(app.staticTexts["School"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "identifier BEGINSWITH %@", "Refresh calendar")).firstMatch.exists)

        app.navigationBars.buttons.element(boundBy: 0).tap()
        let backupSettings = app.buttons["Database backup settings"]
        if !backupSettings.isHittable { app.swipeUp() }
        XCTAssertTrue(backupSettings.waitForExistence(timeout: 5))
        backupSettings.tap()
        XCTAssertTrue(app.navigationBars["Database Backup"].waitForExistence(timeout: 5))
        app.buttons["Create Database Backup"].tap()
        XCTAssertTrue(app.buttons["Save Database Backup"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["Choose Database Backup"].exists)
    }

    @MainActor
    func testMemberCannotSeeAdminCalendarOrBackupSettings() {
        let app = XCUIApplication()
        app.launchArguments.append("-demo-member")
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 10))
        app.buttons["Open profile"].tap()
        app.swipeUp()

        XCTAssertFalse(app.buttons["Calendar feeds settings"].exists)
        XCTAssertFalse(app.buttons["Database backup settings"].exists)
    }
}
