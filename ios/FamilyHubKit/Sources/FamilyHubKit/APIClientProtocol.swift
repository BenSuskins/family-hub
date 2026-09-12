import Foundation

public enum ViewState<T> {
    case idle
    case loading
    case loaded(T)
    case failed(APIError)
}

/// Supplies the bearer token for outgoing requests. APIClient depends on this
/// rather than on AuthManager, which is iOS-only: it drives
/// ASWebAuthenticationSession, which does not exist on watchOS. AnyObject so the
/// client can hold it weakly and avoid a retain cycle.
public protocol TokenProviding: AnyObject {
    func validAPIToken() async throws -> String
}

public protocol APIClientProtocol: AnyObject {
    func fetchDashboardStats() async throws -> DashboardStats
    func fetchChores() async throws -> [Chore]
    func completeChore(id: String) async throws
    func createChore(_ request: ChoreRequest) async throws -> Chore
    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore
    func deleteChore(id: String) async throws
    func fetchUserAvatar(id: String) async throws -> Data
    func fetchMeals(week: Date) async throws -> [MealPlan]
    func fetchRecipes(forceRefresh: Bool) async throws -> [Recipe]
    func fetchRecipe(id: String) async throws -> Recipe
    func fetchRecipeImage(id: String) async throws -> Data
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe
    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe
    func deleteRecipe(id: String) async throws
    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse
    func fetchUsers() async throws -> [User]
    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan
    func deleteMeal(date: String, mealType: String) async throws
    func fetchMe() async throws -> User

    // Avatar
    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User
    func deleteAvatar() async throws

    // Settings
    func fetchSettings() async throws -> AppSettings
    func updateFamilyName(_ name: String) async throws

    // User management (admin)
    func promoteUser(id: String) async throws -> User
    func demoteUser(id: String) async throws -> User

    // Categories (admin)
    func fetchCategories() async throws -> [ChoreCategory]
    func createCategory(name: String) async throws -> ChoreCategory
    func updateCategory(id: String, name: String) async throws -> ChoreCategory
    func deleteCategory(id: String) async throws

    // API tokens (admin)
    func fetchTokens() async throws -> [APIToken]
    func createToken(name: String) async throws -> CreatedToken
    func deleteToken(id: String) async throws

    // Inventory
    func fetchInventory() async throws -> [InventoryArea]
    func createArea(_ request: AreaRequest) async throws -> InventoryArea
    func updateArea(id: String, _ request: AreaRequest) async throws -> InventoryArea
    func deleteArea(id: String) async throws
    func createItem(areaID: String, _ request: ItemRequest) async throws -> InventoryItem
    func updateItem(id: String, _ request: ItemRequest) async throws -> InventoryItem
    func deleteItem(id: String) async throws
}

extension APIClientProtocol {
    /// Cache-friendly convenience used by callers that don't force a refresh.
    public func fetchRecipes() async throws -> [Recipe] {
        try await fetchRecipes(forceRefresh: false)
    }
}
