import Foundation
@testable import FamilyHub

final class FakeAPIClient: APIClientProtocol {
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
    var categoriesResult: Result<[FamilyHub.Category], Error> = .success([])
    var createCategoryResult: Result<FamilyHub.Category, Error> = .success(FamilyHub.Category(id: "1", name: "Test"))
    var updateCategoryResult: Result<FamilyHub.Category, Error> = .success(FamilyHub.Category(id: "1", name: "Updated"))
    var deleteCategoryResult: Result<Void, Error> = .success(())
    var tokensResult: Result<[APIToken], Error> = .success([])
    var createTokenResult: Result<CreatedToken, Error> = .success(CreatedToken(id: "1", name: "Test", plaintext: "abc123"))
    var deleteTokenResult: Result<Void, Error> = .success(())

    func fetchDashboardStats() async throws -> DashboardStats { try dashboardResult.get() }
    func fetchChores() async throws -> [Chore] { try choresResult.get() }
    func completeChore(id: String) async throws { try completeChoreResult.get() }
    func createChore(_ request: ChoreRequest) async throws -> Chore { try createChoreResult.get() }
    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore { try updateChoreResult.get() }
    func deleteChore(id: String) async throws { try deleteChoreResult.get() }
    func fetchUserAvatar(id: String) async throws -> Data { try fetchUserAvatarResult.get() }
    func fetchMeals(week: Date) async throws -> [MealPlan] { try mealsResult.get() }
    func fetchRecipes() async throws -> [Recipe] { try recipesResult.get() }
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
    func fetchCategories() async throws -> [FamilyHub.Category] { try categoriesResult.get() }
    func createCategory(name: String) async throws -> FamilyHub.Category { try createCategoryResult.get() }
    func updateCategory(id: String, name: String) async throws -> FamilyHub.Category { try updateCategoryResult.get() }
    func deleteCategory(id: String) async throws { try deleteCategoryResult.get() }
    func fetchTokens() async throws -> [APIToken] { try tokensResult.get() }
    func createToken(name: String) async throws -> CreatedToken { try createTokenResult.get() }
    func deleteToken(id: String) async throws { try deleteTokenResult.get() }
}
