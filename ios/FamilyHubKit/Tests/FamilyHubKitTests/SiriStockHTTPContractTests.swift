import XCTest
import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif
@testable import FamilyHubKit

final class SiriStockHTTPContractTests: XCTestCase {
    @MainActor
    func testStockChangesAgainstTheGoAPI() async throws {
        guard let address = ProcessInfo.processInfo.environment["FAMILY_HUB_STOCK_CONTRACT_URL"],
              let baseURL = URL(string: address), baseURL.host == "127.0.0.1" else {
            throw XCTSkip("Run RUN_SWIFT_STOCK_CONTRACT=1 go test ./internal/handlers -run TestInventoryStockSwiftContract from server/.")
        }
        let provider = StaticTokenProvider(token: "siri-contract-token")
        let client = APIClient(baseURL: baseURL, tokenProvider: provider)
        let area = try await client.createArea(AreaRequest(name: "Stock contract", icon: "box", tint: "blue"))
        let item = try await client.createItem(areaID: area.id, ItemRequest(name: "Pods", quantity: 12, unit: "pods", lowAt: 3))
        let changed = try await client.changeStock(id: item.id, change: .quantityDelta(-2))
        XCTAssertEqual(changed.quantity, 10)
        XCTAssertEqual(changed.unit, "pods")
        XCTAssertEqual(changed.lowAt, 3)
        let empty = try await client.changeStock(id: item.id, change: .quantityDelta(-20))
        XCTAssertEqual(empty.quantity, 0)
        do {
            _ = try await client.changeStock(id: item.id, change: .level(50))
            XCTFail("Count items reject percentage changes.")
        } catch APIError.badRequest { }
        let fetched = try await client.fetchInventory()
        XCTAssertEqual(fetched.flatMap(\.items).first { $0.id == item.id }?.quantity, 0)
        let level = try await client.createItem(areaID: area.id, ItemRequest(name: "Soap", trackingMode: .level, quantity: 0, level: 80, unit: "", lowAt: 20))
        let saved = try await client.changeStock(id: level.id, change: .level(25))
        XCTAssertEqual(saved.level, 25)
        do {
            _ = try await client.changeStock(id: "uncertain", change: .quantityDelta(1))
            XCTFail("A failed write must surface an error.")
        } catch APIError.server(let status, _) { XCTAssertEqual(status, 503) }
    }
}
