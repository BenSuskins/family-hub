import XCTest
import UIKit

final class MarketingScreenshotsUITests: XCTestCase {
    @MainActor
    func testCaptureMarketingScreens() {
        captureMarketingScreens(suffix: nil)
    }

    @MainActor
    func testCaptureMarketingScreensOnIPadLandscape() throws {
        try captureIPadScreens(orientation: .landscapeLeft, suffix: "ipad-landscape")
    }

    @MainActor
    func testCaptureMarketingScreensOnIPadPortrait() throws {
        try captureIPadScreens(orientation: .portrait, suffix: "ipad-portrait")
    }

    @MainActor
    private func captureIPadScreens(orientation: UIDeviceOrientation, suffix: String) throws {
        guard UIDevice.current.userInterfaceIdiom == .pad else {
            throw XCTSkip("This capture pass is for iPad.")
        }
        XCUIDevice.shared.orientation = orientation
        defer { XCUIDevice.shared.orientation = .portrait }
        captureMarketingScreens(suffix: suffix)
    }

    @MainActor
    private func captureMarketingScreens(suffix: String?) {
        continueAfterFailure = false

        let app = XCUIApplication()
        app.launchArguments = ["-marketing-screenshots", "-AppleInterfaceStyle", "Light"]
        app.launch()

        let demo = app.buttons["Try Demo"]
        for _ in 0..<6 where !demo.exists || !demo.isHittable {
            app.swipeUp()
        }
        XCTAssertTrue(demo.waitForExistence(timeout: 10))
        demo.tap()
        XCTAssertTrue(app.navigationBars["Today"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.staticTexts["Blueberry Pancakes"].waitForExistence(timeout: 10))
        capture(app, named: "01-home", suffix: suffix)
        app.scrollViews.firstMatch.swipeUp()
        XCTAssertTrue(app.staticTexts["Home → Work"].waitForExistence(timeout: 5))
        capture(app, named: "10-commute", suffix: suffix)

        selectFamilyTab("Calendar", in: app)
        XCTAssertTrue(app.navigationBars["Calendar"].waitForExistence(timeout: 5))
        capture(app, named: "06-calendar", suffix: suffix)

        selectFamilyTab("Chores", in: app)
        XCTAssertTrue(app.navigationBars["Chores"].waitForExistence(timeout: 5))
        capture(app, named: "07-chores", suffix: suffix)

        selectFamilyTab("Food", in: app)
        XCTAssertTrue(app.navigationBars["Food"].waitForExistence(timeout: 5))
        if UIDevice.current.userInterfaceIdiom == .phone {
            let mealPlan = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Meal plan")).firstMatch
            XCTAssertTrue(mealPlan.waitForExistence(timeout: 5))
            mealPlan.tap()
        }
        XCTAssertTrue(app.buttons["Previous week"].waitForExistence(timeout: 5))
        capture(app, named: "03-meals", suffix: suffix)

        selectFamilyTab("Food", in: app)
        let recipes = app.buttons["Recipes"]
        XCTAssertTrue(recipes.waitForExistence(timeout: 5))
        recipes.tap()
        XCTAssertTrue(app.navigationBars["Recipes"].waitForExistence(timeout: 5))
        capture(app, named: "04-recipes-book", suffix: suffix)
        let recipe = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch
        XCTAssertTrue(recipe.waitForExistence(timeout: 5))
        recipe.tap()
        XCTAssertTrue(app.buttons["Start cooking"].waitForExistence(timeout: 5))
        capture(app, named: "02-recipe-detail", suffix: suffix)

        selectFamilyTab("Household", in: app)
        XCTAssertTrue(app.navigationBars["Household"].waitForExistence(timeout: 5))
        capture(app, named: "08-household", suffix: suffix)
        openShoppingListIfNeeded(in: app)
        XCTAssertTrue(app.textFields["Item name"].waitForExistence(timeout: 5))
        capture(app, named: "09-shopping", suffix: suffix)

        selectFamilyTab("Household", in: app)
        let inventory = app.buttons["Inventory"]
        XCTAssertTrue(inventory.waitForExistence(timeout: 5))
        inventory.tap()
        XCTAssertTrue(app.navigationBars["Inventory"].waitForExistence(timeout: 5))
        capture(app, named: "05-inventory", suffix: suffix)
    }

    @MainActor
    private func capture(_ app: XCUIApplication, named name: String, suffix: String?) {
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = suffix.map { "\(name)-\($0)" } ?? name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
