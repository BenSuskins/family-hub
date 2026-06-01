import Foundation

private enum DemoData {
    static let alexID = "demo-user-1"
    static let samID = "demo-user-2"
    static let chickenRecipeID = "demo-recipe-1"
    static let pastaRecipeID = "demo-recipe-2"
    static let mainCourseID = "demo-cat-1"
    static let saladID = "demo-cat-2"

    static let users: [User] = [
        User(id: alexID, name: "Alex Demo", email: "alex@demo.example", avatarURL: "", role: "admin"),
        User(id: samID, name: "Sam Demo", email: "sam@demo.example", avatarURL: "", role: "member"),
    ]

    static let categories: [Category] = [
        Category(id: mainCourseID, name: "Main Course"),
        Category(id: saladID, name: "Salads"),
    ]

    static let recipes: [Recipe] = [
        Recipe(
            id: chickenRecipeID,
            title: "Chicken Stir Fry",
            steps: [
                "Heat oil in a wok over high heat.",
                "Add chicken strips and cook until golden, about 5 minutes.",
                "Add vegetables and stir-fry for 3 minutes.",
                "Pour in sauce, toss to coat, and serve over rice.",
            ],
            ingredients: [
                IngredientGroup(name: "Protein", items: ["500g chicken breast, sliced"]),
                IngredientGroup(name: "Vegetables", items: ["1 red pepper", "1 head broccoli", "2 carrots, julienned"]),
                IngredientGroup(name: "Sauce", items: ["3 tbsp soy sauce", "1 tbsp sesame oil", "2 tsp cornstarch"]),
            ],
            mealType: "dinner",
            servings: 4,
            prepTime: "15 min",
            cookTime: "20 min",
            sourceURL: nil,
            categoryID: mainCourseID,
            hasImage: false
        ),
        Recipe(
            id: pastaRecipeID,
            title: "Pasta Salad",
            steps: [
                "Cook pasta according to package directions; drain and cool.",
                "Combine pasta with vegetables and dressing.",
                "Chill for 30 minutes before serving.",
            ],
            ingredients: [
                IngredientGroup(name: "Base", items: ["300g fusilli pasta", "1 can chickpeas, drained"]),
                IngredientGroup(name: "Vegetables", items: ["1 cucumber, diced", "200g cherry tomatoes", "1 red onion, sliced"]),
                IngredientGroup(name: "Dressing", items: ["4 tbsp olive oil", "2 tbsp red wine vinegar", "1 tsp dried oregano"]),
            ],
            mealType: "lunch",
            servings: 4,
            prepTime: "10 min",
            cookTime: "12 min",
            sourceURL: nil,
            categoryID: saladID,
            hasImage: false
        ),
    ]

    static var chores: [Chore] {
        let yesterday = ISO8601DateFormatter().string(from: Calendar.current.date(byAdding: .day, value: -1, to: Date())!)
        let today = ISO8601DateFormatter().string(from: Date())
        let nextWeek = ISO8601DateFormatter().string(from: Calendar.current.date(byAdding: .day, value: 7, to: Date())!)
        return [
            Chore(id: "demo-chore-1", name: "Vacuum living room", description: "Full vacuum including under the sofa.", status: .overdue, dueDate: yesterday, assignedToUserID: alexID, eligibleAssignees: [alexID, samID], recurrenceType: "weekly", recurrenceValue: "{\"interval\":1,\"days\":[\"monday\"]}", seriesID: "demo-series-1"),
            Chore(id: "demo-chore-2", name: "Do the dishes", description: "", status: .pending, dueDate: today, assignedToUserID: samID, eligibleAssignees: [alexID, samID], recurrenceType: "daily", seriesID: "demo-series-2"),
            Chore(id: "demo-chore-3", name: "Take out trash", description: "All bins — kitchen, bathrooms, recycling.", status: .pending, dueDate: nextWeek, assignedToUserID: alexID, eligibleAssignees: [alexID], recurrenceType: "weekly", recurrenceValue: "{\"interval\":1,\"days\":[\"thursday\"]}", seriesID: "demo-series-3"),
            Chore(id: "demo-chore-4", name: "Grocery shopping", description: "Weekly shop — see fridge list.", status: .pending, dueDate: nextWeek, assignedToUserID: samID),
            Chore(id: "demo-chore-5", name: "Clean bathroom", description: "", status: .completed, dueDate: nil, assignedToUserID: samID),
        ]
    }

