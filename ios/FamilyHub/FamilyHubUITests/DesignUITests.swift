import XCTest
import UIKit

final class DesignUITests: XCTestCase {
    @MainActor
    func testScreensInDemo() {
        inspectScreens(largeText: false)
    }

    @MainActor
    func testScreensWithAccessibilityText() {
        inspectScreens(largeText: true)
    }

    @MainActor
    private func inspectScreens(largeText: Bool) {
        continueAfterFailure = false
        if UIDevice.current.userInterfaceIdiom == .pad {
            XCUIDevice.shared.orientation = .landscapeLeft
        }
        defer { XCUIDevice.shared.orientation = .portrait }
        let app = XCUIApplication()
        app.launchArguments = ["-UIPreferredContentSizeCategoryName",
                               largeText ? "UICTContentSizeCategoryAccessibilityXXXL" : "UICTContentSizeCategoryL"]
        app.launch()
        capture(app, "login")
        tap(app.buttons["Try Demo"], in: app)
        XCTAssertTrue(app.navigationBars["Today"].waitForExistence(timeout: 5))
        capture(app, "today")

        tap(app.buttons["Open profile"], in: app)
        XCTAssertTrue(app.navigationBars["Settings"].waitForExistence(timeout: 5))
        capture(app, "settings")
        tap(app.buttons["Home Info"], in: app)
        capture(app, "home-info")
        back(app)
        tap(app.buttons["Family Name"], in: app)
        capture(app, "family-name")
        back(app)
        tap(app.buttons["Family Members"], in: app)
        capture(app, "members")
        back(app)
        tap(app.buttons["Categories"], in: app)
        capture(app, "categories")
        back(app)
        tap(app.buttons["API Tokens"], in: app)
        capture(app, "tokens")
        tap(app.buttons["New Token"], in: app)
        capture(app, "token-form")
        let tokenName = app.textFields["Name"]
        tap(tokenName, in: app)
        tokenName.typeText("Screen Test")
        tap(app.buttons["Save"], in: app)
        tap(app.buttons["Copy Token"], in: app)
        capture(app, "token-created")
        tap(app.buttons["Done"], in: app)
        back(app)
        tap(app.buttons["Done"], in: app)

        select("Calendar", in: app)
        capture(app, "calendar")
        select("Tasks", in: app)
        capture(app, "chores")
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Vacuum living room")).firstMatch, in: app)
        capture(app, "chore-detail")
        back(app)
        tap(app.buttons["New Chore"], in: app)
        capture(app, "chore-form")
        tap(app.buttons["Cancel"], in: app)

        select("Food", in: app)
        capture(app, "meal-plan")
        let meal = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Add breakfast")).firstMatch
        tap(meal, in: app)
        capture(app, "meal-form")
        tap(app.buttons["Cancel"], in: app)
        tap(app.buttons["Recipes"], in: app)
        capture(app, "recipes")
        tap(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch, in: app)
        XCTAssertTrue(app.buttons["Start Cook Mode"].waitForExistence(timeout: 5))
        capture(app, "recipe-detail")
        tap(app.buttons["Start Cook Mode"], in: app)
        capture(app, "cook-ingredients")
        tap(app.buttons["Next"], in: app)
        capture(app, "cook-step")
        tap(app.buttons["Close"], in: app)
        back(app)
        tap(app.buttons["New Recipe"], in: app)
        capture(app, "recipe-form")
        tap(app.buttons["Cancel"], in: app)

        select("Household", in: app)
        capture(app, "shopping")
        tap(app.buttons["Inventory"], in: app)
        capture(app, "inventory")
        tap(app.buttons["New Area"], in: app)
        capture(app, "area-form")
        tap(app.buttons["Cancel"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch, in: app)
        capture(app, "area-detail")
        tap(app.buttons["New Item"], in: app)
        capture(app, "item-form")
        tap(app.buttons["Cancel"], in: app)
    }

    @MainActor
    private func select(_ name: String, in app: XCUIApplication) {
        let tabs = app.tabBars.buttons.matching(identifier: name).allElementsBoundByIndex
        if let tab = tabs.first(where: { $0.isHittable }) {
            tab.tap()
            return
        }
        let buttons = app.buttons.matching(identifier: name).allElementsBoundByIndex
        if let button = buttons.first(where: { $0.isHittable }) {
            button.tap()
            return
        }
        let labels = app.collectionViews.staticTexts.matching(identifier: name).allElementsBoundByIndex
        if let label = labels.first(where: { $0.isHittable }) {
            label.tap()
            return
        }
        let sidebar = app.buttons["ToggleSidebar"]
        if sidebar.exists && sidebar.isHittable { sidebar.tap() }
        let label = app.collectionViews.staticTexts.matching(identifier: name).firstMatch
        tap(label, in: app)
    }

    @MainActor
    private func tap(_ element: XCUIElement, in app: XCUIApplication) {
        for _ in 0..<20 {
            let bottom = app.tabBars.firstMatch.exists ? app.tabBars.firstMatch.frame.minY : app.windows.firstMatch.frame.maxY
            if element.exists && element.isHittable && element.frame.midY < bottom { break }
            let towardTop = element.exists && element.frame.midY < 0
            let start = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.4 : 0.7))
            let end = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.7 : 0.4))
            start.press(forDuration: 0.1, thenDragTo: end)
        }
        XCTAssertTrue(element.waitForExistence(timeout: 5))
        element.tap()
    }

    @MainActor
    private func back(_ app: XCUIApplication) {
        app.navigationBars.buttons["BackButton"].tap()
    }

    @MainActor
    private func capture(_ app: XCUIApplication, _ name: String) {
        var previousFrames: [CGRect] = []
        let stableLayout = NSPredicate { _, _ in
            let frames = app.navigationBars.allElementsBoundByIndex.map(\.frame)
            defer { previousFrames = frames }
            return !frames.isEmpty && frames == previousFrames
        }
        let settled = XCTNSPredicateExpectation(predicate: stableLayout, object: app)
        XCTAssertEqual(XCTWaiter.wait(for: [settled], timeout: 5), .completed)
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
