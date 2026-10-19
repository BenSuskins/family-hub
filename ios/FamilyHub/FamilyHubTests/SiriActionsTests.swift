import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class SiriActionsTests: XCTestCase {
    func testReadMealUsesRequestedDateAndMealType() async throws {
        let client = FakeAPIClient()
        client.mealsResult = .success([
            MealPlan(date: "2026-10-27", mealType: "lunch", name: "Soup", notes: "", recipeID: nil),
            MealPlan(date: "2026-10-27", mealType: "dinner", name: "Butter chicken", notes: "", recipeID: nil),
        ])
        let service = SiriActionService(client: client)
        let day = try SiriDateRules(timezone: "Europe/London").day(DateComponents(year: 2026, month: 10, day: 27), now: Date())
        let meal = try await service.meal(type: "dinner", day: day)
        XCTAssertEqual(meal?.name, "Butter chicken")
    }
}

extension SiriActionsTests {
    func testBulkCompletionReportsPartialFailureAndLeavesFutureChores() async throws {
        let client = FakeAPIClient()
        client.homeInfo = HomeInfo(timezone: "Europe/London")
        client.choresResult = .success([
            Chore(id: "bins", name: "Bins", description: "", status: .pending, dueDate: "2026-10-19", assignedToUserID: "1"),
            Chore(id: "dishes", name: "Dishes", description: "", status: .overdue, dueDate: "2026-10-18", assignedToUserID: "1"),
            Chore(id: "future", name: "Future", description: "", status: .pending, dueDate: "2026-10-21", assignedToUserID: "1"),
        ])
        client.completeChoreErrors["dishes"] = APIError.forbidden
        let service = SiriActionService(client: client, now: { APIDate.iso.date(from: "2026-10-19T12:00:00Z")! })
        let report = try await service.completeMyChores()
        XCTAssertEqual(report.completed, ["Bins"])
        XCTAssertEqual(report.failed, ["Dishes"])
        let chores = try await client.fetchChores()
        XCTAssertEqual(chores.first { $0.id == "future" }?.status, .pending)
    }

    func testBulkCompletionDoesNotCompleteNewRecurrences() async throws {
        let client = FakeAPIClient()
        client.homeInfo = HomeInfo(timezone: "Europe/London")
        client.choresResult = .success([Chore(id: "first", name: "Dishes", description: "", status: .pending, dueDate: "2026-10-19", assignedToUserID: "1")])
        client.afterCompleteChore = { _ in
            let previous = (try? client.choresResult.get()) ?? []
            client.choresResult = .success(previous + [Chore(id: "new", name: "Dishes", description: "", status: .pending, dueDate: "2026-10-19", assignedToUserID: "1")])
        }
        let service = SiriActionService(client: client, now: { APIDate.iso.date(from: "2026-10-19T12:00:00Z")! })
        let report = try await service.completeMyChores()
        XCTAssertEqual(report.completed.count, 1)
        let chores = try await client.fetchChores()
        XCTAssertEqual(chores.first { $0.id == "new" }?.status, .pending)
    }

    func testUncertainCompletionDoesNotClaimSuccess() async throws {
        let client = FakeAPIClient()
        client.homeInfo = HomeInfo(timezone: "Europe/London")
        client.choresResult = .success([Chore(id: "first", name: "Dishes", description: "", status: .pending, dueDate: "2026-10-19", assignedToUserID: "1")])
        client.completeChoreErrors["first"] = APIError.timedOut
        let service = SiriActionService(client: client, now: { APIDate.iso.date(from: "2026-10-19T12:00:00Z")! })
        let report = try await service.completeMyChores()
        XCTAssertEqual(report.completed, [])
        XCTAssertEqual(report.uncertain, ["Dishes"])
    }

