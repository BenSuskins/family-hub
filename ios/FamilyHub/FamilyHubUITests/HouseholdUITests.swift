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
        let weather = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Weather at home")).firstMatch
        reveal(weather, in: app)
        XCTAssertTrue(weather.waitForExistence(timeout: 10))
        let loadedWeather = app.staticTexts["Take a raincoat."]
        reveal(loadedWeather, in: app)
        XCTAssertTrue(loadedWeather.waitForExistence(timeout: 5))
        let home = XCTAttachment(screenshot: app.screenshot())
        home.name = "household-home"
        home.lifetime = .keepAlways
        add(home)
        weather.tap()
        XCTAssertTrue(app.navigationBars["Weather"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Next 24 hours"].exists)
        let forecast = XCTAttachment(screenshot: app.screenshot())
        forecast.name = "household-weather"
        forecast.lifetime = .keepAlways
        add(forecast)
        app.navigationBars.buttons["BackButton"].tap()
        let shopping = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Shopping List")).firstMatch
        reveal(shopping, in: app)
        XCTAssertTrue(shopping.waitForExistence(timeout: 5))
        shopping.tap()
        XCTAssertTrue(app.navigationBars["Shopping List"].waitForExistence(timeout: 5))
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
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.buttons["Try Demo"].waitForExistence(timeout: 10))
        app.buttons["Try Demo"].tap()
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 5))
        app.buttons["Open profile"].tap()
        let homeInfo = app.buttons["Home Info"]
        reveal(homeInfo, in: app)
        XCTAssertTrue(homeInfo.waitForExistence(timeout: 5))
        if !homeInfo.isHittable { app.swipeUp() }
        homeInfo.tap()
        let capture = app.buttons["Use device location"]
        XCTAssertTrue(capture.waitForExistence(timeout: 5))
        capture.tap()
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        let deny = springboard.alerts.buttons.matching(NSPredicate(format: "label CONTAINS[c] %@", "Don")).firstMatch
        if deny.waitForExistence(timeout: 3) { deny.tap() }
        let explanation = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "Allow location access")).firstMatch
        XCTAssertTrue(explanation.waitForExistence(timeout: 5))
        XCTAssertTrue(app.navigationBars["Home Info"].exists)
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
        let weather = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Weather at home")).firstMatch
        reveal(weather, in: app)
        XCTAssertTrue(weather.waitForExistence(timeout: 10))
        let loadedWeather = app.staticTexts["Take a raincoat."]
        reveal(loadedWeather, in: app)
        XCTAssertTrue(loadedWeather.waitForExistence(timeout: 5))
        if UIDevice.current.userInterfaceIdiom == .pad {
            XCUIDevice.shared.orientation = .landscapeLeft
            let landscape = NSPredicate { _, _ in
                app.windows.firstMatch.frame.width > app.windows.firstMatch.frame.height
            }
            expectation(for: landscape, evaluatedWith: app)
            waitForExpectations(timeout: 5)
        }
        if !weather.isHittable { app.swipeUp() }
        let home = XCTAttachment(screenshot: XCUIScreen.main.screenshot())
        home.name = "household-large-text"
        home.lifetime = .keepAlways
        add(home)
        weather.tap()
        XCTAssertTrue(app.navigationBars["Weather"].waitForExistence(timeout: 5))
        let forecast = XCTAttachment(screenshot: XCUIScreen.main.screenshot())
        forecast.name = "weather-large-text"
        forecast.lifetime = .keepAlways
        add(forecast)
    }
}

@MainActor
private func reveal(_ element: XCUIElement, in app: XCUIApplication) {
    for _ in 0..<12 {
        if element.exists && element.isHittable { return }
        app.swipeUp()
    }
}
