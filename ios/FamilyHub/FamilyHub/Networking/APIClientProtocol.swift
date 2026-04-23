import Foundation

enum ViewState<T> {
    case idle
    case loading
    case loaded(T)
    case failed(APIError)
}

protocol APIClientProtocol: AnyObject {
    func fetchDashboardStats() async throws -> DashboardStats
    func fetchChores() async throws -> [Chore]
    func completeChore(id: String) async throws
    func createChore(_ request: ChoreRequest) async throws -> Chore
    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore
    func deleteChore(id: String) async throws
    func fetchUserAvatar(id: String) async throws -> Data
    func fetchMeals(week: Date) async throws -> [MealPlan]
    func fetchRecipes() async throws -> [Recipe]
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
    func fetchCategories() async throws -> [Category]
    func createCategory(name: String) async throws -> Category
    func updateCategory(id: String, name: String) async throws -> Category
    func deleteCategory(id: String) async throws

    // API tokens (admin)
    func fetchTokens() async throws -> [APIToken]
    func createToken(name: String) async throws -> CreatedToken
    func deleteToken(id: String) async throws
}