    static var meals: [MealPlan] {
        let monday = startOfWeekDate(offset: 0)
        let tuesday = startOfWeekDate(offset: 1)
        let wednesday = startOfWeekDate(offset: 2)
        return [
            MealPlan(date: monday, mealType: "dinner", name: "Chicken Stir Fry", notes: "", recipeID: chickenRecipeID),
            MealPlan(date: tuesday, mealType: "lunch", name: "Pasta Salad", notes: "", recipeID: pastaRecipeID),
            MealPlan(date: wednesday, mealType: "breakfast", name: "Oatmeal", notes: "With berries and honey", recipeID: nil),
        ]
    }

    static var calendarEvent: CalendarEvent {
        let saturday = Calendar.current.date(byAdding: .day, value: daysUntilWeekend(), to: Date())!
        let start = Calendar.current.date(bySettingHour: 18, minute: 0, second: 0, of: saturday)!
        let end = Calendar.current.date(byAdding: .hour, value: 2, to: start)!
        return CalendarEvent(id: "demo-event-1", title: "Family Dinner", description: "Weekly family get-together", location: "Home", startTime: start, endTime: end, allDay: false, color: "#4CAF50")
    }

    private static func startOfWeekDate(offset: Int) -> String {
        let calendar = Calendar.current
        let today = Date()
        let weekday = calendar.component(.weekday, from: today)
        let daysToMonday = (weekday == 1 ? -6 : 2 - weekday)
        let monday = calendar.date(byAdding: .day, value: daysToMonday + offset, to: today)!
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: monday)
    }

    private static func daysUntilWeekend() -> Int {
        let weekday = Calendar.current.component(.weekday, from: Date())
        return weekday <= 7 ? (7 - weekday) : 0
    }
}

@MainActor
final class DemoAPIClient: APIClientProtocol {
    private var chores: [Chore] = DemoData.chores
    private var meals: [MealPlan] = DemoData.meals
    private var recipes: [Recipe] = DemoData.recipes
    private var categories: [Category] = DemoData.categories
    private var tokens: [APIToken] = []

    func fetchDashboardStats() async throws -> DashboardStats {
        let overdue = chores.filter { $0.status == .overdue }
        let dueToday = chores.filter { $0.badge == .dueToday }
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd"
        let today = dateFormatter.string(from: Date())
        let todayMeals = meals.filter { $0.date == today }
        return DashboardStats(
            choresDueToday: dueToday.count,
            choresOverdue: overdue.count,
            choresDueTodayList: dueToday,
            choresOverdueList: overdue,
            mealsThisWeek: meals.count,
            todayMeals: todayMeals
        )
    }

    func fetchChores() async throws -> [Chore] { chores }

    func completeChore(id: String) async throws {
        guard let index = chores.firstIndex(where: { $0.id == id }) else { return }
        let old = chores[index]
        chores[index] = Chore(id: old.id, name: old.name, description: old.description, status: .completed, dueDate: old.dueDate, assignedToUserID: old.assignedToUserID, categoryID: old.categoryID, dueTime: old.dueTime, eligibleAssignees: old.eligibleAssignees, recurrenceType: old.recurrenceType, recurrenceValue: old.recurrenceValue, recurOnComplete: old.recurOnComplete, seriesID: old.seriesID, recurrenceUntil: old.recurrenceUntil, recurrenceCount: old.recurrenceCount)
    }

    func createChore(_ request: ChoreRequest) async throws -> Chore {
        let id = "demo-chore-\(UUID().uuidString)"
        let recurring = (request.recurrenceType ?? "none") != "none"
        let chore = Self.choreFrom(request, id: id, status: .pending, seriesID: recurring ? "demo-series-\(id)" : nil)
        chores.append(chore)
        return chore
    }

    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore {
        guard let index = chores.firstIndex(where: { $0.id == id }) else {
            throw APIError.notFound
        }
        let old = chores[index]
        let updated = Self.choreFrom(request, id: id, status: old.status, seriesID: old.seriesID)
        chores[index] = updated
        return updated
    }

    /// Builds a Chore from a request, re-encoding the structured recurrence fields
    /// into a recurrenceValue JSON string so the Manage view reflects edits.
    private static func choreFrom(_ request: ChoreRequest, id: String, status: ChoreStatus, seriesID: String?) -> Chore {
        var config: [String: Any] = [:]
        if let interval = request.recurrenceInterval { config["interval"] = interval }
        if let days = request.recurrenceDays, !days.isEmpty { config["days"] = days }
        if let dom = request.recurrenceDayOfMonth { config["day_of_month"] = dom }
        if let unit = request.recurrenceUnit { config["unit"] = unit }
        var recurrenceValue = ""
        let type = request.recurrenceType ?? "none"
        if type != "none", type != "daily", !config.isEmpty,
           let data = try? JSONSerialization.data(withJSONObject: config),
           let json = String(data: data, encoding: .utf8) {
            recurrenceValue = json
        }
        return Chore(
            id: id,
            name: request.name,
            description: request.description,
            status: status,
            dueDate: request.dueDate,
            assignedToUserID: request.assignees.first,
            categoryID: request.categoryId,
            dueTime: request.dueTime,
            eligibleAssignees: request.assignees,
            recurrenceType: type,
            recurrenceValue: recurrenceValue,
            recurOnComplete: request.recurOnComplete,
            seriesID: seriesID,
            recurrenceUntil: request.recurrenceUntil,
            recurrenceCount: request.recurrenceCount
        )
    }

