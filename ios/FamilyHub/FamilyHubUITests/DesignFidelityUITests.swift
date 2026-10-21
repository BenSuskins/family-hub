import XCTest

final class DesignFidelityUITests: XCTestCase {
    @MainActor
    func testRecipeDraftSurvivesContentNavigationAndSaves() {
        continueAfterFailure = false
        let app = launchDemo()
        openRecipe(in: app)
        app.buttons["Recipe actions"].tap()
        app.buttons["Edit"].tap()
        let ingredients = app.buttons["Edit recipe ingredients"]
        reveal(ingredients, in: app)
        XCTAssertTrue(ingredients.exists)
        ingredients.tap()
        XCTAssertTrue(app.navigationBars["Ingredients"].waitForExistence(timeout: 5))
        let field = app.textViews.firstMatch
        let originalIngredients = field.value as? String ?? ""
        field.coordinate(withNormalizedOffset: CGVector(dx: 0.95, dy: 0.9)).tap()
        field.typeText("\n1 lime")
        XCTAssertEqual(field.value as? String, originalIngredients + "\n1 lime")
        app.buttons["Done"].tap()
        let method = app.buttons["Edit recipe method"]
        reveal(method, in: app)
        method.tap()
        XCTAssertTrue(app.navigationBars["Method"].waitForExistence(timeout: 5))
        app.descendants(matching: .any).matching(identifier: "Step 1").firstMatch.tap()
        app.descendants(matching: .any).matching(identifier: "Step 1").firstMatch.typeText(" Carefully.")
        app.buttons["Done"].tap()
        reveal(ingredients, in: app)
        ingredients.tap()
        XCTAssertTrue((app.textViews.firstMatch.value as? String)?.contains("1 lime") == true)
        capture(app, "recipe-ingredients-draft")
        app.buttons["Done"].tap()
        capture(app, "recipe-editor-summary")
        app.buttons["Save"].tap()
        let lime = app.buttons["Add 1 lime to shopping list"]
        reveal(lime, in: app)
        XCTAssertTrue(lime.exists)
        capture(app, "recipe-saved-lime")
    }

    @MainActor
    func testCookingProgressTracksButtonsAndSwipes() {
        continueAfterFailure = false
        let app = launchDemo()
        openRecipe(in: app)
        app.buttons["Start Cook Mode"].tap()
        app.buttons["Start step 1"].tap()
        let progress = app.progressIndicators["Cooking progress"]
        XCTAssertTrue(progress.waitForExistence(timeout: 5))
        XCTAssertEqual(progress.value as? String, "Step 1 of 4")
        XCTAssertTrue(app.staticTexts["Next: Add chicken strips and cook until golden, about 5 minutes."].exists)
        capture(app, "cooking-step-progress")
        app.buttons["Next step"].tap()
        XCTAssertEqual(progress.value as? String, "Step 2 of 4")
        app.swipeLeft()
        XCTAssertTrue(app.staticTexts["Step 3 of 4"].waitForExistence(timeout: 5))
        XCTAssertEqual(progress.value as? String, "Step 3 of 4")
        app.buttons["Next step"].tap()
        XCTAssertTrue(app.staticTexts["This is the last step."].waitForExistence(timeout: 5))
        capture(app, "cooking-last-step")
        app.buttons["Ingredients"].tap()
        XCTAssertTrue(app.staticTexts["Before you start"].waitForExistence(timeout: 5))
        app.buttons["Close"].tap()
    }

    @MainActor
    func testInventoryFilterKeepsShoppingAndStockActionsAvailable() {
        continueAfterFailure = false
        let app = launchDemo()
        selectFamilyTab("Household", in: app)
        app.buttons["Inventory"].tap()
        capture(app, "inventory-soft-icons")
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch.tap()
        let filters = app.segmentedControls.firstMatch
        XCTAssertTrue(filters.waitForExistence(timeout: 5))
        filters.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Running low")).firstMatch.tap()
        XCTAssertFalse(app.staticTexts["Washing tablets"].exists)
        XCTAssertTrue(app.staticTexts["Fabric softener"].exists)
        XCTAssertTrue(app.buttons["Add Fabric softener to shopping list"].staticTexts["Add to Shopping"].exists)
        capture(app, "inventory-low-filter")
        app.buttons["Add Fabric softener to shopping list"].tap()
        XCTAssertTrue(app.alerts.staticTexts["Added Fabric softener to the shopping list."].waitForExistence(timeout: 5))
        app.alerts.buttons["OK"].tap()
        filters.buttons["All items"].tap()
        XCTAssertTrue(app.staticTexts["Washing tablets"].exists)
        XCTAssertTrue(app.buttons["Increase Washing tablets"].exists)
        capture(app, "inventory-area-actions")
        app.navigationBars.buttons["BackButton"].tap()
        app.buttons["New Area"].tap()
        app.textFields["Name"].typeText("Spare cupboard")
        app.buttons["Save"].tap()
        let area = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Spare cupboard")).firstMatch
        reveal(area, in: app)
        area.tap()
        XCTAssertTrue(app.staticTexts["No items yet"].waitForExistence(timeout: 5))
        XCTAssertGreaterThanOrEqual(app.buttons["Add item"].frame.width, 280)
        capture(app, "inventory-empty-area")
        app.buttons["Add item"].tap()
        XCTAssertTrue(app.navigationBars["New Item"].waitForExistence(timeout: 5))
    }

