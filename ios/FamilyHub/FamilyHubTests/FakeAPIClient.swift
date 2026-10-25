import Foundation
@testable import FamilyHub
import FamilyHubKit

final class FakeAPIClient: APIClientProtocol {
    var serverIdentifier: String?
    var commuteResult: Result<CommuteBoard, Error> = .failure(APIError.offline)
    struct CommuteRequest: Equatable {
        let origin: String
        let destination: String
        let count: Int
    }
    var commuteRequests: [CommuteRequest] = []
    var beforeFetchCommute: (() async -> Void)?

    func fetchCommuteTrains(from origin: String, to destination: String, count: Int) async throws -> CommuteBoard {
        commuteRequests.append(CommuteRequest(origin: origin, destination: destination, count: count))
        await beforeFetchCommute?()
        return try commuteResult.get()
    }
    var removeNotificationDeviceError: Error?
    var registerNotificationDeviceError: Error?

    var choreReminders = ChoreReminders(pushAvailable: true)
    var binSchedule = BinScheduleSettings(enabled: false, uploadedAt: "", collectionCount: 0, pushAvailable: true)
    func fetchBinSchedule() async throws -> BinScheduleSettings { binSchedule }
    func uploadBinSchedule(from url: URL) async throws -> BinScheduleSettings {
        binSchedule = BinScheduleSettings(enabled: true, uploadedAt: "2026-10-25T12:00:00Z", collectionCount: 12, pushAvailable: true)
        return binSchedule
    }
    func updateBinReminders(enabled: Bool) async throws -> BinScheduleSettings {
        binSchedule = BinScheduleSettings(enabled: enabled, uploadedAt: binSchedule.uploadedAt, collectionCount: binSchedule.collectionCount, pushAvailable: binSchedule.pushAvailable)
        return binSchedule
    }
    var notificationDevices: [String: NotificationDevice] = [:]

