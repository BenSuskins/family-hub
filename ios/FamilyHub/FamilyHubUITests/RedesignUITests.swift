import XCTest
import UIKit

final class RedesignUITests: XCTestCase {
    @MainActor
    func testShoppingRenameStaysInlineAndPurchaseCanBeUndoneBeforeItHides() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        app.buttons["Try Demo"].tap()
        capture(app, "small-today")
        selectFamilyTab("Household", in: app)
        capture(app, "small-household")
        openShoppingListIfNeeded(in: app)
        app.staticTexts["Milk"].tap()
        let field = app.textFields["Rename item inline"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        XCTAssertTrue(app.navigationBars["Shopping List"].exists)
        capture(app, "small-shopping-inline")
        app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "items to buy")).firstMatch.tap()
        XCTAssertFalse(app.keyboards.firstMatch.waitForExistence(timeout: 2))
        app.staticTexts["Milk"].tap()
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: 4) + "Oat milk")
        app.buttons["Save item name"].tap()
        let purchase = app.buttons["Purchase Oat milk"]
        XCTAssertTrue(purchase.waitForExistence(timeout: 5))
        purchase.tap()
        XCTAssertTrue(app.buttons["Undo"].waitForExistence(timeout: 2))
        XCTAssertFalse(app.staticTexts["Purchased"].exists)
        capture(app, "small-shopping-undo")
        app.buttons["Undo"].tap()
        XCTAssertTrue(purchase.waitForExistence(timeout: 5))
        purchase.tap()
        XCTAssertTrue(app.buttons["Restore Oat milk"].waitForNonExistence(timeout: 8))
        capture(app, "small-shopping-hidden")
        XCTAssertFalse(app.staticTexts["Oat milk"].exists)
        XCTAssertFalse(app.navigationBars["Restock inventory"].exists)
    }
    @MainActor
    func testCookingAndEmptyTokensRemainReadableAtLargestText() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        app.launch()
        reveal(app.buttons["Try Demo"], in: app)
        app.buttons["Try Demo"].tap()
        app.buttons["Open profile"].tap()
        reveal(app.buttons["API Tokens"], in: app)
        app.buttons["API Tokens"].tap()
        let emptyState = app.cells.containing(.staticText, identifier: "No Tokens").firstMatch
        reveal(emptyState, in: app)
        XCTAssertTrue(emptyState.exists)
        capture(app, "large-tokens-empty")
        app.navigationBars.buttons["BackButton"].tap()
        app.buttons["Done"].tap()
        selectFamilyTab("Food", in: app)
        reveal(app.buttons["Recipes"], in: app)
        app.buttons["Recipes"].tap()
        let recipe = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch
        reveal(recipe, in: app)
        recipe.tap()
        reveal(app.buttons["Start cooking"], in: app)
        app.buttons["Start cooking"].tap()
        app.buttons["Start step 1"].tap()
        XCTAssertGreaterThanOrEqual(app.buttons["Ingredients"].frame.width, 280)
        XCTAssertGreaterThanOrEqual(app.buttons["Previous step"].frame.width, 280)
        capture(app, "large-cook-actions")
        app.buttons["Close"].tap()
        selectFamilyTab("Household", in: app)
        reveal(app.buttons["Inventory"], in: app)
        app.buttons["Inventory"].tap()
        let areaName = app.staticTexts["Laundry cupboard"].firstMatch
        reveal(areaName, in: app)
        XCTAssertGreaterThanOrEqual(areaName.frame.width, 200)
        capture(app, "large-inventory-rows")
    }

    @MainActor
    private func reveal(_ element: XCUIElement, in app: XCUIApplication) {
        _ = element.waitForExistence(timeout: 3)
        for _ in 0..<12 {
            if element.exists && element.isHittable { return }
            app.swipeUp()
        }
        XCTAssertTrue(element.exists)
    }

    @MainActor
    private func capture(_ app: XCUIApplication, _ name: String) {
        Thread.sleep(forTimeInterval: 0.4)
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
