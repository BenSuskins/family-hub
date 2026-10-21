import XCTest
import UIKit

final class MealEntryUITests: XCTestCase {
    @MainActor
    func testWeekViewKeepsTheSelectedWeek() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        selectFamilyTab("Food", in: app)
        if UIDevice.current.userInterfaceIdiom == .phone { app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Meal plan")).firstMatch.tap() }
        app.buttons["Next week"].tap()
        let week = app.staticTexts["meal-week-range"].label

        // The overview opens the full meal plan directly on iPhone.
        if UIDevice.current.userInterfaceIdiom == .pad { app.buttons["View week"].tap() }

        XCTAssertTrue(app.navigationBars["Meal plan"].waitForExistence(timeout: 5))
        XCTAssertEqual(app.staticTexts["meal-week-range"].label, week)
    }

    @MainActor
    func testReturnSavesCustomMealAndRecipeSelectionKeepsTheLink() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        selectFamilyTab("Food", in: app)
        let breakfast = app.buttons.matching(NSPredicate(format: "label == %@ OR label BEGINSWITH %@", "Add breakfast", "Edit breakfast:")).firstMatch
        XCTAssertTrue(breakfast.waitForExistence(timeout: 5))
        breakfast.tap()

        let field = app.textFields["Meal name"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
        if let value = field.value as? String, value != "What is for breakfast?" { field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: value.count)) }
        field.typeText("Toast with jam\n")
        let customMeal = app.buttons["Edit breakfast: Toast with jam"]
        XCTAssertTrue(customMeal.waitForExistence(timeout: 5))
        customMeal.tap()
        XCTAssertEqual(field.value as? String, "Toast with jam")
        field.tap()
        field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: "Toast with jam".count))
        field.typeText("Chicken")
        app.buttons["Chicken Stir Fry"].tap()
        app.buttons["Save"].tap()

        let recipe = app.buttons["View recipe for Chicken Stir Fry"]
        XCTAssertTrue(recipe.waitForExistence(timeout: 5))
        recipe.tap()
        XCTAssertTrue(app.buttons["Start Cook Mode"].waitForExistence(timeout: 5))
        app.navigationBars.buttons["BackButton"].tap()
        app.buttons["Edit breakfast: Chicken Stir Fry"].tap()
        field.tap()
        field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: "Chicken Stir Fry".count))
        field.typeText("Toast and cheese")
        app.buttons["Use name only"].tap()
        app.buttons["Save"].tap()
        XCTAssertTrue(app.buttons["Edit breakfast: Toast and cheese"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["View recipe for Toast and cheese"].exists)
    }
}
