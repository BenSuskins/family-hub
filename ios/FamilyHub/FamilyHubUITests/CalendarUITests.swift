import XCTest
import UIKit

final class CalendarUITests: XCTestCase {
    @MainActor
    func testWeekEventTitlesWrapInsideTheTimeBlock() {
        let app = launchCalendar(layoutFixtures: true)
        app.buttons["calendar-view-menu"].tap()
        app.buttons["calendar-mode-Week"].tap()
        let timeline = app.otherElements["calendar-timeline"]
        XCTAssertTrue(timeline.waitForExistence(timeout: 5))
        let event = app.buttons["calendar-event-fixture-long-title"]
        XCTAssertTrue(event.waitForExistence(timeout: 5))
        for _ in 0..<2 {
            app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.8))
                .press(forDuration: 0.1,
                       thenDragTo: app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.55)),
                       withVelocity: .slow, thenHoldForDuration: 0.1)
        }
        capture(app, "calendar-week-wrapped-title")
        XCTAssertTrue(event.isHittable)
        let titleHeight = titleInkHeight(in: event.screenshot().image)
        XCTAssertGreaterThan(titleHeight, UIFont.preferredFont(forTextStyle: .caption1).lineHeight * 1.5)
    }

    @MainActor
    func testCompactWeekAllDayListContainsEventsChoresAndMeals() {
        let app = launchCalendar()
        app.buttons["calendar-view-menu"].tap()
        app.buttons["calendar-mode-Week"].tap()

        let allDayItems = app.buttons["calendar-all-day-count-2026-03-15"]
        XCTAssertTrue(allDayItems.waitForExistence(timeout: 5))
        XCTAssertTrue(allDayItems.label.contains("5 all-day items"))
        capture(app, "calendar-week-compact-all-day")
        allDayItems.tap()

        XCTAssertTrue(app.buttons["calendar-all-day-done"].waitForExistence(timeout: 5))
        capture(app, "calendar-week-all-day-expanded")
        for identifier in [
            "event-fixture-trip",
            "chore-fixture-dishes",
            "meal-2026-03-15-breakfast",
            "meal-2026-03-15-lunch",
        ] {
            XCTAssertTrue(app.descendants(matching: .any)["calendar-item-\(identifier)"].exists)
        }
        app.descendants(matching: .any)["calendar-item-meal-2026-03-15-lunch"].swipeUp()
        XCTAssertTrue(app.descendants(matching: .any)["calendar-item-meal-2026-03-15-dinner"].exists)

        app.buttons["calendar-all-day-done"].tap()
        XCTAssertTrue(app.otherElements["calendar-timeline"].waitForExistence(timeout: 5))
        XCTAssertEqual(app.buttons["calendar-view-menu"].value as? String, "Week")
        XCTAssertTrue(app.buttons["calendar-all-day-count-2026-03-15"].exists)
    }

    @MainActor
    func testMonthZoomKeepsTheNextHeadingBelowTheLastWeek() {
        let app = launchCalendar()
        let months = app.otherElements["calendar-months"]
        XCTAssertTrue(months.waitForExistence(timeout: 5))
        for _ in 0..<2 {
            let zoomTarget = app.staticTexts["March"]
            XCTAssertTrue(zoomTarget.waitForExistence(timeout: 5))
            zoomTarget.pinch(withScale: 0.6, velocity: -1)
            capture(app, "calendar-month-minimum-zoom")
            let lastDay = app.buttons["calendar-date-2026-03-31"]
            let april = app.staticTexts["April"]
            XCTAssertTrue(lastDay.waitForExistence(timeout: 5))
            XCTAssertTrue(april.waitForExistence(timeout: 5))
            XCTAssertLessThanOrEqual(lastDay.frame.maxY, april.frame.minY)
        }
    }

    private func titleInkHeight(in image: UIImage) -> CGFloat {
        guard let source = image.cgImage else { return 0 }
        let width = source.width
        let height = source.height
        var pixels = [UInt8](repeating: 0, count: width * height * 4)
        let rows = pixels.withUnsafeMutableBytes { buffer -> [Int] in
            guard let context = CGContext(data: buffer.baseAddress, width: width, height: height,
                                          bitsPerComponent: 8, bytesPerRow: width * 4,
                                          space: CGColorSpaceCreateDeviceRGB(),
                                          bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { return [] }
            context.draw(source, in: CGRect(x: 0, y: 0, width: width, height: height))
            let background = (height / 2 * width + width - 1) * 4
            let usesLightText = Int(buffer[background]) + Int(buffer[background + 1]) + Int(buffer[background + 2]) < 384
            return (0..<height).filter { row in
                (Int(7 * image.scale)..<width).contains { column in
                    let offset = (row * width + column) * 4
                    let isText = usesLightText
                        ? buffer[offset] > 180 && buffer[offset + 1] > 180 && buffer[offset + 2] > 180
                        : buffer[offset] < 100 && buffer[offset + 1] < 100 && buffer[offset + 2] < 100
                    return isText && buffer[offset + 3] > 0
                }
            }
        }
        guard let first = rows.first, let last = rows.last else { return 0 }
        return CGFloat(last - first + 1) / image.scale
    }

    @MainActor
    func testMonthScrollPerformance() {
        let app = launchCalendar()
        let months = app.otherElements["calendar-months"]
        XCTAssertTrue(months.waitForExistence(timeout: 5))
        let options = XCTMeasureOptions()
        options.iterationCount = 3
        measure(metrics: [XCTOSSignpostMetric.scrollingAndDecelerationMetric, XCTCPUMetric(application: app)], options: options) {
            months.swipeUp(velocity: .slow)
            months.swipeDown(velocity: .slow)
        }
    }

    @MainActor
    func testYearScrollPerformance() {
        let app = launchCalendar()
        app.buttons["Show year"].tap()
        let years = app.scrollViews["calendar-years"]
        XCTAssertTrue(years.waitForExistence(timeout: 5))
        let options = XCTMeasureOptions()
        options.iterationCount = 3
        measure(metrics: [XCTOSSignpostMetric.scrollingAndDecelerationMetric, XCTCPUMetric(application: app)], options: options) {
            years.swipeUp(velocity: .slow)
            years.swipeDown(velocity: .slow)
        }
    }

    @MainActor
    func testTodayIsInTopBarAndTheMenuContainsOnlySupportedViews() {
        let app = launchCalendar()
        let today = app.buttons["calendar-today"]
        XCTAssertTrue(today.waitForExistence(timeout: 5))
        XCTAssertLessThanOrEqual(today.frame.maxY, app.navigationBars.firstMatch.frame.maxY)
        app.buttons["calendar-view-menu"].tap()
        XCTAssertFalse(app.buttons["Date"].exists)
        XCTAssertFalse(app.buttons["calendar-month-display-Compact"].exists)
        XCTAssertFalse(app.buttons["calendar-month-display-Stacked"].exists)
        XCTAssertFalse(app.buttons["calendar-month-display-List"].exists)
        for mode in ["Year", "Month", "Week", "Day"] {
            XCTAssertTrue(app.buttons["calendar-mode-" + mode].exists)
        }
        XCTAssertFalse(app.buttons["calendar-mode-Multi Day"].exists)
        XCTAssertFalse(app.buttons["calendar-mode-List"].exists)
    }

    @MainActor
    func testMonthAndYearScrollingReturnToToday() {
        let app = launchCalendar()
        let months = app.otherElements["calendar-months"]
        XCTAssertTrue(months.waitForExistence(timeout: 5))
        months.swipeUp()
        months.swipeUp()
        app.buttons["calendar-today"].tap()
        let date = app.buttons["calendar-date-2026-03-15"]
        XCTAssertTrue(date.waitForExistence(timeout: 5))
        XCTAssertTrue(date.isHittable)
        app.buttons["Show year"].tap()
        let years = app.scrollViews["calendar-years"]
        XCTAssertTrue(years.waitForExistence(timeout: 5))
        years.swipeUp()
        years.swipeUp()
        app.buttons["calendar-today"].tap()
        let march = app.buttons["March 2026"]
        XCTAssertTrue(march.waitForExistence(timeout: 5))
        XCTAssertTrue(march.isHittable)
        march.tap()
        XCTAssertTrue(date.waitForExistence(timeout: 5))
        XCTAssertTrue(date.isHittable)
    }

    @MainActor
    func testMonthOpensDayAndOverlappingEventsOpenDetails() {
        let app = launchCalendar()
        let date = app.buttons["calendar-date-2026-03-15"]
        XCTAssertTrue(date.waitForExistence(timeout: 5))
        if app.tabBars.firstMatch.exists {
            XCTAssertLessThanOrEqual(app.buttons["calendar-today"].frame.maxY, app.tabBars.firstMatch.frame.minY)
        }
        capture(app, "calendar-month-details")
        date.tap()
        XCTAssertTrue(app.otherElements["calendar-timeline"].waitForExistence(timeout: 5))
        let school = app.buttons["calendar-event-fixture-school"]
        XCTAssertTrue(school.waitForExistence(timeout: 5))
        XCTAssertTrue(school.isHittable)
        let dentist = app.buttons["calendar-event-fixture-dentist"]
        XCTAssertTrue(dentist.exists)
        XCTAssertLessThan(school.frame.minX, dentist.frame.minX)
        XCTAssertLessThanOrEqual(school.frame.maxX, dentist.frame.minX + 1)
        capture(app, "calendar-day-overlap")
        school.tap()
        XCTAssertTrue(app.navigationBars["Event"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["School run"].exists)
        app.buttons["BackButton"].firstMatch.tap()
        app.buttons["Show month"].tap()
        XCTAssertTrue(date.waitForExistence(timeout: 5))
        app.buttons["Show year"].tap()
        capture(app, "calendar-year")
        XCTAssertTrue(app.buttons["March 2026"].waitForExistence(timeout: 5))
        app.buttons["March 2026"].tap()
        XCTAssertTrue(date.waitForExistence(timeout: 5))
        if UIDevice.current.userInterfaceIdiom == .pad {
            XCUIDevice.shared.orientation = .landscapeLeft
            app.buttons["calendar-today"].tap()
            XCTAssertTrue(date.waitForExistence(timeout: 5))
            XCTAssertTrue(date.isHittable)
            capture(app, "calendar-ipad-landscape")
        }
    }


    @MainActor
    func testViewMenusZoomSwipeAndToday() {
        let app = launchCalendar()
        let months = app.otherElements["calendar-months"]
        XCTAssertTrue(months.waitForExistence(timeout: 5))
        months.pinch(withScale: 1.3, velocity: 1)
        capture(app, "calendar-month-zoom")
        app.buttons["calendar-date-2026-03-15"].tap()
        let timeline = app.otherElements["calendar-timeline"]
        XCTAssertTrue(timeline.waitForExistence(timeout: 5))
        timeline.swipeLeft()
        XCTAssertTrue(app.navigationBars[dayTitle(16)].waitForExistence(timeout: 5))
        app.buttons["calendar-today"].tap()
        XCTAssertTrue(app.navigationBars[dayTitle(15)].waitForExistence(timeout: 5))
        for mode in ["Week", "Year", "Month", "Day"] {
            app.buttons["calendar-view-menu"].tap()
            app.buttons["calendar-mode-" + mode].tap()
            XCTAssertTrue(app.buttons["calendar-today"].isHittable)
            XCTAssertLessThanOrEqual(app.buttons["calendar-today"].frame.maxX, app.windows.firstMatch.frame.maxX)
            capture(app, "calendar-" + mode)
        }
        XCTAssertTrue(app.otherElements["calendar-timeline"].waitForExistence(timeout: 5))
    }

    @MainActor
    func testCalendarWithAccessibilityText() {
        let app = launchCalendar(largeText: true)
        let date = app.descendants(matching: .any).matching(identifier: "calendar-date-picker").firstMatch
        XCTAssertTrue(date.waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["School run"].exists)
        capture(app, "calendar-accessibility")
        app.buttons["calendar-view-menu"].tap()
        app.buttons["calendar-mode-Day"].tap()
        XCTAssertTrue(app.staticTexts["School run"].exists)
    }

    @MainActor
    private func launchCalendar(largeText: Bool = false, layoutFixtures: Bool = false) -> XCUIApplication {
        continueAfterFailure = false
        if UIDevice.current.userInterfaceIdiom == .pad { XCUIDevice.shared.orientation = .portrait }
        let app = XCUIApplication()
        app.launchArguments = ["-calendar-fixtures"]
        if layoutFixtures { app.launchArguments.append("-calendar-layout-fixtures") }
        if largeText {
            app.launchArguments += ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        }
        app.launch()
        let demo = app.buttons["Try Demo"]
        for _ in 0..<6 where !demo.exists || !demo.isHittable { app.swipeUp() }
        XCTAssertTrue(demo.waitForExistence(timeout: 5))
        demo.tap()
        let tab = app.tabBars.buttons["Calendar"]
        if tab.exists && tab.isHittable { tab.tap() }
        else if let button = app.buttons.matching(identifier: "Calendar").allElementsBoundByIndex.first(where: { $0.isHittable }) {
            button.tap()
        } else {
            let sidebar = app.buttons["ToggleSidebar"]
            let calendar = app.collectionViews.staticTexts["Calendar"]
            if !calendar.isHittable && sidebar.exists { sidebar.tap() }
            calendar.tap()
        }
        return app
    }

    private func dayTitle(_ day: Int) -> String {
        Calendar.current.date(from: DateComponents(year: 2026, month: 3, day: day))!
            .formatted(.dateTime.month(.wide).day())
    }

    @MainActor
    private func capture(_ app: XCUIApplication, _ name: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }
}
