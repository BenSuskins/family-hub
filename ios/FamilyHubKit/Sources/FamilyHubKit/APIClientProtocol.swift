import Foundation

public enum ViewState<T> {
    case idle
    case loading
    case loaded(T)
    case failed(APIError)
}

public enum RecipeImageVariant: String, CaseIterable, Sendable {
    case thumbnail
    case original
}

/// Supplies the bearer token for outgoing requests. APIClient depends on this
/// rather than on AuthManager, which is iOS-only: it drives
/// ASWebAuthenticationSession, which does not exist on watchOS. AnyObject so the
/// client can hold it weakly and avoid a retain cycle.
public protocol TokenProviding: AnyObject {
    func validAPIToken() async throws -> String
    func cacheIdentity() async throws -> String?
}

public extension TokenProviding {
    func cacheIdentity() async throws -> String? { nil }
}

public protocol APIClientProtocol: AnyObject {
    func fetchDashboardStats() async throws -> DashboardStats
    func fetchDashboardStats(forceRefresh: Bool) async throws -> DashboardStats
    func fetchChores() async throws -> [Chore]
    func fetchChoreDefinitions(includeInactive: Bool) async throws -> [ChoreDefinition]
    func fetchChoreDefinitions(includeInactive: Bool, forceRefresh: Bool) async throws -> [ChoreDefinition]
    func createChoreDefinition(_ request: ChoreRequest) async throws -> ChoreDefinition
    func updateChoreDefinition(id: String, _ request: ChoreRequest) async throws -> ChoreDefinition
    func deleteChoreDefinition(id: String) async throws
    func fetchChoreHistory(definitionID: String, cursor: String?) async throws -> ChoreHistoryPage
    func skipChore(id: String) async throws
    func completeChore(id: String) async throws
    func createChore(_ request: ChoreRequest) async throws -> Chore
    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore
    func deleteChore(id: String) async throws
    func fetchUserAvatar(id: String) async throws -> Data
    func fetchMeals(week: Date) async throws -> [MealPlan]
    func fetchMeals(week: Date, forceRefresh: Bool) async throws -> [MealPlan]
    func fetchRecipes(forceRefresh: Bool) async throws -> [Recipe]
    func fetchRecipe(id: String) async throws -> Recipe
    func fetchRecipeImage(id: String) async throws -> Data
    func fetchRecipeImage(id: String, variant: RecipeImageVariant) async throws -> Data
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe
    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe
    func deleteRecipe(id: String) async throws
    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse
    func fetchCalendar(view: String, date: Date, forceRefresh: Bool) async throws -> CalendarResponse
    func fetchCalendarSubscriptions() async throws -> [CalendarSubscription]
    func createCalendarSubscription(name: String, url: String, color: String) async throws -> CalendarSubscription
    func updateCalendarSubscriptionColor(id: String, color: String) async throws -> CalendarSubscription
    func refreshCalendarSubscription(id: String) async throws
    func deleteCalendarSubscription(id: String) async throws
    func downloadDatabaseBackup() async throws -> URL
    func restoreDatabaseBackup(from url: URL) async throws
    func fetchBinSchedule() async throws -> BinScheduleSettings
    func uploadBinSchedule(from url: URL) async throws -> BinScheduleSettings
    func updateBinReminders(enabled: Bool) async throws -> BinScheduleSettings
    func fetchUsers() async throws -> [User]
    func fetchUsers(forceRefresh: Bool) async throws -> [User]
    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan
    func deleteMeal(date: String, mealType: String) async throws
    func fetchMe() async throws -> User
    func fetchMe(forceRefresh: Bool) async throws -> User
    func fetchCommuteTrains(from origin: String, to destination: String, count: Int) async throws -> CommuteBoard

    // Avatar
    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User
    func deleteAvatar() async throws

    // Chore reminders
    func sendFamilyBroadcast(message: String, requestID: String) async throws -> BroadcastResult
    func fetchChoreReminders() async throws -> ChoreReminders
    func updateChoreReminders(_ settings: ChoreReminders) async throws
    func registerNotificationDevice(_ device: NotificationDevice) async throws
    func removeNotificationDevice(id: String) async throws

