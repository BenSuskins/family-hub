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
    func fetchMeals(week: Date) async throws -> [MealPlan]
    func fetchRecipes() async throws -> [Recipe]
    func fetchRecipe(id: String) async throws -> Recipe
    func fetchCalendar(month: Date) async throws -> [Chore]
}
