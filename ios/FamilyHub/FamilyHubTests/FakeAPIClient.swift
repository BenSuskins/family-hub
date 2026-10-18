import Foundation
@testable import FamilyHub
import FamilyHubKit

final class FakeAPIClient: APIClientProtocol {
    var serverIdentifier: String?
    var removeNotificationDeviceError: Error?
    var registerNotificationDeviceError: Error?

    var choreReminders = ChoreReminders(pushAvailable: true)
    var notificationDevices: [String: NotificationDevice] = [:]

    var weatherResult: Result<HouseholdWeather, Error> = .failure(APIError.offline)
    var shoppingItems: [ShoppingItem] = []
    var shoppingError: Error?
    var addShoppingError: Error?
    func fetchShopping() async throws -> [ShoppingItem] {
        if let shoppingError { throw shoppingError }
        return shoppingItems
    }
    func addShopping(name: String, inventoryItemID: String?) async throws -> ShoppingItem {
        if let addShoppingError { throw addShoppingError }
        let item = ShoppingItem(id: UUID().uuidString, name: name, inventoryItemID: inventoryItemID)
        shoppingItems.append(item)
        return item
    }
    func updateShopping(id: String, change: ShoppingChange) async throws -> ShoppingItem {
        guard let index = shoppingItems.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let old = shoppingItems[index]
        let purchased = change.purchased ?? old.purchased
        let item = ShoppingItem(id: id, name: change.name ?? old.name, inventoryItemID: old.inventoryItemID, purchased: purchased, purchaseID: purchased ? UUID().uuidString : nil, version: old.version + 1)
        shoppingItems[index] = item
        return item
    }
    var dashboardResult: Result<DashboardStats, Error> = .success(
        DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [], mealsThisWeek: 0, todayMeals: [])
    )
    var choresResult: Result<[Chore], Error> = .success([])
    var completeChoreResult: Result<Void, Error> = .success(())
    var createChoreResult: Result<Chore, Error> = .failure(APIError.notFound)
    var updateChoreResult: Result<Chore, Error> = .failure(APIError.notFound)
    var deleteChoreResult: Result<Void, Error> = .success(())
    var fetchUserAvatarResult: Result<Data, Error> = .success(Data())
    var mealsResult: Result<[MealPlan], Error> = .success([])
    var mealsByWeek: [String: [MealPlan]] = [:]
    var recipesResult: Result<[Recipe], Error> = .success([])
    var recipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var recipeImageResult: Result<Data, Error> = .success(Data())
    var createRecipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var updateRecipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var deleteRecipeResult: Result<Void, Error> = .success(())
    var calendarResult: Result<CalendarResponse, Error> = .success(CalendarResponse(chores: [], events: [], meals: []))
    var usersResult: Result<[User], Error> = .success([])
    var saveMealResult: Result<MealPlan, Error> = .success(MealPlan(date: "", mealType: "", name: "", notes: "", recipeID: nil))
    var deleteMealResult: Result<Void, Error> = .success(())
    var meResult: Result<User, Error> = .success(User(id: "1", name: "Test User", email: "test@example.com", avatarURL: "", role: "member"))
    var uploadAvatarResult: Result<User, Error> = .success(User(id: "1", name: "Test User", email: "test@example.com", avatarURL: "", role: "member"))
    var deleteAvatarResult: Result<Void, Error> = .success(())
    var settingsResult: Result<AppSettings, Error> = .success(AppSettings(familyName: "Family"))
    var updateFamilyNameResult: Result<Void, Error> = .success(())
    var promoteUserResult: Result<User, Error> = .success(User(id: "2", name: "Other", email: "other@example.com", avatarURL: "", role: "admin"))
    var demoteUserResult: Result<User, Error> = .success(User(id: "2", name: "Other", email: "other@example.com", avatarURL: "", role: "member"))
    var categoriesResult: Result<[ChoreCategory], Error> = .success([])
    var createCategoryResult: Result<ChoreCategory, Error> = .success(ChoreCategory(id: "1", name: "Test"))
    var updateCategoryResult: Result<ChoreCategory, Error> = .success(ChoreCategory(id: "1", name: "Updated"))
    var deleteCategoryResult: Result<Void, Error> = .success(())
    var tokensResult: Result<[APIToken], Error> = .success([])
    var createTokenResult: Result<CreatedToken, Error> = .success(CreatedToken(id: "1", name: "Test", plaintext: "abc123"))
    var deleteTokenResult: Result<Void, Error> = .success(())
    var inventoryResult: Result<[InventoryArea], Error> = .success([])
    var createAreaResult: Result<InventoryArea, Error> = .success(InventoryArea(id: "1", name: "Area", icon: "box", tint: "blue", items: []))
    var updateAreaResult: Result<InventoryArea, Error> = .success(InventoryArea(id: "1", name: "Area", icon: "box", tint: "blue", items: []))
    var deleteAreaResult: Result<Void, Error> = .success(())
    var createItemResult: Result<InventoryItem, Error> = .success(InventoryItem(id: "1", areaID: "1", name: "Item", quantity: 1, unit: "pcs", lowAt: 0))
    var updateItemResult: Result<InventoryItem, Error> = .success(InventoryItem(id: "1", areaID: "1", name: "Item", quantity: 1, unit: "pcs", lowAt: 0))
    var deleteItemResult: Result<Void, Error> = .success(())

    func fetchDashboardStats() async throws -> DashboardStats { try dashboardResult.get() }
    func fetchChores() async throws -> [Chore] { try choresResult.get() }
    func completeChore(id: String) async throws { try completeChoreResult.get() }
    func createChore(_ request: ChoreRequest) async throws -> Chore { try createChoreResult.get() }
    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore { try updateChoreResult.get() }
    func deleteChore(id: String) async throws { try deleteChoreResult.get() }
    func fetchUserAvatar(id: String) async throws -> Data { try fetchUserAvatarResult.get() }
    func fetchMeals(week: Date) async throws -> [MealPlan] {
        if !mealsByWeek.isEmpty { return mealsByWeek[APIDate.dayString(week)] ?? [] }
        return try mealsResult.get()
    }
    func fetchRecipes(forceRefresh: Bool) async throws -> [Recipe] { try recipesResult.get() }
    func fetchRecipe(id: String) async throws -> Recipe { try recipeResult.get() }
    func fetchRecipeImage(id: String) async throws -> Data { try recipeImageResult.get() }
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe { try createRecipeResult.get() }
    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe { try updateRecipeResult.get() }
    func deleteRecipe(id: String) async throws { try deleteRecipeResult.get() }
    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse { try calendarResult.get() }
    func fetchUsers() async throws -> [User] { try usersResult.get() }
    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan { try saveMealResult.get() }
    func deleteMeal(date: String, mealType: String) async throws { try deleteMealResult.get() }
    func fetchMe() async throws -> User { try meResult.get() }
    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User { try uploadAvatarResult.get() }
    func deleteAvatar() async throws { try deleteAvatarResult.get() }
    func fetchSettings() async throws -> AppSettings { try settingsResult.get() }
    func updateFamilyName(_ name: String) async throws { try updateFamilyNameResult.get() }
    func promoteUser(id: String) async throws -> User { try promoteUserResult.get() }
    func demoteUser(id: String) async throws -> User { try demoteUserResult.get() }
    func fetchCategories() async throws -> [ChoreCategory] { try categoriesResult.get() }
    func createCategory(name: String) async throws -> ChoreCategory { try createCategoryResult.get() }
    func updateCategory(id: String, name: String) async throws -> ChoreCategory { try updateCategoryResult.get() }
    func deleteCategory(id: String) async throws { try deleteCategoryResult.get() }
    func fetchTokens() async throws -> [APIToken] { try tokensResult.get() }
    func createToken(name: String) async throws -> CreatedToken { try createTokenResult.get() }
    func deleteToken(id: String) async throws { try deleteTokenResult.get() }
    func fetchInventory() async throws -> [InventoryArea] { try inventoryResult.get() }
    func createArea(_ request: AreaRequest) async throws -> InventoryArea { try createAreaResult.get() }
    func updateArea(id: String, _ request: AreaRequest) async throws -> InventoryArea { try updateAreaResult.get() }
    func deleteArea(id: String) async throws { try deleteAreaResult.get() }
    func createItem(areaID: String, _ request: ItemRequest) async throws -> InventoryItem { try createItemResult.get() }
    func updateItem(id: String, _ request: ItemRequest) async throws -> InventoryItem { try updateItemResult.get() }
    func deleteItem(id: String) async throws { try deleteItemResult.get() }
}

