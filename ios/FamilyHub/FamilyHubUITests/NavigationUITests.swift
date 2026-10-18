import XCTest

final class NavigationUITests: XCTestCase {
    @MainActor
    func testTaskGroupsKeepTheirDestinations() {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()

        let tabs = app.tabBars
        XCTAssertTrue(tabs.buttons["Today"].waitForExistence(timeout: 10))
        tabs.buttons["Tasks"].tap()
        XCTAssertTrue(app.navigationBars["Chores"].waitForExistence(timeout: 5))
        tabs.buttons["Food"].tap()
        app.buttons["Recipes"].tap()
        XCTAssertTrue(app.navigationBars["Recipes"].waitForExistence(timeout: 5))
        tabs.buttons["Household"].tap()
        app.buttons["Shopping List"].tap()
        XCTAssertTrue(app.navigationBars["Shopping List"].waitForExistence(timeout: 5))
        tabs.buttons["Food"].tap()
        XCTAssertTrue(app.navigationBars["Recipes"].waitForExistence(timeout: 5))
        tabs.buttons["Household"].tap()
        XCTAssertTrue(app.navigationBars["Shopping List"].waitForExistence(timeout: 5))
    }
}
