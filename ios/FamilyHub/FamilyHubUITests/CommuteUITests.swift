import XCTest
import UIKit

final class CommuteUITests: XCTestCase {
    @MainActor
    func testCommuteSettingsAndHomeOnlyProfileButton() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["-demo-commute"]
        app.launch()
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["To work"].waitForExistence(timeout: 5))
        capture(app, "commute-today")
        app.swipeUp()
        capture(app, "commute-trains")
        app.swipeDown()
        app.buttons["Open profile"].tap()
        app.buttons["Commute settings"].tap()
        let enabled = app.switches["commute-to_work-enabled"]
        XCTAssertTrue(enabled.waitForExistence(timeout: 5))
        XCTAssertEqual(enabled.value as? String, "1")
        capture(app, "commute-settings")
        enabled.coordinate(withNormalizedOffset: CGVector(dx: 0.9, dy: 0.5)).tap()
        XCTAssertEqual(enabled.value as? String, "0")
        app.navigationBars.buttons["BackButton"].tap()
        app.buttons["Done"].tap()
        XCTAssertTrue(app.staticTexts["To work"].waitForNonExistence(timeout: 5))
        for tab in ["Calendar", "Chores", "Food", "Household"] {
            selectFamilyTab(tab, in: app)
            XCTAssertFalse(app.buttons["Open profile"].exists, "Only Today exposes Settings")
        }
        selectFamilyTab("Today", in: app)
        XCTAssertTrue(app.buttons["Open profile"].exists)
    }

    @MainActor
    func testCommuteAtAccessibilityTextSize() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["-demo-commute", "-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        app.launch()
        let demo = app.buttons["Try Demo"]
        for _ in 0..<6 where !demo.isHittable { app.swipeUp() }
        demo.tap()
        XCTAssertTrue(app.staticTexts["To work"].waitForExistence(timeout: 5))
        let train = app.descendants(matching: .any).matching(NSPredicate(format: "label CONTAINS[c] %@", "expected departure")).firstMatch
        XCTAssertTrue(train.waitForExistence(timeout: 5))
        for _ in 0..<20 where !train.isHittable { app.swipeUp() }
        XCTAssertTrue(train.isHittable)
        XCTAssertLessThanOrEqual(train.frame.maxX, app.windows.firstMatch.frame.maxX)
        capture(app, "commute-large-text")
        app.buttons["Open profile"].tap()
        let settings = app.buttons["Commute settings"]
        for _ in 0..<8 where !settings.isHittable { app.swipeUp() }
        settings.tap()
        XCTAssertTrue(app.switches["commute-to_work-enabled"].waitForExistence(timeout: 5))
        capture(app, "commute-large-settings")
    }

    @MainActor
    private func capture(_ app: XCUIApplication, _ name: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
