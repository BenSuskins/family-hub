import XCTest

final class ChoreRemindersUITests: XCTestCase {
    @MainActor
    func testFamilyReminderSettingsShowBothTimesAndTimezone() {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 10))
        app.buttons["Open profile"].tap()
        XCTAssertTrue(app.buttons["Chore Reminders"].waitForExistence(timeout: 5))
        app.buttons["Chore Reminders"].tap()
        XCTAssertTrue(app.navigationBars["Notifications"].waitForExistence(timeout: 5))
        app.swipeUp()
        XCTAssertTrue(app.staticTexts["First reminder"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Second reminder"].exists)
        XCTAssertEqual(app.datePickers.count, 2)
        XCTAssertEqual(app.buttons.matching(identifier: "Time Picker").element(boundBy: 0).value as? String, "10:00")
        XCTAssertEqual(app.buttons.matching(identifier: "Time Picker").element(boundBy: 1).value as? String, "16:30")
        XCTAssertTrue(app.staticTexts["Time zone, Europe/London"].exists)
        XCTAssertTrue(app.switches["Family Reminders"].exists)
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Chore reminder settings"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }
    @MainActor
    func testAdminCanComposeFamilyBroadcast() {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 10))
        app.buttons["Open profile"].tap()
        app.swipeUp()
        app.buttons["Family message"].tap()
        XCTAssertTrue(app.navigationBars["Family message"].waitForExistence(timeout: 5))
        let send = app.buttons["Send to Family"]
        XCTAssertFalse(send.isEnabled)
        let field = app.textFields["Family message"].exists ? app.textFields["Family message"] : app.textViews["Family message"]
        field.tap()
        field.typeText("Dinner is ready.")
        XCTAssertTrue(send.isEnabled)
        if let done = app.buttons.matching(identifier: "Done").allElementsBoundByIndex.first(where: \.isHittable) { done.tap() }
        if !send.isHittable { app.swipeUp() }
        send.tap()
        XCTAssertTrue(app.staticTexts["No family iPhones have notifications enabled."].waitForExistence(timeout: 5))
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Family broadcast result"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }

}
