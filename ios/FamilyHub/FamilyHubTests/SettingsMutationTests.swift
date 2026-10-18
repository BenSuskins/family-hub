import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class SettingsMutationTests: XCTestCase {
    func testFailedCategoryDeleteKeepsTheCategory() async {
        let client = FakeAPIClient()
        client.categoriesResult = .success([ChoreCategory(id: "category", name: "Kitchen")])
        client.deleteCategoryResult = .failure(APIError.offline)
        let model = CategoriesViewModel(apiClient: client)
        await model.load()
        await model.delete(at: IndexSet(integer: 0))
        XCTAssertEqual(model.categories.map(\.name), ["Kitchen"])
        XCTAssertNotNil(model.actionError)
    }

    func testFailedTokenDeleteKeepsTheToken() async {
        let client = FakeAPIClient()
        client.tokensResult = .success([APIToken(id: "token", name: "Tablet", createdAt: Date())])
        client.deleteTokenResult = .failure(APIError.offline)
        let model = APITokensViewModel(apiClient: client)
        await model.load()
        await model.delete(at: IndexSet(integer: 0))
        XCTAssertEqual(model.tokens.map(\.name), ["Tablet"])
        XCTAssertNotNil(model.actionError)
    }
}
