import XCTest

final class ChoreSetupUITests: XCTestCase {
    @MainActor
    func testTasksShowsSetupAndHomeShowsActions() {
        inspectChores(largeText: false)
    }

    @MainActor
    func testChoresWithAccessibilityText() {
        inspectChores(largeText: true)
    }

    @MainActor
    private func inspectChores(largeText: Bool) {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["-UIPreferredContentSizeCategoryName", largeText ? "UICTContentSizeCategoryAccessibilityXXXL" : "UICTContentSizeCategoryL"]
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        app.tabBars.buttons["Tasks"].tap()
        XCTAssertTrue(app.staticTexts["Configured chores"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["Complete Vacuum living room"].exists)
        let vacuum = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Vacuum living room")).firstMatch
        XCTAssertTrue(vacuum.waitForExistence(timeout: 5))
        capture(app, name: "chore-setup-list")
        vacuum.tap()
        let edit = app.buttons["Edit Chore"]
        for _ in 0..<20 where !edit.isHittable { app.swipeUp() }
        XCTAssertTrue(edit.exists)
        capture(app, name: "chore-setup-detail")
        edit.tap()
        XCTAssertTrue(app.navigationBars["Edit Chore"].waitForExistence(timeout: 5))
        app.buttons["Cancel"].tap()
        app.navigationBars.buttons.element(boundBy: 0).tap()
        app.tabBars.buttons["Today"].tap()
        let complete = app.buttons["Complete Vacuum living room"]
        for _ in 0..<20 where !complete.isHittable { app.swipeUp() }
        XCTAssertTrue(complete.exists)
        capture(app, name: "home-chores")
        app.buttons["Actions for Vacuum living room"].tap()
        app.buttons["Skip this task"].tap()
        XCTAssertFalse(complete.exists)
    }

    @MainActor
    private func capture(_ app: XCUIApplication, name: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
