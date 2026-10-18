import XCTest
@testable import FamilyHub

@MainActor
final class AppNavigationTests: XCTestCase {
    func testMealWidgetOpensTheMealPlanFromAnotherFoodScreen() {
        let navigation = AppNavigation()
        navigation.foodPath = [.recipes]
        navigation.open(URL(string: "familyhub://meals")!)
        XCTAssertEqual(navigation.selectedTab, .food)
        XCTAssertEqual(navigation.foodPath, [.mealPlan])
    }

    func testTodayAndCalendarLinksPreserveOtherGroups() {
        let navigation = AppNavigation()
        navigation.foodPath = [.recipes]
        navigation.householdPath = [.shopping]
        navigation.open(URL(string: "familyhub://calendar")!)
        XCTAssertEqual(navigation.selectedTab, .calendar)
        navigation.open(URL(string: "familyhub://today")!)
        XCTAssertEqual(navigation.selectedTab, .today)
        XCTAssertEqual(navigation.foodPath, [.recipes])
        XCTAssertEqual(navigation.householdPath, [.shopping])
    }

    func testUnknownLinksKeepTheCurrentScreen() {
        let navigation = AppNavigation()
        navigation.selectedTab = .tasks
        navigation.open(URL(string: "https://meals")!)
        navigation.open(URL(string: "familyhub://unknown")!)
        XCTAssertEqual(navigation.selectedTab, .tasks)
    }
}
