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

    func testSelectingFoodAgainReturnsToFoodOverview() {
        let navigation = AppNavigation()
        navigation.selectedTab = .food
        navigation.foodPath = [.recipes]

        navigation.select(.food)

        XCTAssertEqual(navigation.selectedTab, .food)
        XCTAssertEqual(navigation.foodPath, [])
    }

    func testSwitchingAwayFromFoodPreservesItsNavigationPath() {
        let navigation = AppNavigation()
        navigation.selectedTab = .food
        navigation.foodPath = [.recipes]

        navigation.select(.calendar)
        navigation.select(.food)

        XCTAssertEqual(navigation.foodPath, [.recipes])
    }

    func testSelectingHouseholdAgainReturnsToHouseholdOverview() {
        let navigation = AppNavigation()
        navigation.selectedTab = .household
        navigation.householdPath = [.inventory]

        navigation.select(.household)

        XCTAssertEqual(navigation.selectedTab, .household)
        XCTAssertEqual(navigation.householdPath, [])
    }

    func testSwitchingAwayFromHouseholdPreservesItsNavigationPath() {
        let navigation = AppNavigation()
        navigation.selectedTab = .household
        navigation.householdPath = [.inventory]

        navigation.select(.calendar)
        navigation.select(.household)

        XCTAssertEqual(navigation.householdPath, [.inventory])
    }

    func testUnknownLinksKeepTheCurrentScreen() {
        let navigation = AppNavigation()
        navigation.selectedTab = .tasks
        navigation.open(URL(string: "https://meals")!)
        navigation.open(URL(string: "familyhub://unknown")!)
        XCTAssertEqual(navigation.selectedTab, .tasks)
    }
}
