import XCTest
import UIKit

final class NativeRedesignAuditUITests: XCTestCase {
    @MainActor
    func testCalendarMenuScreens() {
        let app = demo()
        selectFamilyTab("Calendar", in: app)
        tap(app.buttons["calendar-view-menu"], in: app)
        capture(app, "calendar-view-menu")
    }

    @MainActor
    func testChoreScheduleOptionScreens() {
        let app = demo()
        selectFamilyTab("Chores", in: app)
        tap(app.buttons["New Chore"], in: app)
        app.swipeUp()
        app.swipeUp()
        let repeats = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Repeats")).firstMatch
        tap(repeats, in: app)
        capture(app, "tasks-repeat-menu")
        tap(app.buttons["Monthly"], in: app)
        app.swipeUp()
        app.swipeUp()
        capture(app, "tasks-repeat-monthly")
        tap(repeats, in: app)
        tap(app.buttons["Custom"], in: app)
        app.swipeUp()
        app.swipeUp()
        capture(app, "tasks-repeat-custom")
        reveal(app.switches["Ends On Date"], in: app)
        app.switches["Ends On Date"].coordinate(withNormalizedOffset: CGVector(dx: 0.93, dy: 0.5)).tap()
        XCTAssertEqual(app.switches["Ends On Date"].value as? String, "1")
        capture(app, "tasks-repeat-end-date")
        reveal(app.switches["Limit Occurrences"], in: app)
        app.switches["Limit Occurrences"].coordinate(withNormalizedOffset: CGVector(dx: 0.93, dy: 0.5)).tap()
        XCTAssertEqual(app.switches["Limit Occurrences"].value as? String, "1")
        capture(app, "tasks-repeat-count")
    }

    @MainActor
    func testReduceTransparencyScreens() {
        let settings = XCUIApplication(bundleIdentifier: "com.apple.Preferences")
        settings.launch()
        tap(settings.cells.containing(.staticText, identifier: "Accessibility").firstMatch, in: settings)
        tap(settings.cells.containing(.staticText, identifier: "Display & Text Size").firstMatch, in: settings)
        let toggle = settings.switches.matching(NSPredicate(format: "label BEGINSWITH %@", "Reduce Transparency")).firstMatch
        reveal(toggle, in: settings)
        let wasEnabled = toggle.value as? String == "1"
        if !wasEnabled { toggle.coordinate(withNormalizedOffset: CGVector(dx: 0.93, dy: 0.5)).tap() }
        XCTAssertEqual(toggle.value as? String, "1")
        let enabled = NSPredicate { _, _ in UIAccessibility.isReduceTransparencyEnabled }
        XCTAssertEqual(XCTWaiter.wait(for: [XCTNSPredicateExpectation(predicate: enabled, object: nil)], timeout: 5), .completed)
        testDisplayPreferenceScreens()
        settings.activate()
        if !wasEnabled { toggle.coordinate(withNormalizedOffset: CGVector(dx: 0.93, dy: 0.5)).tap() }
        XCUIApplication().activate()
    }