    // Settings
    func fetchSettings() async throws -> AppSettings
    func updateMealWeekStart(_ weekday: MealWeekday) async throws
    func updateFamilyName(_ name: String) async throws

    // User management (admin)
    func promoteUser(id: String) async throws -> User
    func demoteUser(id: String) async throws -> User

    // Categories (admin)
    func fetchCategories() async throws -> [ChoreCategory]
    func fetchCategories(forceRefresh: Bool) async throws -> [ChoreCategory]
    func createCategory(name: String) async throws -> ChoreCategory
    func updateCategory(id: String, name: String) async throws -> ChoreCategory
    func deleteCategory(id: String) async throws

    // API tokens (admin)
    func fetchTokens() async throws -> [APIToken]
    func createToken(name: String) async throws -> CreatedToken
    func deleteToken(id: String) async throws

    var serverIdentifier: String? { get }
    func snapshotMetadata(for key: String) async -> SnapshotMetadata?
    func fetchHomeInfo() async throws -> HomeInfo
    func updateHomeInfo(_ home: HomeInfo) async throws
    func fetchWeather() async throws -> HouseholdWeather
    func fetchDepartures() async throws -> HouseholdDepartures
    func fetchShopping() async throws -> [ShoppingItem]
    func addShopping(name: String, inventoryItemID: String?) async throws -> ShoppingItem
    func updateShopping(id: String, change: ShoppingChange) async throws -> ShoppingItem
    func deleteShopping(id: String) async throws
    func clearPurchasedShopping() async throws
    func restockShopping(id: String, request: ShoppingRestock) async throws -> ShoppingItem

    // Inventory
    func fetchInventory() async throws -> [InventoryArea]
    func createArea(_ request: AreaRequest) async throws -> InventoryArea
    func updateArea(id: String, _ request: AreaRequest) async throws -> InventoryArea
    func deleteArea(id: String) async throws
    func createItem(areaID: String, _ request: ItemRequest) async throws -> InventoryItem
    func updateItem(id: String, _ request: ItemRequest) async throws -> InventoryItem
    func changeStock(id: String, change: InventoryStockChange) async throws -> InventoryItem
    func deleteItem(id: String) async throws
}

public struct BinScheduleSettings: Codable, Equatable {
    public let enabled: Bool
    public let uploadedAt: String
    public let collectionCount: Int
    public let pushAvailable: Bool

    public init(enabled: Bool, uploadedAt: String, collectionCount: Int, pushAvailable: Bool) {
        self.enabled = enabled
        self.uploadedAt = uploadedAt
        self.collectionCount = collectionCount
        self.pushAvailable = pushAvailable
    }

    enum CodingKeys: String, CodingKey {
        case enabled
        case uploadedAt = "uploaded_at"
        case collectionCount = "collection_count"
        case pushAvailable = "push_available"
    }
}

extension APIClientProtocol {
    public func fetchDashboardStats(forceRefresh: Bool) async throws -> DashboardStats { try await fetchDashboardStats() }
    public func fetchMeals(week: Date, forceRefresh: Bool) async throws -> [MealPlan] { try await fetchMeals(week: week) }
    public func fetchCalendar(view: String, date: Date, forceRefresh: Bool) async throws -> CalendarResponse { try await fetchCalendar(view: view, date: date) }
    public func fetchChoreDefinitions(includeInactive: Bool, forceRefresh: Bool) async throws -> [ChoreDefinition] { try await fetchChoreDefinitions(includeInactive: includeInactive) }
    public func fetchUsers(forceRefresh: Bool) async throws -> [User] { try await fetchUsers() }
    public func fetchMe(forceRefresh: Bool) async throws -> User { try await fetchMe() }
    public func fetchCategories(forceRefresh: Bool) async throws -> [ChoreCategory] { try await fetchCategories() }
    public func snapshotMetadata(for key: String) async -> SnapshotMetadata? { nil }

    public func fetchRecipeImage(id: String, variant: RecipeImageVariant) async throws -> Data {
        try await fetchRecipeImage(id: id)
    }

    /// Cache-friendly convenience used by callers that don't force a refresh.
    public func fetchRecipes() async throws -> [Recipe] {
        try await fetchRecipes(forceRefresh: false)
    }
}
