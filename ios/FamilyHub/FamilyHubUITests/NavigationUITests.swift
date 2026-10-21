import XCTest

final class NavigationUITests: XCTestCase {
    @MainActor
    func testTaskGroupsKeepTheirDestinations() {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()

        XCTAssertTrue(app.buttons["Today"].waitForExistence(timeout: 10))
        selectFamilyTab("Tasks", in: app)
        XCTAssertTrue(app.navigationBars["Chores"].waitForExistence(timeout: 5))
        selectFamilyTab("Food", in: app)
        XCTAssertTrue(app.staticTexts["Meal plan"].waitForExistence(timeout: 5))
        app.buttons["Recipes"].tap()
        XCTAssertTrue(app.navigationBars["Recipes"].waitForExistence(timeout: 5))
        selectFamilyTab("Household", in: app)
        XCTAssertTrue(app.navigationBars["Household"].waitForExistence(timeout: 5))
        app.buttons["Inventory"].tap()
        XCTAssertTrue(app.navigationBars["Inventory"].waitForExistence(timeout: 5))
        selectFamilyTab("Food", in: app)
        XCTAssertTrue(app.navigationBars["Recipes"].waitForExistence(timeout: 5))
        selectFamilyTab("Household", in: app)
        XCTAssertTrue(app.navigationBars["Inventory"].waitForExistence(timeout: 5))
    }
}
