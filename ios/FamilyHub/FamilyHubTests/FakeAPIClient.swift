import Foundation
@testable import FamilyHub

final class FakeAPIClient: APIClientProtocol {
    var dashboardResult: Result<DashboardStats, Error> = .success(
        DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [])
    )
    var choresResult: Result<[Chore], Error> = .success([])
    var completeChoreResult: Result<Void, Error> = .success(())
    var mealsResult: Result<[MealPlan], Error> = .success([])
    var recipesResult: Result<[Recipe], Error> = .success([])
    var recipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
    var calendarResult: Result<CalendarResponse, Error> = .success(CalendarResponse(chores: [], events: [], meals: []))
    var usersResult: Result<[User], Error> = .success([])
    var saveMealResult: Result<MealPlan, Error> = .success(MealPlan(date: "", mealType: "", name: "", notes: "", recipeID: nil))
    var deleteMealResult: Result<Void, Error> = .success(())

    func fetchDashboardStats() async throws -> DashboardStats { try dashboardResult.get() }
    func fetchChores() async throws -> [Chore] { try choresResult.get() }
    func completeChore(id: String) async throws { try completeChoreResult.get() }
    func fetchMeals(week: Date) async throws -> [MealPlan] { try mealsResult.get() }
    func fetchRecipes() async throws -> [Recipe] { try recipesResult.get() }
    func fetchRecipe(id: String) async throws -> Recipe { try recipeResult.get() }
    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse { try calendarResult.get() }
    func fetchUsers() async throws -> [User] { try usersResult.get() }
    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan { try saveMealResult.get() }
    func deleteMeal(date: String, mealType: String) async throws { try deleteMealResult.get() }
}
