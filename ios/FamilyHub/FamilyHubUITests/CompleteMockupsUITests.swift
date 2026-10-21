import XCTest

final class CompleteMockupsUITests: XCTestCase {
    @MainActor
    func testRepeatRuleDraftReturnsToTheChoreFormAndSaves() {
        continueAfterFailure = false
        let app = demo()
        selectFamilyTab("Chores", in: app)
        app.buttons["New Chore"].tap()
        let name = app.textFields["Name"]
        name.tap()
        name.typeText("Clean the windows\n")
        let repeatRule = app.buttons["Edit repeat rule"]
        reveal(repeatRule, in: app)
        XCTAssertTrue(repeatRule.exists)
        repeatRule.tap()
        XCTAssertTrue(app.navigationBars["Repeat"].waitForExistence(timeout: 5))
        app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Frequency,")).firstMatch.tap()
        app.buttons["Weekly"].tap()
        app.buttons["Mon"].tap()
        capture(app, "chore-repeat-draft")
        app.buttons["Done"].tap()
        reveal(name, in: app, towardTop: true)
        XCTAssertEqual(name.value as? String, "Clean the windows")
        app.buttons["Save"].tap()
        let chore = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Clean the windows")).firstMatch
        reveal(chore, in: app)
        XCTAssertTrue(chore.waitForExistence(timeout: 5))
        chore.tap()
        XCTAssertTrue(app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "Weekly")).firstMatch.exists)
        capture(app, "chore-detail-saved")
    }

    @MainActor
    func testMessageResultReturnsToTheDraft() {
        continueAfterFailure = false
        let app = demo()
        app.buttons["Open profile"].tap()
        reveal(app.buttons["Family message"], in: app)
        app.buttons["Family message"].tap()
        let field = app.textFields["Family message"].exists ? app.textFields["Family message"] : app.textViews["Family message"]
        field.tap()
        field.typeText("Dinner is ready.")
        if let done = app.buttons.matching(identifier: "Done").allElementsBoundByIndex.first(where: \.isHittable) { done.tap() }
        let send = app.buttons["Send to Family"]
        reveal(send, in: app)
        send.tap()
        XCTAssertTrue(app.staticTexts["No family iPhones have notifications enabled."].waitForExistence(timeout: 5))
        let back = app.buttons["Back to message"]
        reveal(back, in: app)
        XCTAssertTrue(back.exists)
        capture(app, "family-message-result")
        let help = app.buttons["Notification help"]
        reveal(help, in: app)
        XCTAssertGreaterThanOrEqual(help.frame.width, 280)
        help.tap()
        XCTAssertTrue(app.navigationBars["Notifications"].waitForExistence(timeout: 5))
        app.navigationBars.buttons["BackButton"].firstMatch.tap()
        back.tap()
        reveal(field, in: app, towardTop: true)
        XCTAssertEqual(field.value as? String, "Dinner is ready.")
        capture(app, "family-message-draft")
    }

    @MainActor
    func testTokenCreationShowsTheOneTimeResultAndCopies() {
        continueAfterFailure = false
        let app = demo()
        app.buttons["Open profile"].tap()
        reveal(app.buttons["API Tokens"], in: app)
        app.buttons["API Tokens"].tap()
        app.buttons["New Token"].tap()
        let name = app.textFields["Token name"]
        name.tap()
        name.typeText("Kitchen display\n")
        app.buttons["Create token"].tap()
        XCTAssertTrue(app.staticTexts["Save your token"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["demo-token-preview-only"].exists)
        let copy = app.buttons["Copy Token"]
        reveal(copy, in: app)
        XCTAssertGreaterThanOrEqual(copy.frame.width, 280)
        copy.tap()
        XCTAssertTrue(app.buttons["Copied"].waitForExistence(timeout: 2))
        capture(app, "token-created-copied")
        app.buttons["Done"].tap()
        XCTAssertTrue(app.staticTexts["demo-token-preview-only"].waitForNonExistence(timeout: 5))
        capture(app, "token-list-populated")
    }

    @MainActor
    func testRoleSelectionNeedsSaveAndProtectsYourOwnRole() {
        continueAfterFailure = false
        let app = demo()
        app.buttons["Open profile"].tap()
        reveal(app.buttons["Family Members"], in: app)
        app.buttons["Family Members"].tap()
        let member = app.buttons["member-demo-user-2"]
        reveal(member, in: app)
        member.tap()
        let admin = app.buttons["Choose admin role"]
        XCTAssertTrue(admin.waitForExistence(timeout: 5))
        admin.tap()
        app.navigationBars.buttons["BackButton"].firstMatch.tap()
        XCTAssertEqual(member.value as? String, "Member")
        member.tap()
        XCTAssertFalse(admin.isSelected)
        admin.tap()
        app.buttons["Save"].tap()
        app.buttons["Make admin"].tap()
        XCTAssertTrue(member.waitForExistence(timeout: 5))
        XCTAssertEqual(member.value as? String, "Admin")
        capture(app, "members-role-saved")
        app.buttons["member-demo-user-1"].tap()
        XCTAssertTrue(app.staticTexts["You cannot change your own role here."].waitForExistence(timeout: 5))
        XCTAssertFalse(app.buttons["Choose member role"].isEnabled)
        XCTAssertFalse(app.buttons["Save"].isEnabled)
        capture(app, "member-own-role-protected")
    }

    @MainActor
    func testCategoryRowSupportsRenameAndConfirmedDelete() {
        continueAfterFailure = false
        let app = demo()
        app.buttons["Open profile"].tap()
        reveal(app.buttons["Categories"], in: app)
        app.buttons["Categories"].tap()
        app.buttons["Cleaning"].tap()
        let name = app.textFields["Category name"]
        XCTAssertTrue(name.waitForExistence(timeout: 5))
        name.tap()
        name.typeText(String(repeating: XCUIKeyboardKey.delete.rawValue, count: 8) + "Housework\n")
        app.buttons["Save"].tap()
        let renamed = app.buttons["Housework"]
        XCTAssertTrue(renamed.waitForExistence(timeout: 5))
        renamed.tap()
        app.buttons["Delete category"].tap()
        XCTAssertTrue(app.staticTexts["Delete Housework?"].waitForExistence(timeout: 5))
        if let cancel = app.buttons.matching(identifier: "Cancel").allElementsBoundByIndex.first(where: \.isHittable) { cancel.tap() }
        else { app.coordinate(withNormalizedOffset: CGVector(dx: 0.15, dy: 0.2)).tap() }
        XCTAssertTrue(app.textFields["Category name"].exists)
        app.buttons["Delete category"].tap()
        app.buttons.matching(identifier: "Delete category").allElementsBoundByIndex.last(where: \.isHittable)!.tap()
        XCTAssertTrue(renamed.waitForNonExistence(timeout: 5))
        XCTAssertTrue(app.buttons["Garden"].exists)
    }

    @MainActor
    private func demo(largeText: Bool = false) -> XCUIApplication {
        let app = XCUIApplication()
        if largeText { app.launchArguments = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"] }
        app.launch()
        let button = app.buttons["Try Demo"]
        XCTAssertTrue(button.waitForExistence(timeout: 10))
        reveal(button, in: app)
        button.tap()
        return app
    }

    @MainActor
    private func reveal(_ element: XCUIElement, in app: XCUIApplication, towardTop: Bool = false) {
        _ = element.waitForExistence(timeout: 3)
        for _ in 0..<16 {
            if element.exists && element.isHittable { return }
            let start = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.45 : 0.75))
            let end = app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: towardTop ? 0.75 : 0.45))
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
