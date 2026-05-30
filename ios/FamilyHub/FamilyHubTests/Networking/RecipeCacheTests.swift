import XCTest
@testable import FamilyHub

final class RecipeCacheTests: XCTestCase {

    private func makeRecipe(id: String, title: String = "Recipe", steps: [String]? = nil) -> Recipe {
        Recipe(id: id, title: title, steps: steps, ingredients: [], mealType: nil, servings: nil, prepTime: nil, cookTime: nil, sourceURL: nil, categoryID: nil, hasImage: false)
    }

    func testListRoundTripDoesNotSeedDetail() async {
        let cache = RecipeCache()
        let summary = makeRecipe(id: "1", steps: nil) // list summaries have no steps
        await cache.storeList([summary])

        let list = await cache.cachedList()
        XCTAssertEqual(list?.map(\.id), ["1"])
        // A list-only id must NOT satisfy a detail lookup.
        let detail = await cache.cachedDetail(id: "1")
        XCTAssertNil(detail)
    }

    func testStoreDetailRoundTripAndUpdatesListEntry() async {
        let cache = RecipeCache()
        await cache.storeList([makeRecipe(id: "1", title: "Old", steps: nil)])

        let full = makeRecipe(id: "1", title: "New", steps: ["a", "b"])
        await cache.storeDetail(full)

        let detail = await cache.cachedDetail(id: "1")
        XCTAssertEqual(detail?.steps, ["a", "b"])
        // The matching list entry is updated in place.
        let listEntry = await cache.cachedList()?.first
        XCTAssertEqual(listEntry?.title, "New")
    }

    func testUpsertAddsToListAndDetail() async {
        let cache = RecipeCache()
        await cache.storeList([makeRecipe(id: "1")])

        await cache.upsert(makeRecipe(id: "2", title: "Added"))

        let ids = await cache.cachedList()?.map(\.id)
        XCTAssertEqual(ids, ["1", "2"])
        let detail = await cache.cachedDetail(id: "2")
        XCTAssertEqual(detail?.title, "Added")
    }

    func testRemoveDropsFromBothStores() async {
        let cache = RecipeCache()
        await cache.storeList([makeRecipe(id: "1"), makeRecipe(id: "2")])
        await cache.storeDetail(makeRecipe(id: "1", steps: ["x"]))

        await cache.remove(id: "1")

        let ids = await cache.cachedList()?.map(\.id)
        XCTAssertEqual(ids, ["2"])
        let detail = await cache.cachedDetail(id: "1")
        XCTAssertNil(detail)
    }

    func testInvalidateAllClearsBothStores() async {
        let cache = RecipeCache()
        await cache.storeList([makeRecipe(id: "1")])
        await cache.storeDetail(makeRecipe(id: "1", steps: ["x"]))

        await cache.invalidateAll()

        let list = await cache.cachedList()
        let detail = await cache.cachedDetail(id: "1")
        XCTAssertNil(list)
        XCTAssertNil(detail)
    }
}