extension FakeAPIClient {

    func sendFamilyBroadcast(message: String, requestID: String) async throws -> BroadcastResult {
        BroadcastResult(accepted: notificationDevices.count, failed: 0)
    }
    func fetchChoreReminders() async throws -> ChoreReminders { choreReminders }
    func updateChoreReminders(_ settings: ChoreReminders) async throws { choreReminders = settings }
    func registerNotificationDevice(_ device: NotificationDevice) async throws {
        if let registerNotificationDeviceError { throw registerNotificationDeviceError }
        notificationDevices[device.id] = device
    }
    func removeNotificationDevice(id: String) async throws {
        if let removeNotificationDeviceError { throw removeNotificationDeviceError }
        notificationDevices.removeValue(forKey: id)
    }

    func fetchHomeInfo() async throws -> HomeInfo { HomeInfo() }
    func updateHomeInfo(_ home: HomeInfo) async throws {}
    func fetchWeather() async throws -> HouseholdWeather { try weatherResult.get() }
    func fetchDepartures() async throws -> HouseholdDepartures { throw APIError.offline }
    func deleteShopping(id: String) async throws { shoppingItems.removeAll { $0.id == id } }
    func clearPurchasedShopping() async throws { shoppingItems.removeAll(where: \.purchased) }
    func restockShopping(id: String, request: ShoppingRestock) async throws -> ShoppingItem { throw APIError.notFound }
}
