import XCTest

@MainActor
func selectFamilyTab(_ name: String, in app: XCUIApplication) {
    let title = name == "Tasks" ? "Chores" : name
    let native = app.tabBars.buttons[title]
    if native.exists && native.isHittable { native.tap(); return }
    if let button = app.buttons.matching(NSPredicate(format: "label == %@", title)).allElementsBoundByIndex.first(where: { $0.isHittable }) {
        button.tap(); return
    }
    let sidebar = app.buttons["ToggleSidebar"]
    if sidebar.exists && sidebar.isHittable { sidebar.tap() }
    let accessibleItems = app.descendants(matching: .any).matching(NSPredicate(format: "label == %@", title))
    if let item = accessibleItems.allElementsBoundByIndex.first(where: { $0.isHittable }) {
        item.tap()
        return
    }
    app.collectionViews.staticTexts[title].tap()
}

@MainActor
func openShoppingListIfNeeded(in app: XCUIApplication) {
    if !app.textFields["Item name"].exists {
        let link = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Shopping list")).firstMatch
        XCTAssertTrue(link.waitForExistence(timeout: 5))
        link.tap()
    }
}
