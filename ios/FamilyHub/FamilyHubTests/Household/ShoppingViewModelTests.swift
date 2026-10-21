import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class ShoppingViewModelTests: XCTestCase {
    func testLowStockAdditionKeepsTheInventoryLinkForRestock() async {
        let client = FakeAPIClient()
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        let stock = InventoryItem(id: "softener", areaID: "laundry", name: "Fabric softener",
                                  quantity: 1, unit: "bottles", lowAt: 2)

        await model.addInventoryItem(stock)

        XCTAssertEqual(model.items.first?.name, "Fabric softener")
        XCTAssertEqual(model.items.first?.inventoryItemID, "softener")
        guard let item = model.items.first else { return XCTFail("The shopping item must be available.") }
        let purchased = await model.toggle(item)
        XCTAssertNotNil(purchased?.purchaseID)
        XCTAssertEqual(purchased?.inventoryItemID, "softener")
    }

    func testFailureKeepsShoppingReadOnly() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "soap", name: "Soap")]
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        XCTAssertTrue(model.canEdit)
        client.shoppingError = APIError.offline
        await model.load()
        XCTAssertFalse(model.canEdit)
        XCTAssertEqual(model.items.first?.name, "Soap")
        await model.add(name: "Milk")
        XCTAssertEqual(client.shoppingItems.count, 1)
    }

    func testInvalidNameLeavesTheListEditable() async {
        let client = FakeAPIClient()
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        client.addShoppingError = APIError.badRequest(serverMessage: "Name is too long.")
        await model.add(name: String(repeating: "a", count: 201))
        XCTAssertTrue(model.canEdit)
        XCTAssertTrue(model.items.isEmpty)
    }

    func testPurchaseOffersRestockOnlyForLinkedItems() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "soap", name: "Soap", inventoryItemID: "stock")]
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        let purchased = await model.toggle(model.items[0])
        XCTAssertEqual(purchased?.inventoryItemID, "stock")
        XCTAssertTrue(purchased?.purchased == true)
    }
}

@MainActor
final class HouseholdViewModelTests: XCTestCase {
    func testWeatherRefreshUsesProviderTimestamp() async throws {
        let client = FakeAPIClient()
        let data = Data(#"{"status":"ok","fetchedAt":"2026-10-18T09:46:00Z","current":{"time":"2026-10-18T09:45:00Z","temperature":14,"code":3,"rainProbability":20,"windSpeed":10},"high":17,"low":8,"hours":[],"advice":[]}"#.utf8)
        client.weatherResult = .success(try JSONDecoder().decode(HouseholdWeather.self, from: data))
        var now = APIDate.parse("2026-10-18T10:00:00Z")!
        let model = HouseholdViewModel(apiClient: client, now: { now })
        await model.loadWeather(force: true)
        XCTAssertNotNil(model.weather)
        client.weatherResult = .failure(APIError.offline)
        now = APIDate.parse("2026-10-18T10:01:00Z")!
        await model.loadWeather(force: false)
        XCTAssertNil(model.weather)
        XCTAssertEqual(model.weatherMessage, "Unavailable")
    }

}

@MainActor
final class ShoppingPurchaseFeedbackTests: XCTestCase {
    func testUndoHintUsesThePurchaseDeadlineAndClearsOnUndo() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk")]
        var now = Date(timeIntervalSince1970: 1_000)
        let model = ShoppingViewModel(apiClient: client, now: { now })
        await model.load()
        XCTAssertNil(model.remainingUndoSeconds(for: model.items[0]))
        _ = await model.toggle(model.items[0])
        XCTAssertEqual(model.remainingUndoSeconds(for: model.items[0]), 5)
        now += 2.2
        XCTAssertEqual(model.remainingUndoSeconds(for: model.items[0]), 3)
        now += 2.8
        XCTAssertEqual(model.remainingUndoSeconds(for: model.items[0]), 0)
        _ = await model.toggle(model.items[0])
        XCTAssertNil(model.remainingUndoSeconds(for: model.items[0]))
    }

