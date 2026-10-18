import XCTest

final class MealEntryUITests: XCTestCase {
    @MainActor
    func testWeekViewKeepsTheSelectedWeek() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        app.tabBars.buttons["Food"].tap()
        app.buttons["Next week"].tap()
        let week = app.staticTexts["meal-week-range"].label

        app.buttons["View week"].tap()

        XCTAssertTrue(app.navigationBars["Meal Plan"].waitForExistence(timeout: 5))
        XCTAssertEqual(app.staticTexts["meal-week-range"].label, week)
    }

    @MainActor
    func testReturnSavesCustomMealAndRecipeSelectionKeepsTheLink() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        app.tabBars.buttons["Food"].tap()
        let breakfast = app.buttons["Add breakfast"]
        XCTAssertTrue(breakfast.waitForExistence(timeout: 5))
        breakfast.tap()

        let field = app.textFields["Meal name"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
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
        field.typeText("Toast and cheese\n")
        XCTAssertTrue(app.buttons["Edit breakfast: Toast and cheese"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["View recipe for Toast and cheese"].exists)
    }
}
