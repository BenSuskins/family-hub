import XCTest

// Keep visual capture runs separate from the normal regression suite.
final class CompleteMockupsAuditUITests: XCTestCase {
    @MainActor
    func testChoreScreens() {
        let app = demo()
        selectFamilyTab("Chores", in: app)
        capture(app, "chores-list")
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Vacuum living room")).firstMatch, in: app)
        capture(app, "chore-detail")
        app.swipeUp()
        capture(app, "chore-detail-history")
        tap(app.buttons["Edit Chore"], in: app)
        capture(app, "chore-edit")
        tap(app.buttons["Edit repeat rule"], in: app)
        capture(app, "chore-repeat-current")
        frequency("Weekly", in: app)
        capture(app, "chore-repeat-weekly")
        frequency("Monthly", in: app)
        capture(app, "chore-repeat-monthly")
        frequency("Custom", in: app)
        capture(app, "chore-repeat-custom")
        tap(app.switches["Ends On Date"], in: app, shouldTap: false)
        app.switches["Ends On Date"].coordinate(withNormalizedOffset: CGVector(dx: 0.93, dy: 0.5)).tap()
        XCTAssertEqual(app.switches["Ends On Date"].value as? String, "1")
        capture(app, "chore-repeat-end-date")
        tap(app.switches["Limit Occurrences"], in: app, shouldTap: false)
        app.switches["Limit Occurrences"].coordinate(withNormalizedOffset: CGVector(dx: 0.93, dy: 0.5)).tap()
        XCTAssertEqual(app.switches["Limit Occurrences"].value as? String, "1")
        capture(app, "chore-repeat-count")
        app.buttons["Done"].tap()
        app.navigationBars.buttons["Cancel"].tap()
        tap(app.buttons["Delete Chore"], in: app)
        capture(app, "chore-delete-confirmation")
        dismissPopover(app)
        back(app)
        app.buttons["New Chore"].tap()
        capture(app, "chore-create")
        let name = app.textFields["Name"]
        name.tap()
        name.typeText("Water plants")
        capture(app, "chore-create-keyboard")
    }