    @MainActor
    func testDisplayPreferenceScreens() {
        let preferences = XCTAttachment(string: "Reduce Motion: \(UIAccessibility.isReduceMotionEnabled)\nReduce Transparency: \(UIAccessibility.isReduceTransparencyEnabled)\nIncrease Contrast: \(UIAccessibility.isDarkerSystemColorsEnabled)")
        preferences.name = "Effective accessibility preferences"
        preferences.lifetime = .keepAlways
        add(preferences)
        let app = demo()
        capture(app, "variant-today")
        for tab in ["Calendar", "Chores", "Food", "Household"] {
            selectFamilyTab(tab, in: app)
            capture(app, "variant-" + tab.lowercased())
        }
        tap(app.buttons["Open profile"], in: app)
        capture(app, "variant-settings")
        tap(app.buttons["Chore Reminders"], in: app)
        capture(app, "variant-reminders")
        back(app)
        tap(app.buttons["Done"], in: app)
        selectFamilyTab("Chores", in: app)
        tap(app.buttons["New Chore"], in: app)
        capture(app, "variant-chore-form")
        tap(app.buttons["Cancel"], in: app)
        selectFamilyTab("Food", in: app)
        tap(app.buttons["Recipes"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch, in: app)
        capture(app, "variant-recipe")
        tap(app.buttons["Start Cook Mode"], in: app)
        capture(app, "variant-cook-ingredients")
        tap(app.buttons["Start step 1"], in: app)
        capture(app, "variant-cook-step")
    }

    @MainActor
    func testAccessScreens() {
        let app = launch()
        capture(app, "access-welcome")
        tap(app.buttons["Setup Help"], in: app)
        capture(app, "access-setup-help")
        tap(app.buttons["Set up a Family Hub server · For admins"], in: app)
        app.swipeUp()
        capture(app, "access-setup-help-bottom")
        tap(app.buttons["Done"], in: app)
        let server = app.textFields["Server URL"]
        if server.exists {
            tap(server, in: app)
            server.typeText("invalid")
            capture(app, "access-server-keyboard")
            tap(app.buttons["Connect"], in: app)
            capture(app, "access-invalid-server")
        }
    }

    @MainActor
    func testTodayScreens() {
        let app = demo()
        capture(app, "today-overview")
        tap(app.buttons["home-weather"], in: app)
        capture(app, "today-weather")
        for _ in 0..<5 { app.swipeUp() }
        capture(app, "today-weather-bottom")
        back(app)
        let event = app.buttons.containing(.staticText, identifier: "Family Dinner").firstMatch
        tap(event, in: app)
        capture(app, "today-event")
        back(app)
        tap(app.buttons["Actions for Vacuum living room"], in: app)
        capture(app, "today-chore-actions")
        dismissPopover(app)
        app.swipeUp()
        capture(app, "today-bottom")
    }

    @MainActor
    func testReminderAndBroadcastScreens() {
        let app = settings()
        tap(app.buttons["Chore Reminders"], in: app)
        capture(app, "settings-reminders")
        app.swipeUp()
        capture(app, "settings-reminders-bottom")
        back(app)
        tap(app.buttons["Family message"], in: app)
        capture(app, "settings-broadcast")
        let message = app.textFields["Family message"].exists ? app.textFields["Family message"] : app.textViews["Family message"]
        tap(message, in: app)
        message.typeText("Dinner is ready.")
        capture(app, "settings-broadcast-keyboard")
        tap(app.buttons["Send to Family"], in: app)
        capture(app, "settings-broadcast-result")
    }

    @MainActor
    func testProfileAndConfigurationScreens() {
        let app = settings()
        capture(app, "settings-profile")
        tap(app.buttons["Change Photo"], in: app)
        capture(app, "settings-photo-picker")
        if app.buttons["Cancel"].exists { app.buttons["Cancel"].tap() }
        else { app.swipeDown() }
    }

    @MainActor
    func testServerConfigurationScreens() {
        let app = settings()
        let server = app.cells.containing(.staticText, identifier: "Server URL").firstMatch
        tap(server, in: app)
        capture(app, "settings-server-configuration")
        let address = app.textFields.firstMatch
        if address.exists {
            tap(address, in: app)
            address.typeText("invalid")
            tap(app.buttons["Connect"], in: app)
            capture(app, "settings-server-validation")
        }
    }

    @MainActor
    func testCategoryScreens() {
        let app = settings()
        tap(app.buttons["Categories"], in: app)
        capture(app, "settings-categories")
        tap(app.buttons["New Category"], in: app)
        capture(app, "settings-category-create")
        tap(app.buttons["Cancel"], in: app)
        let category = app.staticTexts["Cleaning"].firstMatch
        if category.exists {
            category.swipeLeft()
            capture(app, "settings-category-actions")
            if app.buttons["Rename"].exists {
                app.buttons["Rename"].tap()
                capture(app, "settings-category-rename")
                app.buttons["Cancel"].tap()
            }
        }
    }

    @MainActor
    func testMemberAndHomeScreens() {
        let app = settings()
        tap(app.buttons["Family Members"], in: app)
        capture(app, "settings-family-members")
        let member = app.buttons["member-demo-user-2"]
        if member.exists {
            tap(member, in: app)
            capture(app, "settings-member-role")
            back(app)
        }
        back(app)
        tap(app.buttons["Home location"], in: app)
        capture(app, "settings-home-info")
        app.swipeUp()
        capture(app, "settings-home-info-bottom")
        back(app)
        tap(app.buttons["Family Name"], in: app)
        capture(app, "settings-family-name")
        tap(app.textFields.firstMatch, in: app)
        capture(app, "settings-family-name-keyboard")
    }

    @MainActor
    func testChoreEditScreens() {
        let app = demo()
        selectFamilyTab("Chores", in: app)
        capture(app, "tasks-list")
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Vacuum living room")).firstMatch, in: app)
        capture(app, "tasks-detail")
        tap(app.buttons["Edit Chore"], in: app)
        capture(app, "tasks-edit")
        app.swipeUp()
        capture(app, "tasks-edit-schedule")
        app.swipeUp()
        capture(app, "tasks-edit-repeat")
        app.swipeUp()
        capture(app, "tasks-edit-end")
        tap(app.buttons.matching(identifier: "Delete Chore").allElementsBoundByIndex.first(where: { $0.isHittable }) ?? app.buttons["Delete Chore"].firstMatch, in: app)
        capture(app, "tasks-delete-confirmation")
    }

    @MainActor
    func testChoreCreateScreens() {
        let app = demo()
        selectFamilyTab("Chores", in: app)
        tap(app.buttons["New Chore"], in: app)
        capture(app, "tasks-create")
        tap(app.textFields["Name"], in: app)
        app.textFields["Name"].typeText("Water plants")
        capture(app, "tasks-create-keyboard")
        app.swipeUp()
        app.swipeUp()
        capture(app, "tasks-create-schedule")
    }

    @MainActor
    func testMealScreens() {
        let app = demo()
        selectFamilyTab("Food", in: app)
        capture(app, "food-overview")
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Meal plan")).firstMatch, in: app)
        capture(app, "food-week")
        app.swipeUp()
        capture(app, "food-week-bottom")
        back(app)
        let emptyMeal = app.buttons.matching(NSPredicate(format: "label == %@ OR label == %@", "Add lunch", "Add dinner")).firstMatch
        tap(emptyMeal, in: app)
        capture(app, "food-add-meal")
        tap(app.buttons["Cancel"], in: app)
        let breakfast = app.buttons.matching(NSPredicate(format: "label == %@ OR label BEGINSWITH %@", "Add breakfast", "Edit breakfast:")).firstMatch
        tap(breakfast, in: app)
        capture(app, "food-edit-meal-existing")
        let name = app.textFields["Meal name"]
        tap(name, in: app)
        if let value = name.value as? String, value != "What is for breakfast?" { name.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: value.count)) }
        name.typeText("Chicken")
        capture(app, "food-meal-search")
        tap(app.buttons["Chicken Stir Fry"], in: app)
        capture(app, "food-meal-recipe-selected")
        tap(app.buttons["Save"], in: app)
        tap(app.buttons["Edit breakfast: Chicken Stir Fry"], in: app)
        capture(app, "food-edit-meal")
    }

    @MainActor
    func testRecipeEditScreens() {
        let app = recipe()
        capture(app, "food-recipe-detail")
        app.swipeUp()
        capture(app, "food-recipe-ingredients")
        app.swipeUp()
        capture(app, "food-recipe-method")
        tap(app.buttons["Recipe actions"], in: app)
        capture(app, "food-recipe-actions")
        tap(app.buttons["Edit"], in: app)
        capture(app, "food-recipe-edit")
        app.swipeUp()
        capture(app, "food-recipe-edit-details")
        tap(app.buttons["Edit recipe ingredients"], in: app)
        capture(app, "food-recipe-edit-ingredients")
        tap(app.buttons["Done"], in: app)
        tap(app.buttons["Edit recipe method"], in: app)
        capture(app, "food-recipe-edit-steps")
    }

    @MainActor
    func testRecipeSearchAndCookScreens() {
        let app = recipe()
        tap(app.buttons["Start Cook Mode"], in: app)
        capture(app, "food-cook-ingredients")
        tap(app.buttons["Start step 1"], in: app)
        capture(app, "food-cook-step")
        for _ in 0..<15 {
            if app.buttons["Finish cooking"].exists { break }
            app.buttons["Next step"].tap()
        }
        capture(app, "food-cook-final-step")
        tap(app.buttons["Finish cooking"], in: app)
        back(app)
        capture(app, "food-recipes")
        app.swipeDown()
        let search = app.searchFields.firstMatch
        if search.exists {
            search.tap()
            search.typeText("No matching recipe")
            capture(app, "food-recipes-no-results")
        }
    }

    @MainActor
    func testRecipeCreateScreens() {
        let app = demo()
        selectFamilyTab("Food", in: app)
        tap(app.buttons["Recipes"], in: app)
        tap(app.buttons["New Recipe"], in: app)
        capture(app, "food-recipe-create")
        app.swipeUp()
        capture(app, "food-recipe-create-details")
        tap(app.buttons["Edit recipe ingredients"], in: app)
        capture(app, "food-recipe-create-ingredients")
        tap(app.buttons["Done"], in: app)
        tap(app.buttons["Edit recipe method"], in: app)
        capture(app, "food-recipe-create-steps")
    }

    @MainActor
    func testTokenScreens() {
        let app = settings()
        tap(app.buttons["API Tokens"], in: app)
        capture(app, "settings-tokens")
        tap(app.buttons["New Token"], in: app)
        capture(app, "settings-token-create")
        let name = app.textFields["Token name"]
        tap(name, in: app)
        name.typeText("Kitchen display")
        tap(app.buttons["Create token"], in: app)
        capture(app, "settings-token-created")
        tap(app.buttons["Copy Token"], in: app)
        capture(app, "settings-token-copied")
        tap(app.buttons["Done"], in: app)
        capture(app, "settings-tokens-populated")
    }

    @MainActor
    func testShoppingScreens() {
        let app = demo()
        selectFamilyTab("Household", in: app)
        capture(app, "household-overview")
        openShoppingListIfNeeded(in: app)
        capture(app, "household-shopping")
        tap(app.buttons["Actions for Milk"], in: app)
        capture(app, "household-shopping-actions")
        tap(app.buttons["Rename"], in: app)
        capture(app, "household-shopping-rename")
        tap(app.buttons["Cancel"], in: app)
        tap(app.buttons["Purchase Milk"], in: app)
        capture(app, "household-shopping-pending")
        tap(app.buttons["Undo"], in: app)
        capture(app, "household-shopping-undone")
        tap(app.buttons["Purchase Milk"], in: app)
        XCTAssertTrue(app.buttons["Restore Milk"].waitForNonExistence(timeout: 8))
        capture(app, "household-shopping-hidden")
    }

    @MainActor
    func testInventoryScreens() {
        let app = demo()
        selectFamilyTab("Household", in: app)
        tap(app.buttons["Inventory"], in: app)
        capture(app, "household-inventory")
        let area = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch
        tap(area, in: app)
        capture(app, "household-area")
        let item = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Fabric softener")).firstMatch
        tap(item, in: app)
        capture(app, "household-item-edit-level")
        tap(app.buttons["Delete Item"], in: app)
        capture(app, "household-item-delete-confirmation")
        dismissPopover(app)
    }

    @MainActor
    func testItemCreateScreens() {
        let app = demo()
        selectFamilyTab("Household", in: app)
        tap(app.buttons["Inventory"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch, in: app)
        tap(app.buttons["New Item"], in: app)
        capture(app, "household-item-create-count")
        app.swipeUp()
        capture(app, "household-item-create-stock")
    }

    @MainActor
    func testCountItemAndEmptyAreaScreens() {
        let app = demo()
        selectFamilyTab("Household", in: app)
        tap(app.buttons["Inventory"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch, in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Washing tablets")).firstMatch, in: app)
        capture(app, "household-item-edit-count")
        tap(app.navigationBars.buttons["Cancel"], in: app)
        back(app)
        tap(app.buttons["New Area"], in: app)
        let name = app.textFields["Name"]
        name.typeText("Audit empty area")
        tap(app.buttons["Save"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Audit empty area")).firstMatch, in: app)
        capture(app, "household-area-empty")
    }

    @MainActor
    func testAreaEditScreens() {
        let app = demo()
        selectFamilyTab("Household", in: app)
        tap(app.buttons["Inventory"], in: app)
        let area = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Laundry cupboard")).firstMatch
        reveal(area, in: app)
        area.press(forDuration: 1)
        capture(app, "household-area-actions")
        tap(app.buttons["Edit Area"], in: app)
        capture(app, "household-area-edit")
        tap(app.buttons["Cancel"], in: app)
        tap(app.buttons["New Area"], in: app)
        capture(app, "household-area-create")
    }

    @MainActor
    private func launch() -> XCUIApplication {
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()
        return app
    }

    @MainActor
    private func demo() -> XCUIApplication {
        let app = launch()
        tap(app.buttons["Try Demo"], in: app)
        XCTAssertTrue(app.buttons["Open profile"].waitForExistence(timeout: 10))
        return app
    }

    @MainActor
    private func settings() -> XCUIApplication {
        let app = demo()
        tap(app.buttons["Open profile"], in: app)
        return app
    }

    @MainActor
    private func recipe() -> XCUIApplication {
        let app = demo()
        selectFamilyTab("Food", in: app)
        tap(app.buttons["Recipes"], in: app)
        tap(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Chicken Stir Fry")).firstMatch, in: app)
        XCTAssertTrue(app.buttons["Start Cook Mode"].waitForExistence(timeout: 5))
        return app
    }

    @MainActor
    private func reveal(_ element: XCUIElement, in app: XCUIApplication, towardTop: Bool = false) {
        if !element.exists { _ = element.waitForExistence(timeout: 3) }
        for _ in 0..<12 {
            let bottom = app.tabBars.firstMatch.exists ? app.tabBars.firstMatch.frame.minY : app.windows.firstMatch.frame.maxY
            if element.exists && element.isHittable && element.frame.midY > 0 && element.frame.midY < app.windows.firstMatch.frame.maxY { return }
            let upward = element.exists ? element.frame.midY >= bottom : !towardTop
            app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: upward ? 0.7 : 0.35))
                .press(forDuration: 0.1, thenDragTo: app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: upward ? 0.35 : 0.7)))
        }
    }

    @MainActor
    private func tap(_ element: XCUIElement, in app: XCUIApplication, towardTop: Bool = false) {
        reveal(element, in: app, towardTop: towardTop)
        XCTAssertTrue(element.exists, "Missing audit element: \(element)")
        element.tap()
    }

    @MainActor
    private func dismissPopover(_ app: XCUIApplication) {
        app.coordinate(withNormalizedOffset: CGVector(dx: 0.15, dy: 0.2)).tap()
    }

    @MainActor
    private func back(_ app: XCUIApplication) {
        app.navigationBars.buttons["BackButton"].firstMatch.tap()
    }

    @MainActor
    private func capture(_ app: XCUIApplication, _ name: String) {
        // Let native sheets and popovers finish their transitions before capture.
        Thread.sleep(forTimeInterval: 0.4)
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