    func testCreateRejectsExistingItemInSameArea() async throws {
        let client = FakeAPIClient()
        client.inventoryResult = .success([InventoryArea(id: "kitchen", name: "Kitchen", icon: "box", tint: "blue", items: [InventoryItem(id: "soap", areaID: "kitchen", name: "Dish soap", quantity: 1, unit: "bottles", lowAt: 0)])])
        do {
            _ = try await SiriActionService(client: client).createItem(areaID: "kitchen", request: ItemRequest(name: " DISH   soap ", quantity: 2, unit: "bottles", lowAt: 1))
            XCTFail("Duplicate creation must fail.")
        } catch SiriActionError.invalid(let message) {
            XCTAssertTrue(message.contains("already exists"))
        }
    }
}

extension SiriActionsTests {
    func testAbsoluteStockTimeoutReconcilesWithSavedValue() async throws {
        let client = FakeAPIClient()
        client.inventoryResult = .success([InventoryArea(id: "kitchen", name: "Kitchen", icon: "box", tint: "blue", items: [InventoryItem(id: "tins", areaID: "kitchen", name: "Tins", quantity: 2, unit: "tins", lowAt: 1)])])
        client.changeStockError = APIError.timedOut
        let saved = try await SiriActionService(client: client).changeStock(id: "tins", change: .quantity(5))
        XCTAssertEqual(saved.quantity, 5)
    }

    func testRelativeStockTimeoutDoesNotReplayTheChange() async throws {
        let client = FakeAPIClient()
        client.inventoryResult = .success([InventoryArea(id: "kitchen", name: "Kitchen", icon: "box", tint: "blue", items: [InventoryItem(id: "tins", areaID: "kitchen", name: "Tins", quantity: 2, unit: "tins", lowAt: 1)])])
        client.changeStockError = APIError.timedOut
        do {
            _ = try await SiriActionService(client: client).changeStock(id: "tins", change: .quantityDelta(3))
            XCTFail("A relative change remains uncertain after a lost response.")
        } catch SiriActionError.uncertain { }
        let inventory = try await client.fetchInventory()
        XCTAssertEqual(inventory.flatMap(\.items).first?.quantity, 5)
    }
}

extension SiriActionsTests {
    func testMealChangesKeepDateMealTypeAndRecipeLink() async throws {
        let client = DemoAPIClient()
        let recipes = try await client.fetchRecipes(forceRefresh: true)
        let recipe = try XCTUnwrap(recipes.first)
        let service = SiriActionService(client: client)
        let day = try SiriDateRules(timezone: "Europe/London").day(DateComponents(year: 2026, month: 10, day: 27), now: Date())
        _ = try await service.saveMeal(type: "lunch", name: "Soup", day: day, recipeID: nil)
        _ = try await service.saveMeal(type: "dinner", name: "Original", day: day, recipeID: nil)
        _ = try await service.saveMeal(type: "dinner", name: recipe.title, day: day, recipeID: recipe.id)
        let dinner = try await service.meal(type: "dinner", day: day)
        let lunch = try await service.meal(type: "lunch", day: day)
        let breakfast = try await service.meal(type: "breakfast", day: day)
        XCTAssertEqual(dinner?.date, "2026-10-27")
        XCTAssertEqual(dinner?.recipeID, recipe.id)
        XCTAssertEqual(lunch?.name, "Soup")
        XCTAssertNil(breakfast)
    }

    func testNewInventoryItemKeepsCountAndPercentageModes() async throws {
        let client = DemoAPIClient()
        let service = SiriActionService(client: client)
        let area = try await client.createArea(AreaRequest(name: "Siri stock", icon: "box", tint: "blue"))
        let count = try await service.createItem(areaID: area.id, request: ItemRequest(name: "Tins", quantity: 4, unit: "tins", lowAt: 1))
        let level = try await service.createItem(areaID: area.id, request: ItemRequest(name: "Soap", trackingMode: .level, quantity: 0, level: 60, unit: "", lowAt: 20))
        let inventory = try await service.inventory()
        let saved = try XCTUnwrap(inventory.first { $0.id == area.id })
        XCTAssertEqual(saved.items.first { $0.id == count.id }?.quantity, 4)
        XCTAssertEqual(saved.items.first { $0.id == level.id }?.level, 60)
        XCTAssertEqual(saved.items.first { $0.id == level.id }?.trackingMode, .level)
    }
}