    @MainActor
    func testWelcomeHelpAndInvalidAddressKeepTheUserInput() {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.staticTexts["Your family,\nin one place."].waitForExistence(timeout: 5))
        capture(app, "welcome")
        reveal(app.buttons["Setup Help"], in: app)
        app.buttons["Setup Help"].tap()
        XCTAssertTrue(app.staticTexts["Join your family"].waitForExistence(timeout: 5))
        capture(app, "join-family")
        app.buttons["Back to Welcome"].tap()
        let address = app.textFields["Server URL"]
        address.tap()
        address.typeText("invalid")
        app.buttons["Connect"].tap()
        XCTAssertTrue(app.staticTexts["Server connection error"].waitForExistence(timeout: 5))
        XCTAssertEqual(address.value as? String, "invalid")
        capture(app, "welcome-invalid-address")
    }

    @MainActor
    func testFloatingTabsLeaveFinalRowsReachableAndHideForTheKeyboard() {
        continueAfterFailure = false
        let app = launchDemo()
        capture(app, "today-floating-tabs")
        selectFamilyTab("Food", in: app)
        app.buttons["Recipes"].tap()
        let create = app.buttons["New recipe"]
        reveal(create, in: app)
        app.swipeUp()
        XCTAssertTrue(create.isHittable)
        XCTAssertLessThanOrEqual(create.frame.maxY, app.buttons["Today"].frame.minY)
        capture(app, "recipes-floating-tabs-last-row")
        selectFamilyTab("Household", in: app)
        openShoppingListIfNeeded(in: app)
        let field = app.textFields["Item name"]
        field.tap()
        XCTAssertTrue(app.keyboards.firstMatch.waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["Today"].isHittable)
        capture(app, "shopping-keyboard-no-tabs")
    }

    @MainActor
    func testLargestTextKeepsContentEditorsCookingAndInventoryUsable() {
        continueAfterFailure = false
        let app = launchDemo(largeText: true)
        capture(app, "large-today-native-tabs")
        openRecipe(in: app)
        app.buttons["Recipe actions"].tap()
        app.buttons["Edit"].tap()
        let method = app.buttons["Edit recipe method"]
        reveal(method, in: app)
        method.tap()
        XCTAssertTrue(app.navigationBars["Method"].waitForExistence(timeout: 5))
        capture(app, "large-method-editor")
        app.buttons["Done"].tap()
        app.buttons["Cancel"].tap()
        app.buttons["Start Cook Mode"].tap()
        reveal(app.buttons["Start step 1"], in: app)
        app.buttons["Start step 1"].tap()
        XCTAssertGreaterThanOrEqual(app.buttons["Ingredients"].frame.width, 280)
        XCTAssertGreaterThanOrEqual(app.buttons["Previous step"].frame.width, 280)
        capture(app, "large-cooking-progress")
        app.buttons["Close"].tap()
        selectFamilyTab("Household", in: app)
        reveal(app.buttons["Inventory"], in: app)
        app.buttons["Inventory"].tap()
        let area = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch
        reveal(area, in: app)
        area.tap()
        capture(app, "large-inventory-actions")
        let add = app.buttons["Add Washing tablets to shopping list"]
        reveal(add, in: app)
        XCTAssertTrue(add.isHittable)
        XCTAssertGreaterThanOrEqual(add.frame.height, 44)
        capture(app, "large-inventory-shopping-action")
    }

    @MainActor
    private func launchDemo(largeText: Bool = false) -> XCUIApplication {
        let app = XCUIApplication()
        if largeText { app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"] }
        app.launch()
        let demo = app.buttons["Try Demo"]
        XCTAssertTrue(demo.waitForExistence(timeout: 10))
        reveal(demo, in: app)
        demo.tap()
        return app
    }

    @MainActor
    private func openRecipe(in app: XCUIApplication) {
        selectFamilyTab("Food", in: app)
        reveal(app.buttons["Recipes"], in: app)
        app.buttons["Recipes"].tap()
        let recipe = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch
        XCTAssertTrue(recipe.waitForExistence(timeout: 5))
        reveal(recipe, in: app)
        recipe.tap()
        XCTAssertTrue(app.buttons["Start Cook Mode"].waitForExistence(timeout: 5))
    }

    @MainActor
    private func reveal(_ element: XCUIElement, in app: XCUIApplication) {
        _ = element.waitForExistence(timeout: 3)
        for _ in 0..<8 {
            if element.exists && element.isHittable { return }
            let start = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.75))
            let end = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.45))
            start.press(forDuration: 0.05, thenDragTo: end)
        }
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