    @MainActor
    func testSettingsScreens() {
        let app = settings()
        capture(app, "settings-top")
        tap(app.buttons["Chore Reminders"], in: app)
        capture(app, "notifications-top")
        app.swipeUp()
        capture(app, "notifications-schedule")
        back(app)
        tap(app.buttons["Home location"], in: app)
        capture(app, "home-location")
        app.swipeUp()
        capture(app, "home-location-bottom")
        back(app)
        tap(app.buttons["Family Name"], in: app)
        capture(app, "family-name")
        app.textFields.firstMatch.tap()
        capture(app, "family-name-keyboard")
        back(app)
        tap(app.buttons["Server configuration"], in: app)
        capture(app, "server")
        let field = app.textFields.firstMatch
        field.tap()
        if let value = field.value as? String { field.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: value.count)) }
        field.typeText("invalid")
        capture(app, "server-keyboard")
        tap(app.buttons["Connect"], in: app)
        capture(app, "server-invalid")
        back(app)
        app.swipeUp()
        capture(app, "settings-bottom")
    }

    @MainActor
    func testManagementScreens() {
        let app = settings()
        tap(app.buttons["Family Members"], in: app)
        capture(app, "family-members")
        app.buttons["member-demo-user-2"].tap()
        capture(app, "member-role")
        app.buttons["Choose admin role"].tap()
        capture(app, "member-role-draft")
        app.buttons["Save"].tap()
        capture(app, "member-role-confirmation")
        dismissPopover(app)
        back(app)
        back(app)
        tap(app.buttons["Categories"], in: app)
        capture(app, "categories")
        app.buttons["Cleaning"].tap()
        app.textFields["Category name"].tap()
        capture(app, "category-edit-keyboard")
        app.textFields["Category name"].typeText("\n")
        capture(app, "category-edit")
        tap(app.buttons["Delete category"], in: app)
        capture(app, "category-delete-confirmation")
        dismissPopover(app)
        app.navigationBars.buttons["Cancel"].tap()
        app.buttons["New Category"].tap()
        capture(app, "category-create")
    }

    @MainActor
    func testMessageAndTokenScreens() {
        let app = settings()
        tap(app.buttons["Family message"], in: app)
        capture(app, "family-message-empty")
        let field = app.textFields["Family message"].exists ? app.textFields["Family message"] : app.textViews["Family message"]
        field.tap()
        field.typeText("Dinner is ready.")
        capture(app, "family-message-keyboard")
        app.buttons["Done"].firstMatch.tap()
        capture(app, "family-message-ready")
        tap(app.buttons["Send to Family"], in: app)
        XCTAssertTrue(app.staticTexts["No family iPhones have notifications enabled."].waitForExistence(timeout: 5))
        capture(app, "family-message-result")
        app.swipeUp()
        capture(app, "family-message-result-actions")
        back(app)
        back(app)
        tap(app.buttons["API Tokens"], in: app)
        capture(app, "tokens-empty")
        app.buttons["New Token"].tap()
        app.textFields["Token name"].tap()
        capture(app, "token-create-keyboard")
        app.textFields["Token name"].typeText("Kitchen display\n")
        capture(app, "token-create-ready")
        tap(app.buttons["Create token"], in: app)
        XCTAssertTrue(app.staticTexts["Save your token"].waitForExistence(timeout: 5))
        capture(app, "token-created")
        tap(app.buttons["Copy Token"], in: app)
        capture(app, "token-copied")
        app.buttons["Done"].tap()
        capture(app, "tokens-populated")
        app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Kitchen display")).firstMatch.tap()
        capture(app, "token-details")
        tap(app.buttons["Delete token"], in: app)
        capture(app, "token-delete-confirmation")
    }

    @MainActor
    func testMealWeatherAndEventScreens() {
        let app = demo()
        app.buttons["home-weather"].tap()
        capture(app, "weather")
        app.swipeUp()
        capture(app, "weather-hours")
        for _ in 0..<6 { app.swipeUp() }
        capture(app, "weather-provider")
        back(app)
        tap(app.buttons.containing(.staticText, identifier: "Family Dinner").firstMatch, in: app)
        capture(app, "event-details")
        app.swipeUp()
        capture(app, "event-details-bottom")
        back(app)
        selectFamilyTab("Food", in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Meal plan")).firstMatch, in: app)
        capture(app, "meal-week")
        app.swipeUp()
        capture(app, "meal-week-bottom")
        back(app)
        let meal = app.buttons.matching(NSPredicate(format: "label == %@ OR label BEGINSWITH %@", "Add breakfast", "Edit breakfast:")).firstMatch
        tap(meal, in: app)
        capture(app, "meal-editor")
        let name = app.textFields["Meal name"]
        name.tap()
        if let value = name.value as? String, value != "What is for breakfast?" { name.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: value.count)) }
        name.typeText("Chicken")
        capture(app, "meal-search-keyboard")
        tap(app.buttons["Chicken Stir Fry"], in: app)
        capture(app, "meal-recipe-selected")
    }

    @MainActor
    func testShoppingAndInventoryScreens() {
        let app = demo()
        selectFamilyTab("Household", in: app)
        openShoppingListIfNeeded(in: app)
        app.buttons["Actions for Milk"].tap()
        app.buttons["Rename"].tap()
        capture(app, "shopping-inline-rename")
        app.buttons["Cancel"].tap()
        app.buttons["Purchase Milk"].tap()
        capture(app, "shopping-undo")
        app.buttons["Undo"].tap()
        back(app)
        app.buttons["Inventory"].tap()
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch.tap()
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Washing tablets")).firstMatch.tap()
        capture(app, "inventory-count")
        app.buttons["Cancel"].tap()
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Fabric softener")).firstMatch.tap()
        capture(app, "inventory-level")
        app.buttons["Cancel"].tap()
        app.buttons["New Item"].tap()
        capture(app, "inventory-create-keyboard")
    }

    @MainActor
    func testLargestTextScreens() {
        let app = demo(largeText: true)
        selectFamilyTab("Chores", in: app)
        capture(app, "large-chores")
        tap(app.buttons["New Chore"], in: app)
        capture(app, "large-chore-form")
        tap(app.buttons["Edit repeat rule"], in: app)
        capture(app, "large-repeat")
        app.buttons["Done"].tap()
        app.buttons["Cancel"].tap()
        selectFamilyTab("Today", in: app)
        app.buttons["Open profile"].tap()
        tap(app.buttons["Chore Reminders"], in: app)
        capture(app, "large-notifications")
        app.swipeUp()
        capture(app, "large-notifications-bottom")
        back(app)
        tap(app.buttons["Family Members"], in: app)
        capture(app, "large-members")
        tap(app.buttons["member-demo-user-2"], in: app)
        capture(app, "large-role")
        back(app)
        back(app)
        tap(app.buttons["Family message"], in: app)
        capture(app, "large-family-message")
        tap(app.buttons["Send to Family"], in: app, shouldTap: false)
        capture(app, "large-family-message-action")
        back(app)
        app.buttons["Done"].tap()
        selectFamilyTab("Household", in: app)
        tap(app.buttons["Inventory"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch, in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Washing tablets")).firstMatch, in: app)
        capture(app, "large-inventory-count")
        app.swipeUp()
        capture(app, "large-inventory-count-bottom")
    }

    @MainActor
    func testLargeInventoryFormScreens() {
        let app = demo(largeText: true)
        selectFamilyTab("Household", in: app)
        tap(app.buttons["Inventory"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch, in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Washing tablets")).firstMatch, in: app)
        capture(app, "large-inventory-count")
        let quantity = app.cells.containing(.textField, identifier: "Count").firstMatch
        tap(quantity.buttons["plus"], in: app)
        XCTAssertEqual(app.textFields["Count"].value as? String, "46")
        capture(app, "large-inventory-count-updated")
    }

    @MainActor
    func testDarkScreens() {
        let app = demo()
        capture(app, "dark-today")
        selectFamilyTab("Chores", in: app)
        capture(app, "dark-chores")
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Vacuum living room")).firstMatch, in: app)
        capture(app, "dark-chore-detail")
        app.buttons["Edit Chore"].tap()
        tap(app.buttons["Edit repeat rule"], in: app)
        capture(app, "dark-repeat")
        app.buttons["Done"].tap()
        app.buttons["Cancel"].tap()
        selectFamilyTab("Today", in: app)
        app.buttons["home-weather"].tap()
        capture(app, "dark-weather")
        back(app)
        app.buttons["Open profile"].tap()
        capture(app, "dark-settings")
        tap(app.buttons["Chore Reminders"], in: app)
        capture(app, "dark-notifications")
        back(app)
        tap(app.buttons["Family message"], in: app)
        capture(app, "dark-family-message")
        let field = app.textFields["Family message"].exists ? app.textFields["Family message"] : app.textViews["Family message"]
        field.tap()
        field.typeText("Dinner is ready.")
        app.buttons["Done"].firstMatch.tap()
        tap(app.buttons["Send to Family"], in: app)
        XCTAssertTrue(app.staticTexts["No family iPhones have notifications enabled."].waitForExistence(timeout: 5))
        capture(app, "dark-message-result")
        back(app)
        back(app)
        app.buttons["Done"].tap()
        selectFamilyTab("Household", in: app)
        app.buttons["Inventory"].tap()
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch.tap()
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Washing tablets")).firstMatch.tap()
        capture(app, "dark-inventory-count")
    }

    @MainActor
    private func demo(largeText: Bool = false) -> XCUIApplication {
        continueAfterFailure = false
        let app = XCUIApplication()
        if largeText { app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"] }
        app.launch()
        tap(app.buttons["Try Demo"], in: app)
        return app
    }

    @MainActor
    private func settings() -> XCUIApplication {
        let app = demo()
        app.buttons["Open profile"].tap()
        return app
    }

    @MainActor
    private func frequency(_ name: String, in app: XCUIApplication) {
        let picker = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Frequency,")).firstMatch
        tap(picker, in: app, towardTop: true)
        app.buttons[name].tap()
    }

    @MainActor
    private func dismissPopover(_ app: XCUIApplication) {
        if let cancel = app.buttons.matching(identifier: "Cancel").allElementsBoundByIndex.first(where: \.isHittable) { cancel.tap() }
        else { app.coordinate(withNormalizedOffset: CGVector(dx: 0.15, dy: 0.2)).tap() }
    }

    @MainActor
    private func back(_ app: XCUIApplication) { app.navigationBars.buttons["BackButton"].firstMatch.tap() }

    @MainActor
    private func tap(_ element: XCUIElement, in app: XCUIApplication, towardTop: Bool = false, shouldTap: Bool = true) {
        _ = element.waitForExistence(timeout: 5)
        for _ in 0..<18 {
            if element.exists && element.isHittable { break }
            app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.45 : 0.75))
                .press(forDuration: 0.05, thenDragTo: app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.75 : 0.45)))
        }
        XCTAssertTrue(element.exists)
        if shouldTap { element.tap() }
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
