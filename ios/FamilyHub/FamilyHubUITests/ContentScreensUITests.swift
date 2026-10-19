import XCTest
import UIKit

final class ContentScreensUITests: XCTestCase {
    @MainActor
    func testHomeAgendaLimitsTheDescriptionAndKeepsFullEventDetails() {
        continueAfterFailure = false
        let app = launchDemo(layoutFixtures: true)
        let description = app.staticTexts["home-event-description-demo-event-1"]
        XCTAssertTrue(description.waitForExistence(timeout: 5))
        let fullDescription = description.label
        capture(app, "home-agenda-description")
        XCTAssertLessThanOrEqual(description.frame.height, ceil(UIFont.preferredFont(forTextStyle: .footnote).lineHeight) * 2 + 4)
        app.buttons.containing(.staticText, identifier: "Family Dinner").firstMatch.tap()
        XCTAssertTrue(app.navigationBars["Event"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts.matching(NSPredicate(format: "label == %@", fullDescription)).firstMatch.exists)
    }

    @MainActor
    func testHomeAndIngredientActionsWithAccessibilityText() {
        continueAfterFailure = false
        if UIDevice.current.userInterfaceIdiom == .pad { XCUIDevice.shared.orientation = .landscapeLeft }
        defer { XCUIDevice.shared.orientation = .portrait }
        let app = launchDemo(largeText: true, mealImages: true)
        let weather = app.buttons["home-weather"]
        reveal(weather, in: app)
        let date = app.staticTexts["home-date"]
        XCTAssertGreaterThanOrEqual(weather.frame.minY, date.frame.maxY - 1)
        capture(app, "home-header-accessibility")
        let breakfast = app.buttons["today-meal-breakfast"]
        reveal(breakfast, in: app)
        XCTAssertLessThanOrEqual(breakfast.frame.maxX, app.windows.firstMatch.frame.maxX)
        capture(app, "home-meal-accessibility")
        breakfast.tap()
        let ingredient = app.buttons["Add 500g chicken breast, sliced to shopping list"]
        reveal(ingredient, in: app)
        XCTAssertGreaterThanOrEqual(ingredient.frame.width.rounded(), 44)
        XCTAssertGreaterThanOrEqual(ingredient.frame.height.rounded(), 44)
        ingredient.tap()
        XCTAssertTrue(app.buttons["Added 500g chicken breast, sliced to shopping list"].waitForExistence(timeout: 5))
        capture(app, "recipe-ingredient-accessibility")
    }

    @MainActor
    func testHomeKeepsWeatherInTheHeaderAndMealCardsEqual() {
        continueAfterFailure = false
        let app = launchDemo(mealImages: true)
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["Shopping List"].exists)
        XCTAssertFalse(app.buttons["Manage"].exists)
        let weather = app.buttons["home-weather"]
        XCTAssertTrue(weather.waitForExistence(timeout: 5))
        let date = app.staticTexts["home-date"]
        XCTAssertLessThan(abs(date.frame.midY - weather.frame.midY), 24)

        let breakfast = app.buttons["today-meal-breakfast"]
        reveal(breakfast, in: app)
        let lunch = app.buttons["today-meal-lunch"]
        let dinner = app.buttons["today-meal-dinner"]
        XCTAssertEqual(breakfast.frame.height, lunch.frame.height, accuracy: 1)
        XCTAssertEqual(breakfast.frame.height, dinner.frame.height, accuracy: 1)
        XCTAssertEqual(breakfast.frame.width, dinner.frame.width, accuracy: 1)
        capture(app, "home-meal-images")

        reveal(weather, in: app, towardTop: true)
        weather.tap()
        XCTAssertTrue(app.navigationBars["Weather"].waitForExistence(timeout: 5))
    }

    @MainActor
    func testIngredientAddsOneShoppingItemAndKeepsTheChecklist() {
        continueAfterFailure = false
        let app = launchDemo()
        selectTab("Food", in: app)
        let recipes = app.buttons["Recipes"]
        reveal(recipes, in: app)
        let breakfast = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Add breakfast")).firstMatch
        XCTAssertLessThan(breakfast.frame.minY, recipes.frame.minY)
        recipes.tap()
        let recipe = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch
        reveal(recipe, in: app)
        recipe.tap()
        let ingredient = app.buttons["Add 500g chicken breast, sliced to shopping list"]
        reveal(ingredient, in: app)
        XCTAssertGreaterThanOrEqual(ingredient.frame.height.rounded(), 44)
        ingredient.tap()
        let added = app.buttons["Added 500g chicken breast, sliced to shopping list"]
        XCTAssertTrue(added.waitForExistence(timeout: 5))
        XCTAssertFalse(added.isEnabled)
        let checklist = app.buttons["500g chicken breast, sliced"]
        checklist.tap()
        XCTAssertEqual(checklist.value as? String, "Checked")
        capture(app, "recipe-ingredient-added")

        selectTab("Household", in: app)
        let item = app.buttons["Purchase 500g chicken breast, sliced"]
        reveal(item, in: app)
        XCTAssertTrue(item.exists)
        XCTAssertFalse(app.buttons["Purchase 3 tbsp soy sauce"].exists)
        let inventory = app.buttons["Inventory"]
        reveal(inventory, in: app)
        XCTAssertLessThan(item.frame.minY, inventory.frame.minY)
        capture(app, "shopping-before-inventory")
        inventory.tap()
        XCTAssertTrue(app.navigationBars["Inventory"].waitForExistence(timeout: 5))
    }

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
        XCTAssertGreaterThanOrEqual(purchase.frame.height.rounded(), 44)
        purchase.tap()
        XCTAssertTrue(app.buttons["Restore Apples"].waitForExistence(timeout: 5))
        let inventory = app.buttons["Inventory"]
        reveal(inventory, in: app)
        inventory.tap()
        XCTAssertTrue(app.navigationBars["Inventory"].waitForExistence(timeout: 5))
    }

    @MainActor
    private func launchDemo(largeText: Bool = false, mealImages: Bool = false, layoutFixtures: Bool = false) -> XCUIApplication {
        let app = XCUIApplication()
        if largeText {
            app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        }
        if mealImages { app.launchArguments.append("-demo-meal-card-images") }
        if layoutFixtures { app.launchArguments.append("-calendar-layout-fixtures") }
        app.launch()
        let demo = app.buttons["Try Demo"]
        reveal(demo, in: app)
        demo.tap()
        return app
    }

    @MainActor
    private func capture(_ app: XCUIApplication, _ name: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
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
