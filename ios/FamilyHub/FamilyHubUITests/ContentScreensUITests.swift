import XCTest
import UIKit

final class ContentScreensUITests: XCTestCase {
    @MainActor
    func testLowStockAdditionCanBePurchasedAndRestocked() {
        continueAfterFailure = false
        let app = launchDemo()
        selectTab("Household", in: app)
        let addStock = app.buttons["Add Fabric softener to shopping list"]
        reveal(addStock, in: app)
        addStock.tap()

        let purchase = app.buttons["Purchase Fabric softener"]
        reveal(purchase, in: app, towardTop: true)
        purchase.tap()
        XCTAssertTrue(app.navigationBars["Restock inventory"].waitForExistence(timeout: 5))
        let amount = app.textFields["New level (%)"]
        amount.tap()
        amount.typeText("80")
        app.navigationBars["Restock inventory"].buttons["Restock"].tap()

        XCTAssertTrue(app.staticTexts["Restocked"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["Restore Fabric softener"].exists)
    }

    @MainActor
    func testContentAndMealEntryWithAccessibilityText() {
        continueAfterFailure = false
        if UIDevice.current.userInterfaceIdiom == .pad { XCUIDevice.shared.orientation = .landscapeLeft }
        defer { XCUIDevice.shared.orientation = .portrait }
        let app = launchDemo(largeText: true)
        selectTab("Food", in: app)
        let date = app.descendants(matching: .any).matching(identifier: "meal-date-picker").firstMatch
        reveal(date, in: app)
        let breakfast = app.buttons["Add breakfast"]
        reveal(breakfast, in: app)
        breakfast.tap()
        let meal = app.textFields["Meal name"]
        XCTAssertTrue(meal.waitForExistence(timeout: 5))
        meal.tap()
        meal.typeText("Soup and bread\n")
        XCTAssertTrue(app.buttons["Edit breakfast: Soup and bread"].waitForExistence(timeout: 5))

        selectTab("Household", in: app)
        let item = app.textFields["Item name"]
        reveal(item, in: app)
        item.tap()
        item.typeText("Apples\n")
        let purchase = app.buttons["Purchase Apples"]
        reveal(purchase, in: app)
        XCTAssertGreaterThanOrEqual(purchase.frame.height, 44)
        purchase.tap()
        XCTAssertTrue(app.buttons["Restore Apples"].waitForExistence(timeout: 5))
        let inventory = app.buttons["Inventory"]
        reveal(inventory, in: app, towardTop: true)
        inventory.tap()
        XCTAssertTrue(app.navigationBars["Inventory"].waitForExistence(timeout: 5))
    }

    @MainActor
    private func launchDemo(largeText: Bool = false) -> XCUIApplication {
        let app = XCUIApplication()
        if largeText {
            app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        }
        app.launch()
        let demo = app.buttons["Try Demo"]
        reveal(demo, in: app)
        demo.tap()
        return app
    }

    @MainActor
    private func selectTab(_ name: String, in app: XCUIApplication) {
        let tabs = app.tabBars.buttons.matching(NSPredicate(format: "label == %@", name)).allElementsBoundByIndex
        if let tab = tabs.first(where: { $0.isHittable }) { tab.tap(); return }
        let buttons = app.buttons.matching(NSPredicate(format: "label == %@", name)).allElementsBoundByIndex
        if let button = buttons.first(where: { $0.isHittable }) { button.tap(); return }
        let sidebar = app.buttons["ToggleSidebar"]
        if sidebar.exists && sidebar.isHittable { sidebar.tap() }
        let label = app.collectionViews.staticTexts.matching(NSPredicate(format: "label == %@", name)).firstMatch
        XCTAssertTrue(label.waitForExistence(timeout: 5))
        label.tap()
    }

    @MainActor
    private func reveal(_ element: XCUIElement, in app: XCUIApplication, towardTop: Bool = false) {
        var scrollTowardTop = towardTop
        for _ in 0..<20 {
            if element.exists && element.isHittable {
                let top = app.navigationBars.firstMatch.exists ? app.navigationBars.firstMatch.frame.maxY : 0
                let bottom = app.tabBars.firstMatch.exists ? app.tabBars.firstMatch.frame.minY : app.windows.firstMatch.frame.maxY
                if element.frame.midY > top && element.frame.midY < bottom { return }
                scrollTowardTop = element.frame.midY <= top
            }
            let start = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: scrollTowardTop ? 0.4 : 0.7))
            let end = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: scrollTowardTop ? 0.7 : 0.4))
            start.press(forDuration: 0.1, thenDragTo: end)
        }
        XCTAssertTrue(element.waitForExistence(timeout: 5))
        XCTAssertTrue(element.isHittable)
    }
}
