import XCTest
import UIKit

final class MealPlanningSettingsUITests: XCTestCase {
    @MainActor
    func testAdminChangesMealWeekAndPlannerUsesTheNewRange() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["-AppleLanguages", "(en)", "-AppleLocale", "en_GB"]
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 10))
        app.buttons["Open profile"].tap()
        let entry = app.buttons["Meal planning"]
        for _ in 0..<5 where !entry.isHittable { app.swipeUp() }
        XCTAssertTrue(entry.waitForExistence(timeout: 5))
        entry.tap()
        XCTAssertTrue(app.navigationBars["Meal planning"].waitForExistence(timeout: 5))
        let save = app.buttons["Save"]
        XCTAssertFalse(save.isEnabled)
        app.buttons["Meal week starts on"].tap()
        app.buttons["Monday"].tap()
        XCTAssertTrue(save.isEnabled)
        let picker = XCTAttachment(screenshot: app.screenshot())
        picker.name = "Meal planning weekday setting"
        picker.lifetime = .keepAlways
        add(picker)
        save.tap()
        XCTAssertTrue(app.navigationBars["Settings"].waitForExistence(timeout: 5))
        app.buttons["Done"].tap()
        selectFamilyTab("Food", in: app)
        if UIDevice.current.userInterfaceIdiom == .phone {
            let mealPlan = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Meal plan")).firstMatch
            XCTAssertTrue(mealPlan.waitForExistence(timeout: 5))
            XCTAssertEqual(mealPlan.value as? String, "Monday–Sunday")
            mealPlan.tap()
        }
        let range = app.staticTexts["meal-week-range"]
        XCTAssertTrue(range.waitForExistence(timeout: 5))
        XCTAssertTrue(range.label.contains("Monday"), range.label)
        XCTAssertTrue(range.label.contains("Sunday"), range.label)
        let planner = XCTAttachment(screenshot: app.screenshot())
        planner.name = "Configured meal week"
        planner.lifetime = .keepAlways
        add(planner)
    }
}