    var weatherResult: Result<HouseholdWeather, Error> = .failure(APIError.offline)
    var shoppingItems: [ShoppingItem] = []
    var shoppingError: Error?
    var addShoppingError: Error?
    var beforeAddShopping: (() async -> Void)?
    func fetchShopping() async throws -> [ShoppingItem] {
        if let shoppingError { throw shoppingError }
        return shoppingItems
    }
    func addShopping(name: String, inventoryItemID: String?) async throws -> ShoppingItem {
        await beforeAddShopping?()
        if let addShoppingError { throw addShoppingError }
        let item = ShoppingItem(id: UUID().uuidString, name: name, inventoryItemID: inventoryItemID)
        shoppingItems.append(item)
        return item
    }
    var beforeUpdateShopping: ((String, ShoppingChange) async -> Void)?
    var updateShoppingError: Error?
    func updateShopping(id: String, change: ShoppingChange) async throws -> ShoppingItem {
        await beforeUpdateShopping?(id, change)
        if let updateShoppingError { throw updateShoppingError }
        guard let index = shoppingItems.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let old = shoppingItems[index]
        guard change.version == old.version else { throw APIError.conflict }
        let purchased = change.purchased ?? old.purchased
        let item = ShoppingItem(id: id, name: change.name ?? old.name, inventoryItemID: old.inventoryItemID, purchased: purchased, purchaseID: purchased ? (old.purchased ? old.purchaseID : UUID().uuidString) : nil, restocked: old.restocked, version: old.version + 1)
        shoppingItems[index] = item
        return item
    }
    var dashboardResult: Result<DashboardStats, Error> = .success(
        DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [], mealsThisWeek: 0, todayMeals: [])
    )
    var choresResult: Result<[Chore], Error> = .success([])
    var completeChoreErrors: [String: Error] = [:]
    var afterCompleteChore: ((String) -> Void)?
    var homeInfo = HomeInfo()
    var completeChoreResult: Result<Void, Error> = .success(())
    var createChoreResult: Result<Chore, Error> = .failure(APIError.notFound)
    var updateChoreResult: Result<Chore, Error> = .failure(APIError.notFound)
    var deleteChoreResult: Result<Void, Error> = .success(())
    var fetchUserAvatarResult: Result<Data, Error> = .success(Data())
    var mealsResult: Result<[MealPlan], Error> = .success([])
    var mealsByWeek: [String: [MealPlan]] = [:]
    var beforeFetchMeals: ((Date) async -> Void)?
    var recipesResult: Result<[Recipe], Error> = .success([])
    var recipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var recipeImageResult: Result<Data, Error> = .success(Data())
    var createRecipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var updateRecipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var deleteRecipeResult: Result<Void, Error> = .success(())
    var calendarResult: Result<CalendarResponse, Error> = .success(CalendarResponse(chores: [], events: [], meals: []))
    var calendarsByMonth: [String: CalendarResponse] = [:]
    var beforeFetchCalendar: ((Date) async throws -> Void)?
    var calendarRequests: [String] = []
    var calendarSubscriptions: [CalendarSubscription] = []
    var usersResult: Result<[User], Error> = .success([])
    var saveMealResult: Result<MealPlan, Error> = .success(MealPlan(date: "", mealType: "", name: "", notes: "", recipeID: nil))
    var deleteMealResult: Result<Void, Error> = .success(())
    var meResult: Result<User, Error> = .success(User(id: "1", name: "Test User", email: "test@example.com", avatarURL: "", role: "member"))
    var uploadAvatarResult: Result<User, Error> = .success(User(id: "1", name: "Test User", email: "test@example.com", avatarURL: "", role: "member"))
    var deleteAvatarResult: Result<Void, Error> = .success(())
    var beforeFetchSettings: (() async -> Void)?
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
    var changeStockError: Error?
    var inventoryResult: Result<[InventoryArea], Error> = .success([])
    var createAreaResult: Result<InventoryArea, Error> = .success(InventoryArea(id: "1", name: "Area", icon: "box", tint: "blue", items: []))
    var updateAreaResult: Result<InventoryArea, Error> = .success(InventoryArea(id: "1", name: "Area", icon: "box", tint: "blue", items: []))
    var deleteAreaResult: Result<Void, Error> = .success(())
    var createItemResult: Result<InventoryItem, Error> = .success(InventoryItem(id: "1", areaID: "1", name: "Item", quantity: 1, unit: "pcs", lowAt: 0))
    var updateItemResult: Result<InventoryItem, Error> = .success(InventoryItem(id: "1", areaID: "1", name: "Item", quantity: 1, unit: "pcs", lowAt: 0))
    var deleteItemResult: Result<Void, Error> = .success(())

    var choreDefinitions: [ChoreDefinition] = []
    var skipChoreResult: Result<Void, Error> = .success(())
    var historyPage = ChoreHistoryPage(items: [])
    func fetchChoreDefinitions(includeInactive: Bool) async throws -> [ChoreDefinition] {
        choreDefinitions.filter { includeInactive || $0.active }
    }
    func createChoreDefinition(_ request: ChoreRequest) async throws -> ChoreDefinition {
        let chore = try createChoreResult.get()
        let definition = ChoreDefinition(id: chore.id, chore: chore, currentTask: chore, active: true)
        choreDefinitions.append(definition)
        return definition
    }
    func updateChoreDefinition(id: String, _ request: ChoreRequest) async throws -> ChoreDefinition {
        let chore = try updateChoreResult.get()
        let definition = ChoreDefinition(id: id, chore: chore, currentTask: chore, active: true)
        choreDefinitions.removeAll { $0.id == id }
        choreDefinitions.append(definition)
        return definition
    }
    func deleteChoreDefinition(id: String) async throws {
        try deleteChoreResult.get()
        choreDefinitions.removeAll { $0.id == id }
    }
    func fetchChoreHistory(definitionID: String, cursor: String?) async throws -> ChoreHistoryPage { historyPage }
    func skipChore(id: String) async throws { try skipChoreResult.get() }

    func fetchDashboardStats() async throws -> DashboardStats { try dashboardResult.get() }
    func fetchChores() async throws -> [Chore] { try choresResult.get() }
    func completeChore(id: String) async throws {
        if let error = completeChoreErrors[id] { throw error }
        try completeChoreResult.get()
        choresResult = .success(try choresResult.get().map { $0.id == id ? $0.completed : $0 })
        afterCompleteChore?(id)
    }
    func createChore(_ request: ChoreRequest) async throws -> Chore { try createChoreResult.get() }
    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore { try updateChoreResult.get() }
    func deleteChore(id: String) async throws { try deleteChoreResult.get() }
    func fetchUserAvatar(id: String) async throws -> Data { try fetchUserAvatarResult.get() }
    func fetchMeals(week: Date) async throws -> [MealPlan] {
        await beforeFetchMeals?(week)
        if !mealsByWeek.isEmpty { return mealsByWeek[APIDate.dayString(week)] ?? [] }
        return try mealsResult.get()
    }
    func fetchRecipes(forceRefresh: Bool) async throws -> [Recipe] { try recipesResult.get() }
    func fetchRecipe(id: String) async throws -> Recipe { try recipeResult.get() }
    func fetchRecipeImage(id: String) async throws -> Data { try recipeImageResult.get() }
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe { try createRecipeResult.get() }
    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe { try updateRecipeResult.get() }
    func deleteRecipe(id: String) async throws { try deleteRecipeResult.get() }
    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse {
        calendarRequests.append(APIDate.monthString(date))
        try await beforeFetchCalendar?(date)
        if !calendarsByMonth.isEmpty {
            return calendarsByMonth[APIDate.monthString(date)] ?? CalendarResponse(chores: [], events: [], meals: [])
        }
        return try calendarResult.get()
    }
    func fetchCalendarSubscriptions() async throws -> [CalendarSubscription] { calendarSubscriptions }
    func createCalendarSubscription(name: String, url: String, color: String) async throws -> CalendarSubscription {
        let subscription = CalendarSubscription(id: UUID().uuidString, name: name, url: url, color: color)
        calendarSubscriptions.append(subscription)
        return subscription
    }
    func updateCalendarSubscriptionColor(id: String, color: String) async throws -> CalendarSubscription {
        guard let index = calendarSubscriptions.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let old = calendarSubscriptions[index]
        let updated = CalendarSubscription(id: old.id, name: old.name, url: old.url, color: color, lastFetchedAt: old.lastFetchedAt, createdAt: old.createdAt)
        calendarSubscriptions[index] = updated
        return updated
    }
    func refreshCalendarSubscription(id: String) async throws {
        guard calendarSubscriptions.contains(where: { $0.id == id }) else { throw APIError.notFound }
    }
    func deleteCalendarSubscription(id: String) async throws { calendarSubscriptions.removeAll { $0.id == id } }
    func downloadDatabaseBackup() async throws -> URL {
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("family-hub-demo.db.gz")
        try Data("Demo backup archive".utf8).write(to: url)
        return url
    }
    func restoreDatabaseBackup(from url: URL) async throws {
        guard url.lastPathComponent.hasSuffix(".db.gz") else { throw APIError.badRequest(serverMessage: "Select a .db.gz backup.") }
    }
    func fetchUsers() async throws -> [User] { try usersResult.get() }
    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan { try saveMealResult.get() }
    func deleteMeal(date: String, mealType: String) async throws { try deleteMealResult.get() }
    func fetchMe() async throws -> User { try meResult.get() }
    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User { try uploadAvatarResult.get() }
    func deleteAvatar() async throws { try deleteAvatarResult.get() }
    func fetchSettings() async throws -> AppSettings {
        let result = settingsResult
        await beforeFetchSettings?()
        return try result.get()
    }
    func updateMealWeekStart(_ weekday: MealWeekday) async throws {
        try updateFamilyNameResult.get()
        let settings = try settingsResult.get()
        settingsResult = .success(AppSettings(familyName: settings.familyName, mealWeekStart: weekday))
    }
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
    func changeStock(id: String, change: InventoryStockChange) async throws -> InventoryItem {
        guard let item = try inventoryResult.get().flatMap(\.items).first(where: { $0.id == id }) else { throw APIError.notFound }
        let saved = try change.applying(to: item)
        inventoryResult = .success(try inventoryResult.get().map { area in
            InventoryArea(id: area.id, name: area.name, icon: area.icon, tint: area.tint,
                          items: area.items.map { $0.id == id ? saved : $0 })
        })
        if let changeStockError { throw changeStockError }
        return saved
    }

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

    func fetchHomeInfo() async throws -> HomeInfo { homeInfo }
    func updateHomeInfo(_ home: HomeInfo) async throws {}
    func fetchWeather() async throws -> HouseholdWeather { try weatherResult.get() }
    func fetchDepartures() async throws -> HouseholdDepartures { throw APIError.offline }
    func deleteShopping(id: String) async throws { shoppingItems.removeAll { $0.id == id } }
    func clearPurchasedShopping() async throws { shoppingItems.removeAll(where: \.purchased) }
    func restockShopping(id: String, request: ShoppingRestock) async throws -> ShoppingItem { throw APIError.notFound }
}
