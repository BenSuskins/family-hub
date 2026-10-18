import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class ShoppingViewModelTests: XCTestCase {
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

    func testProviderFailuresDoNotPreventShoppingSummary() async {
        let client = FakeAPIClient()
        client.shoppingItems = [ShoppingItem(id: "soap", name: "Soap")]
        let model = HouseholdViewModel(apiClient: client)
        await model.load(force: true)
        XCTAssertNil(model.weather)
        XCTAssertNil(model.departures)
        XCTAssertEqual(model.shoppingCount, 1)
        XCTAssertEqual(model.weatherMessage, "Unavailable")
    }
}