    func deleteChore(id: String) async throws {
        chores.removeAll { $0.id == id }
    }

    func fetchUserAvatar(id: String) async throws -> Data { Data() }

    func fetchMeals(week: Date) async throws -> [MealPlan] { meals }

    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan {
        meals.removeAll { $0.date == date && $0.mealType == mealType }
        let meal = MealPlan(date: date, mealType: mealType, name: name, notes: "", recipeID: recipeID)
        meals.append(meal)
        return meal
    }

    func deleteMeal(date: String, mealType: String) async throws {
        meals.removeAll { $0.date == date && $0.mealType == mealType }
    }

    func fetchRecipes(forceRefresh: Bool) async throws -> [Recipe] { recipes }

    func fetchRecipe(id: String) async throws -> Recipe {
        guard let recipe = recipes.first(where: { $0.id == id }) else { throw APIError.notFound }
        return recipe
    }

    func fetchRecipeImage(id: String) async throws -> Data { Data() }

    func createRecipe(_ request: RecipeRequest) async throws -> Recipe {
        let recipe = Recipe(
            id: "demo-recipe-\(UUID().uuidString)",
            title: request.title,
            steps: request.steps,
            ingredients: request.ingredients,
            mealType: request.mealType,
            servings: request.servings,
            prepTime: request.prepTime,
            cookTime: request.cookTime,
            sourceURL: request.sourceURL,
            categoryID: nil,
            hasImage: false
        )
        recipes.append(recipe)
        return recipe
    }

    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe {
        guard let index = recipes.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let updated = Recipe(
            id: id,
            title: request.title,
            steps: request.steps,
            ingredients: request.ingredients,
            mealType: request.mealType,
            servings: request.servings,
            prepTime: request.prepTime,
            cookTime: request.cookTime,
            sourceURL: request.sourceURL,
            categoryID: recipes[index].categoryID,
            hasImage: false
        )
        recipes[index] = updated
        return updated
    }

    func deleteRecipe(id: String) async throws {
        recipes.removeAll { $0.id == id }
    }

    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse {
        CalendarResponse(chores: chores, events: [DemoData.calendarEvent], meals: meals)
    }

    func fetchUsers() async throws -> [User] { DemoData.users }

    func fetchMe() async throws -> User { DemoData.users[0] }

    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User { DemoData.users[0] }

    func deleteAvatar() async throws {}

    func fetchSettings() async throws -> AppSettings {
        AppSettings(familyName: "The Demo Family")
    }

    func updateFamilyName(_ name: String) async throws {}

    func promoteUser(id: String) async throws -> User {
        guard let user = DemoData.users.first(where: { $0.id == id }) else { throw APIError.notFound }
        return User(id: user.id, name: user.name, email: user.email, avatarURL: user.avatarURL, role: "admin")
    }

    func demoteUser(id: String) async throws -> User {
        guard let user = DemoData.users.first(where: { $0.id == id }) else { throw APIError.notFound }
        return User(id: user.id, name: user.name, email: user.email, avatarURL: user.avatarURL, role: "member")
    }

    func fetchCategories() async throws -> [Category] { categories }

    func createCategory(name: String) async throws -> Category {
        let category = Category(id: "demo-cat-\(UUID().uuidString)", name: name)
        categories.append(category)
        return category
    }

    func updateCategory(id: String, name: String) async throws -> Category {
        guard let index = categories.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let updated = Category(id: id, name: name)
        categories[index] = updated
        return updated
    }

    func deleteCategory(id: String) async throws {
        categories.removeAll { $0.id == id }
    }

    func fetchTokens() async throws -> [APIToken] { tokens }

    func createToken(name: String) async throws -> CreatedToken {
        let token = APIToken(id: "demo-token-\(UUID().uuidString)", name: name, createdAt: Date())
        tokens.append(token)
        return CreatedToken(id: token.id, name: token.name, plaintext: "demo-token-preview-only")
    }

    func deleteToken(id: String) async throws {
        tokens.removeAll { $0.id == id }
    }
}