    func testSlowConfirmationKeepsTheRowVisibleUntilTheServerConfirms() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk")]
        let started = expectation(description: "Purchase starts")
        var release: CheckedContinuation<Void, Never>?
        client.beforeUpdateShopping = { _, _ in
            started.fulfill()
            await withCheckedContinuation { release = $0 }
        }
        var now = Date(timeIntervalSince1970: 1_000)
        let model = ShoppingViewModel(apiClient: client, now: { now })
        await model.load()
        let purchase = Task { await model.toggle(model.items[0]) }
        await fulfillment(of: [started], timeout: 2)
        now.addTimeInterval(6)
        model.expirePurchases()
        XCTAssertEqual(model.visibleItems.count, 1)
        XCTAssertFalse(client.shoppingItems[0].purchased)
        release?.resume()
        _ = await purchase.value
        XCTAssertTrue(model.visibleItems.isEmpty)
        XCTAssertTrue(client.shoppingItems[0].purchased)
    }

    func testConflictReconcilesWithoutReplayingThePurchase() async throws {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk")]
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        _ = try await client.updateShopping(id: "milk", change: ShoppingChange(name: "Oat milk", version: 0))
        _ = await model.toggle(model.items[0])
        XCTAssertEqual(model.items[0].name, "Oat milk")
        XCTAssertEqual(client.shoppingItems[0].version, 1)
        XCTAssertFalse(client.shoppingItems[0].purchased)
        XCTAssertNotNil(model.rowErrors["milk"])
        XCTAssertTrue(model.canEdit)
    }

    func testEachItemHasItsOwnDeadlineAndUndoCancelsOnlyThatItem() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk"), ShoppingItem(id: "bread", name: "Bread")]
        var now = Date(timeIntervalSince1970: 1_000)
        let model = ShoppingViewModel(apiClient: client, now: { now })
        await model.load()
        _ = await model.toggle(model.items[0])
        now.addTimeInterval(3)
        _ = await model.toggle(model.items[1])
        now.addTimeInterval(2)
        model.expirePurchases()
        XCTAssertEqual(model.visibleItems.map(\.id), ["bread"])
        _ = await model.toggle(model.items[1])
        now.addTimeInterval(10)
        model.expirePurchases()
        XCTAssertEqual(model.visibleItems.map(\.id), ["bread"])
        XCTAssertFalse(client.shoppingItems[1].purchased)
    }

    func testFailureRestoresTheItemAndKeepsOtherRowsAvailable() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk"), ShoppingItem(id: "bread", name: "Bread")]
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        client.updateShoppingError = APIError.offline
        _ = await model.toggle(model.items[0])
        XCTAssertFalse(model.isPurchased(model.items[0]))
        XCTAssertEqual(model.visibleItems.map(\.id), ["milk", "bread"])
        XCTAssertNotNil(model.rowErrors["milk"])
        XCTAssertTrue(model.canEdit)
    }

    func testRenameValidatesUTF8AndPreservesNameAfterFailure() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk")]
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        let tooLong = await model.rename(model.items[0], name: String(repeating: "🍎", count: 51))
        XCTAssertFalse(tooLong)
        XCTAssertEqual(model.items[0].name, "Milk")
        client.updateShoppingError = APIError.badRequest(serverMessage: "Try another name.")
        let failed = await model.rename(model.items[0], name: "Oat milk")
        XCTAssertFalse(failed)
        XCTAssertEqual(model.items[0].name, "Milk")
        XCTAssertNotNil(model.rowErrors["milk"])
        client.updateShoppingError = nil
        let saved = await model.rename(model.items[0], name: "  Oat milk  ")
        XCTAssertTrue(saved)
        XCTAssertEqual(model.items[0].name, "Oat milk")
    }

    func testUndoWhilePurchaseIsInFlightUsesTheConfirmedVersion() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk")]
        let started = expectation(description: "Purchase starts")
        var release: CheckedContinuation<Void, Never>?
        client.beforeUpdateShopping = { _, change in
            if change.purchased == true {
                started.fulfill()
                await withCheckedContinuation { release = $0 }
            }
        }
        let model = ShoppingViewModel(apiClient: client)
        await model.load()
        let purchase = Task { await model.toggle(model.items[0]) }
        await fulfillment(of: [started], timeout: 2)
        XCTAssertTrue(model.isPurchased(model.items[0]))
        let undo = Task { await model.toggle(model.items[0]) }
        await Task.yield()
        release?.resume()
        _ = await purchase.value
        _ = await undo.value
        XCTAssertFalse(client.shoppingItems[0].purchased)
        XCTAssertEqual(client.shoppingItems[0].version, 2)
        XCTAssertEqual(model.visibleItems.count, 1)
        XCTAssertNil(model.rowErrors["milk"])
    }

    func testPurchaseKeepsItsPositionForFiveSecondsThenHidesWithoutDeleting() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "milk", name: "Milk"), ShoppingItem(id: "bread", name: "Bread")]
        var now = Date(timeIntervalSince1970: 1_000)
        let model = ShoppingViewModel(apiClient: client, now: { now })
        await model.load()

        _ = await model.toggle(model.items[0])

        XCTAssertEqual(model.visibleItems.map(\.id), ["milk", "bread"])
        now.addTimeInterval(4.9)
        model.expirePurchases()
        XCTAssertEqual(model.visibleItems.map(\.id), ["milk", "bread"])
        now.addTimeInterval(0.1)
        model.expirePurchases()
        XCTAssertEqual(model.visibleItems.map(\.id), ["bread"])
        XCTAssertTrue(client.shoppingItems.first { $0.id == "milk" }!.purchased)
    }
}
