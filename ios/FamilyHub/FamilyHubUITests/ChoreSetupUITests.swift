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
        let demo = app.buttons["Try Demo"]
        _ = demo.waitForExistence(timeout: 5)
        scrollTo(demo, in: app)
        XCTAssertTrue(demo.waitForExistence(timeout: 10))
        demo.tap()
        selectFamilyTab("Tasks", in: app)
        scrollTo(app.staticTexts["Chore rules"], in: app)
        XCTAssertTrue(app.staticTexts["Chore rules"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["Complete Vacuum living room"].exists)
        let vacuum = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Vacuum living room")).firstMatch
        scrollTo(vacuum, in: app)
        XCTAssertTrue(vacuum.waitForExistence(timeout: 5))
        capture(app, name: "chore-setup-list")
        vacuum.tap()
        let edit = app.buttons["Edit Chore"]
        scrollTo(edit, in: app)
        XCTAssertTrue(edit.exists)
        capture(app, name: "chore-setup-detail")
        edit.tap()
        XCTAssertTrue(app.navigationBars["Edit Chore"].waitForExistence(timeout: 5))
        app.buttons["Cancel"].tap()
        app.navigationBars.buttons.element(boundBy: 0).tap()
        selectFamilyTab("Today", in: app)
        let complete = app.buttons["Complete Vacuum living room"]
        scrollTo(complete, in: app)
        XCTAssertTrue(complete.exists)
        capture(app, name: "home-chores")
        app.buttons["Actions for Vacuum living room"].tap()
        app.buttons["Skip this task"].tap()
        app.buttons["Skip task"].tap()
        XCTAssertTrue(complete.waitForNonExistence(timeout: 5))
    }

    @MainActor
    private func scrollTo(_ element: XCUIElement, in app: XCUIApplication) {
        for _ in 0..<20 {
            let bottom = app.tabBars.firstMatch.exists ? app.tabBars.firstMatch.frame.minY : app.windows.firstMatch.frame.maxY
            if element.exists && element.isHittable && element.frame.midY < bottom { return }
            let towardTop = element.exists && element.frame.midY < 0
            let start = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.4 : 0.7))
            let end = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.7 : 0.4))
            start.press(forDuration: 0.1, thenDragTo: end)
        }
    }

    @MainActor
    private func capture(_ app: XCUIApplication, name: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
