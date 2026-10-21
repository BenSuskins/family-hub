import XCTest
import UIKit

final class HouseholdUITests: XCTestCase {
    @MainActor
    func testWeatherAndShoppingInDemo() {
        let app = XCUIApplication()
        app.launch()
        let demo = app.buttons["Try Demo"]
        reveal(demo, in: app)
        XCTAssertTrue(demo.waitForExistence(timeout: 10))
        demo.tap()
        let weather = app.buttons["home-weather"]
        reveal(weather, in: app)
        XCTAssertTrue(weather.waitForExistence(timeout: 10))
        let home = XCTAttachment(screenshot: app.screenshot())
        home.name = "household-home"
        home.lifetime = .keepAlways
        add(home)
        weather.tap()
        XCTAssertTrue(app.navigationBars["Home weather"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Take a raincoat."].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Next 24 hours"].exists)
        let forecast = XCTAttachment(screenshot: app.screenshot())
        forecast.name = "household-weather"
        forecast.lifetime = .keepAlways
        add(forecast)
        app.navigationBars.buttons["BackButton"].tap()
        selectHousehold(in: app)
        XCTAssertTrue(app.navigationBars["Household"].waitForExistence(timeout: 5))
        openShoppingListIfNeeded(in: app)
        let field = app.textFields["Item name"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
        field.typeText("Test bread")
        app.buttons["Add item"].tap()
        XCTAssertTrue(app.staticTexts["Test bread"].waitForExistence(timeout: 5))
        app.buttons["Purchase Test bread"].tap()
        XCTAssertTrue(app.buttons["Restore Test bread"].waitForExistence(timeout: 5))
        let list = XCTAttachment(screenshot: app.screenshot())
        list.name = "household-shopping"
        list.lifetime = .keepAlways
        add(list)
    }
}

extension HouseholdUITests {
    @MainActor
    func testDeniedLocationKeepsHomeSettingsAvailable() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 5))
        app.buttons["Open profile"].tap()
        let homeInfo = app.buttons["Home location"]
        reveal(homeInfo, in: app)
        XCTAssertTrue(homeInfo.waitForExistence(timeout: 5))
        homeInfo.tap()
        XCTAssertTrue(app.navigationBars["Home location"].waitForExistence(timeout: 5))
        let capture = app.buttons["Use device location"]
        reveal(capture, in: app)
        XCTAssertTrue(capture.waitForExistence(timeout: 5))
        capture.tap()
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        let deny = springboard.alerts.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Don")).firstMatch
        if deny.waitForExistence(timeout: 3) { deny.tap() }
        let explanation = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "Allow location access")).firstMatch
        XCTAssertTrue(explanation.waitForExistence(timeout: 5))
        XCTAssertTrue(app.navigationBars["Home location"].exists)
    }

    @MainActor
    func testWeatherWithLargeText() {
        defer { XCUIDevice.shared.orientation = .portrait }
        let app = XCUIApplication()
        app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        app.launch()
        let demo = app.buttons["Try Demo"]
        reveal(demo, in: app)
        XCTAssertTrue(demo.waitForExistence(timeout: 10))
        demo.tap()
        let weather = app.buttons["home-weather"]
        reveal(weather, in: app)
        XCTAssertTrue(weather.waitForExistence(timeout: 10))
        if UIDevice.current.userInterfaceIdiom == .pad {
            XCUIDevice.shared.orientation = .landscapeLeft
        }
        XCTAssertGreaterThanOrEqual(weather.frame.minX, 0)
        XCTAssertLessThanOrEqual(weather.frame.maxX, app.windows.firstMatch.frame.maxX)
        if !weather.isHittable { app.swipeUp() }
        let home = XCTAttachment(screenshot: XCUIScreen.main.screenshot())
        home.name = "household-large-text"
        home.lifetime = .keepAlways
        add(home)
        weather.tap()
        XCTAssertTrue(app.navigationBars["Home weather"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Take a raincoat."].waitForExistence(timeout: 5))
        let forecast = XCTAttachment(screenshot: XCUIScreen.main.screenshot())
        forecast.name = "weather-large-text"
        forecast.lifetime = .keepAlways
        add(forecast)
    }
}

@MainActor
private func reveal(_ element: XCUIElement, in app: XCUIApplication) {
    for _ in 0..<12 {
        if element.exists && element.isHittable && element.frame.midY > 130 && element.frame.midY < app.frame.height - 80 { return }
        let towardTop = element.exists && element.frame.midY < 130
        let start = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.45 : 0.75))
        let end = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.75 : 0.45))
        start.press(forDuration: 0.05, thenDragTo: end)
    }
}

@MainActor
private func selectHousehold(in app: XCUIApplication) {
    let tabs = app.tabBars.buttons.matching(NSPredicate(format: "label == %@", "Household")).allElementsBoundByIndex
    if let tab = tabs.first(where: { $0.isHittable }) { tab.tap(); return }
    let buttons = app.buttons.matching(NSPredicate(format: "label == %@", "Household")).allElementsBoundByIndex
    if let button = buttons.first(where: { $0.isHittable }) { button.tap(); return }
    let sidebar = app.buttons["ToggleSideBar"]
    if sidebar.exists && sidebar.isHittable { sidebar.tap() }
    app.collectionViews.staticTexts["Household"].tap()
}
