import Foundation
import UIKit
import FamilyHubKit

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

    static let categories: [ChoreCategory] = [
        ChoreCategory(id: mainCourseID, name: "Cleaning"),
        ChoreCategory(id: saladID, name: "Garden"),
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
        let yesterday = APIDate.iso.string(from: Calendar.current.date(byAdding: .day, value: -1, to: Date())!)
        let today = APIDate.iso.string(from: Date())
        let nextWeek = APIDate.iso.string(from: Calendar.current.date(byAdding: .day, value: 7, to: Date())!)
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

    // Inventory — areas with stocked items (mirrors the design handoff seed).
    static var inventory: [InventoryArea] {
        func item(_ id: String, _ name: String, _ qty: Int, _ unit: String, _ lowAt: Int, area: String) -> InventoryItem {
            InventoryItem(id: id, areaID: area, name: name, quantity: qty, unit: unit, lowAt: lowAt)
        }
        func levelItem(_ id: String, _ name: String, _ level: Int, _ lowAt: Int, area: String) -> InventoryItem {
            InventoryItem(id: id, areaID: area, name: name, trackingMode: .level, quantity: 0, level: level, unit: "", lowAt: lowAt)
        }
        return [
            InventoryArea(id: "laundry", name: "Laundry cupboard", icon: "drop", tint: "blue", items: [
                item("la1", "Washing tablets", 45, "pods", 20, area: "laundry"),
                levelItem("la2", "Fabric softener", 15, 20, area: "laundry"),
                item("la3", "Dryer sheets", 28, "sheets", 20, area: "laundry"),
                item("la4", "Stain remover", 2, "sprays", 1, area: "laundry"),
            ]),
            InventoryArea(id: "pantry", name: "Kitchen pantry", icon: "cart", tint: "orange", items: [
                item("pa1", "Pasta", 6, "packs", 3, area: "pantry"),
                item("pa2", "Tinned tomatoes", 8, "tins", 4, area: "pantry"),
                item("pa3", "Olive oil", 1, "bottles", 2, area: "pantry"),
                item("pa4", "Coffee beans", 2, "bags", 1, area: "pantry"),
                item("pa5", "Cereal", 3, "boxes", 2, area: "pantry"),
            ]),
            InventoryArea(id: "bath", name: "Bathroom", icon: "pills", tint: "teal", items: [
                item("ba1", "Toilet roll", 12, "rolls", 8, area: "bath"),
                item("ba2", "Toothpaste", 4, "tubes", 2, area: "bath"),
                item("ba3", "Shampoo", 1, "bottles", 2, area: "bath"),
                item("ba4", "Hand soap", 3, "refills", 2, area: "bath"),
            ]),
            InventoryArea(id: "cleaning", name: "Cleaning closet", icon: "sparkles", tint: "green", items: [
                item("cl1", "Surface spray", 2, "sprays", 1, area: "cleaning"),
                item("cl2", "Bin bags", 40, "bags", 20, area: "cleaning"),
                item("cl3", "Sponges", 5, "sponges", 4, area: "cleaning"),
                item("cl4", "Bleach", 1, "bottles", 2, area: "cleaning"),
                item("cl5", "Kitchen roll", 3, "rolls", 4, area: "cleaning"),
            ]),
            InventoryArea(id: "medicine", name: "Medicine cabinet", icon: "heart", tint: "purple", items: [
                item("me1", "Paracetamol", 24, "tablets", 8, area: "medicine"),
                item("me2", "Plasters", 30, "pcs", 10, area: "medicine"),
                item("me3", "Ibuprofen", 16, "tablets", 8, area: "medicine"),
                item("me4", "Cough syrup", 1, "bottles", 1, area: "medicine"),
            ]),
        ]
    }

    static var calendarEvent: CalendarEvent {
        let saturday = Calendar.current.date(byAdding: .day, value: daysUntilWeekend(), to: Date())!
        let start = Calendar.current.date(bySettingHour: 18, minute: 0, second: 0, of: saturday)!
        let end = Calendar.current.date(byAdding: .hour, value: 2, to: start)!
        let description = CalendarDemoData.usesLayoutFixtures
            ? "Calendar not up to date? Check the calendar subscription for the latest event details.\n\nSupport the calendar provider for more event updates and details."
            : "Weekly family get-together"
        return CalendarEvent(id: "demo-event-1", title: "Family Dinner", description: description, location: "Home", startTime: start, endTime: end, allDay: false, color: "#4CAF50")
    }

    private static func startOfWeekDate(offset: Int) -> String {
        let calendar = Calendar.current
        let today = Date()
        let weekday = calendar.component(.weekday, from: today)
        let daysToMonday = (weekday == 1 ? -6 : 2 - weekday)
        let monday = calendar.date(byAdding: .day, value: daysToMonday + offset, to: today)!
        return APIDate.dayString(monday)
    }

    private static func daysUntilWeekend() -> Int {
        let weekday = Calendar.current.component(.weekday, from: Date())
        return weekday <= 7 ? (7 - weekday) : 0
    }
}

@MainActor
final class DemoAPIClient: APIClientProtocol {
    private static var didSeedCommute = false
    func fetchCommuteTrains(direction: CommuteDirection, count: Int) async throws -> CommuteBoard {
        guard (1...10).contains(count) else { throw APIError.badRequest(serverMessage: "Count must be between 1 and 10.") }
        let now = Date()
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_GB")
        formatter.timeZone = TimeZone(identifier: "Europe/London")
        formatter.dateFormat = "HH:mm"
        let trains = (0..<min(count, 3)).map { index in
            let departure = now.addingTimeInterval(Double(12 + index * 22) * 60)
            let arrival = departure.addingTimeInterval(55 * 60)
            let delay = index == 1 ? 300.0 : 0
            return CommuteTrain(serviceID: "demo-\(direction.rawValue)-\(index)", operatorName: "Greater Anglia",
                                platform: index == 2 ? nil : direction == .toWork ? "1" : "10",
                                status: index == 1 ? "delayed" : "on_time",
                                scheduledDeparture: formatter.string(from: departure), expectedDeparture: formatter.string(from: departure.addingTimeInterval(delay)),
                                scheduledArrival: formatter.string(from: arrival), expectedArrival: formatter.string(from: arrival.addingTimeInterval(delay)))
        }
        let kelvedon = CommuteStation(code: "KEL", name: "Kelvedon")
        let london = CommuteStation(code: "LST", name: "London Liverpool Street")
        let timestamp = ISO8601DateFormatter().string(from: now)
        return CommuteBoard(direction: direction, origin: direction == .toWork ? kelvedon : london,
                            destination: direction == .toWork ? london : kelvedon,
                            generatedAt: timestamp, fetchedAt: timestamp, trains: trains)
    }

    var choreReminders = ChoreReminders(pushAvailable: true)
    var notificationDevices: [String: NotificationDevice] = [:]

    private var homeInfo = HomeInfo(latitude: 51.8, longitude: 0.7, timezone: "Europe/London")
    private var shoppingItems = [ShoppingItem(id: "demo-shopping", name: "Milk")]
    private var chores: [Chore] = DemoData.chores
    private var meals: [MealPlan] = DemoData.meals
    private var recipes: [Recipe] = DemoData.recipes
    private var users: [User] = DemoData.users
    private var categories: [ChoreCategory] = DemoData.categories
    private var tokens: [APIToken] = []
    private var areas: [InventoryArea] = DemoData.inventory

    init() {
        #if DEBUG
        if ProcessInfo.processInfo.arguments.contains("-demo-commute"), !Self.didSeedCommute {
            Self.didSeedCommute = true
            var preferences = CommutePreferences()
            preferences.toWork = CommuteSchedule(enabled: true, weekdays: Set(1...7), start: "00:00", end: "23:59")
            CommutePreferencesStore().save(preferences, server: "demo", userID: DemoData.alexID)
        }
        if ProcessInfo.processInfo.arguments.contains("-demo-meal-card-images") {
            let today = APIDate.dayString(Date())
            meals = [
                MealPlan(date: today, mealType: "breakfast", name: "Eggs and toast", notes: "", recipeID: DemoData.chickenRecipeID),
                MealPlan(date: today, mealType: "lunch", name: "Roasted vegetables with rice and lemon", notes: "", recipeID: nil),
                MealPlan(date: today, mealType: "dinner", name: "Pasta salad", notes: "", recipeID: DemoData.pastaRecipeID),
            ]
        }
        #endif
    }

    func fetchDashboardStats() async throws -> DashboardStats {
        let overdue = chores.filter { $0.status == .overdue && !deletedDefinitions.contains($0.seriesID ?? $0.id) }
        let dueToday = chores.filter { $0.status == .pending && ($0.badge == .dueToday || $0.dueDate == nil) && !deletedDefinitions.contains($0.seriesID ?? $0.id) }
        let today = APIDate.dayString(Date())
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

    private var deletedDefinitions: Set<String> = []

    func fetchChoreDefinitions(includeInactive: Bool) async throws -> [ChoreDefinition] {
        let groups = Dictionary(grouping: chores) { $0.seriesID ?? $0.id }
        return groups.compactMap { identifier, tasks in
            let task = tasks.first { $0.status == .pending || $0.status == .overdue }
            guard let representative = task ?? tasks.first else { return nil }
            let active = task != nil && !deletedDefinitions.contains(identifier)
            guard includeInactive || active else { return nil }
            return ChoreDefinition(id: identifier, chore: representative, currentTask: active ? task : nil, active: active,
                                   deletedAt: deletedDefinitions.contains(identifier) ? "2026-10-18" : nil)
        }.sorted { $0.chore.name < $1.chore.name }
    }

    func createChoreDefinition(_ request: ChoreRequest) async throws -> ChoreDefinition {
        let chore = try await createChore(request)
        return ChoreDefinition(id: chore.seriesID ?? chore.id, chore: chore, currentTask: chore, active: true)
    }

    func updateChoreDefinition(id: String, _ request: ChoreRequest) async throws -> ChoreDefinition {
        guard let task = chores.first(where: { $0.id == id || $0.seriesID == id }) else { throw APIError.notFound }
        let updated = try await updateChore(id: task.id, request)
        return ChoreDefinition(id: updated.seriesID ?? updated.id, chore: updated, currentTask: updated, active: true)
    }

    func deleteChoreDefinition(id: String) async throws {
        deletedDefinitions.insert(id)
    }

    func fetchChoreHistory(definitionID: String, cursor: String?) async throws -> ChoreHistoryPage {
        ChoreHistoryPage(items: chores.filter { ($0.seriesID ?? $0.id) == definitionID && $0.status != .pending && $0.status != .overdue })
    }

    func skipChore(id: String) async throws {
        guard let index = chores.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        chores[index] = chores[index].with(status: .skipped)
    }

    func fetchChores() async throws -> [Chore] { chores }

    func completeChore(id: String) async throws {
        guard let index = chores.firstIndex(where: { $0.id == id }) else { return }
        chores[index] = chores[index].completed
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

    func fetchRecipeImage(id: String) async throws -> Data {
        #if DEBUG
        if ProcessInfo.processInfo.arguments.contains("-demo-meal-card-images") {
            guard id == DemoData.chickenRecipeID else { throw APIError.offline }
            return UIGraphicsImageRenderer(size: CGSize(width: 64, height: 64)).pngData { context in
                UIColor.systemOrange.setFill()
                context.fill(CGRect(x: 0, y: 0, width: 64, height: 64))
                UIImage(systemName: "carrot.fill")?.withTintColor(.white).draw(in: CGRect(x: 12, y: 12, width: 40, height: 40))
            }
        }
        #endif
        return Data()
    }

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
        if CalendarDemoData.isEnabled { return CalendarDemoData.response(view: view, date: date) }
        return CalendarResponse(chores: chores, events: [DemoData.calendarEvent], meals: meals)
    }

    func fetchUsers() async throws -> [User] { users }

    func fetchMe() async throws -> User { users[0] }

    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User { DemoData.users[0] }

    func deleteAvatar() async throws {}

    func fetchSettings() async throws -> AppSettings {
        AppSettings(familyName: "The Demo Family")
    }

    func updateFamilyName(_ name: String) async throws {}

    func promoteUser(id: String) async throws -> User {
        try updateRole(id: id, role: "admin")
    }

    func demoteUser(id: String) async throws -> User {
        try updateRole(id: id, role: "member")
    }

    private func updateRole(id: String, role: String) throws -> User {
        guard let index = users.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let user = users[index]
        let updated = User(id: user.id, name: user.name, email: user.email, avatarURL: user.avatarURL, role: role)
        users[index] = updated
        return updated
    }

    func fetchCategories() async throws -> [ChoreCategory] { categories }

    func createCategory(name: String) async throws -> ChoreCategory {
        let category = ChoreCategory(id: "demo-cat-\(UUID().uuidString)", name: name)
        categories.append(category)
        return category
    }

    func updateCategory(id: String, name: String) async throws -> ChoreCategory {
        guard let index = categories.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let updated = ChoreCategory(id: id, name: name)
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

    // MARK: - Inventory

    func fetchInventory() async throws -> [InventoryArea] { areas }

    func createArea(_ request: AreaRequest) async throws -> InventoryArea {
        let area = InventoryArea(
            id: "demo-area-\(UUID().uuidString)",
            name: request.name,
            icon: request.icon.isEmpty ? "box" : request.icon,
            tint: request.tint.isEmpty ? "blue" : request.tint,
            items: []
        )
        areas.append(area)
        return area
    }

    func updateArea(id: String, _ request: AreaRequest) async throws -> InventoryArea {
        guard let index = areas.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let updated = InventoryArea(
            id: id,
            name: request.name,
            icon: request.icon.isEmpty ? areas[index].icon : request.icon,
            tint: request.tint.isEmpty ? areas[index].tint : request.tint,
            items: areas[index].items
        )
        areas[index] = updated
        return updated
    }

    func deleteArea(id: String) async throws {
        let identifiers = Set(areas.filter { $0.id == id }.flatMap(\.items).map(\.id))
        detachShoppingInventory(identifiers)
        areas.removeAll { $0.id == id }
    }

    func createItem(areaID: String, _ request: ItemRequest) async throws -> InventoryItem {
        guard let index = areas.firstIndex(where: { $0.id == areaID }) else { throw APIError.notFound }
        let item = InventoryItem(
            id: "demo-item-\(UUID().uuidString)",
            areaID: areaID,
            name: request.name,
            trackingMode: request.trackingMode,
            quantity: max(0, request.quantity),
            level: min(100, max(0, request.level)),
            unit: request.unit,
            lowAt: max(0, request.lowAt)
        )
        areas[index] = replacingItems(areas[index], areas[index].items + [item])
        return item
    }

    func updateItem(id: String, _ request: ItemRequest) async throws -> InventoryItem {
        guard let areaIndex = areas.firstIndex(where: { $0.items.contains(where: { $0.id == id }) }) else {
            throw APIError.notFound
        }
        let updated = InventoryItem(
            id: id,
            areaID: areas[areaIndex].id,
            name: request.name,
            trackingMode: request.trackingMode,
            quantity: max(0, request.quantity),
            level: min(100, max(0, request.level)),
            unit: request.unit,
            lowAt: max(0, request.lowAt)
        )
        let newItems = areas[areaIndex].items.map { $0.id == id ? updated : $0 }
        areas[areaIndex] = replacingItems(areas[areaIndex], newItems)
        return updated
    }

    func changeStock(id: String, change: InventoryStockChange) async throws -> InventoryItem {
        guard let item = areas.flatMap(\.items).first(where: { $0.id == id }) else { throw APIError.notFound }
        return try await updateItem(id: id, change.applying(to: item).asRequest)
    }

    func deleteItem(id: String) async throws {
        detachShoppingInventory([id])
        guard let areaIndex = areas.firstIndex(where: { $0.items.contains(where: { $0.id == id }) }) else { return }
        let newItems = areas[areaIndex].items.filter { $0.id != id }
        areas[areaIndex] = replacingItems(areas[areaIndex], newItems)
    }

    private func detachShoppingInventory(_ identifiers: Set<String>) {
        shoppingItems = shoppingItems.map { item in
            guard let linked = item.inventoryItemID, identifiers.contains(linked) else { return item }
            return ShoppingItem(id: item.id, name: item.name, purchased: item.purchased, purchaseID: item.purchaseID, restocked: item.restocked, version: item.version)
        }
    }

    private func replacingItems(_ area: InventoryArea, _ items: [InventoryItem]) -> InventoryArea {
        InventoryArea(id: area.id, name: area.name, icon: area.icon, tint: area.tint, items: items)
    }
}


extension DemoAPIClient {
    var serverIdentifier: String? { nil }
    func sendFamilyBroadcast(message: String, requestID: String) async throws -> BroadcastResult {
        BroadcastResult(accepted: notificationDevices.count, failed: 0)
    }
    func fetchChoreReminders() async throws -> ChoreReminders { choreReminders }
    func updateChoreReminders(_ settings: ChoreReminders) async throws { choreReminders = settings }
    func registerNotificationDevice(_ device: NotificationDevice) async throws { notificationDevices[device.id] = device }
    func removeNotificationDevice(id: String) async throws { notificationDevices.removeValue(forKey: id) }

    func fetchHomeInfo() async throws -> HomeInfo { homeInfo }
    func updateHomeInfo(_ home: HomeInfo) async throws { homeInfo = home }
    func fetchWeather() async throws -> HouseholdWeather {
        let now = Date()
        let hours = (0..<24).map { index -> [String: Any] in
            let timestamp = APIDate.iso.string(from: now.addingTimeInterval(Double(index) * 3600))
            let temperature = 14 - index % 6
            let code = index < 3 ? 3 : 61
            let rain = index < 3 ? 20 : 70
            return ["time": timestamp, "temperature": temperature, "code": code, "rainProbability": rain, "windSpeed": 16]
        }
        let object: [String: Any] = ["status": "ok", "fetchedAt": APIDate.iso.string(from: now), "current": hours[0], "high": 17, "low": 8, "hours": hours, "advice": ["Take a raincoat."]]
        return try JSONDecoder().decode(HouseholdWeather.self, from: JSONSerialization.data(withJSONObject: object))
    }
    func fetchDepartures() async throws -> HouseholdDepartures {
        let now = Date()
        let item: [String: Any] = ["eventID": "demo-event-1", "title": "Family Dinner", "destination": "10 High Street", "startTime": APIDate.iso.string(from: now.addingTimeInterval(2700)), "leaveAt": APIDate.iso.string(from: now.addingTimeInterval(900)), "drivingSeconds": 1200, "fetchedAt": APIDate.iso.string(from: now)]
        return try JSONDecoder().decode(HouseholdDepartures.self, from: JSONSerialization.data(withJSONObject: ["status": "ok", "items": [item], "selectedEventID": "demo-event-1"]))
    }
    func fetchShopping() async throws -> [ShoppingItem] { shoppingItems }
    func addShopping(name: String, inventoryItemID: String?) async throws -> ShoppingItem {
        if let inventoryItemID, let existing = shoppingItems.first(where: { !$0.purchased && $0.inventoryItemID == inventoryItemID }) { return existing }
        let item = ShoppingItem(id: UUID().uuidString, name: name, inventoryItemID: inventoryItemID)
        shoppingItems.append(item)
        return item
    }
    func updateShopping(id: String, change: ShoppingChange) async throws -> ShoppingItem {
        guard let index = shoppingItems.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let old = shoppingItems[index]
        guard old.version == change.version else { throw APIError.conflict }
        let purchased = change.purchased ?? old.purchased
        if !purchased, let linkedID = old.inventoryItemID, shoppingItems.contains(where: { $0.id != id && !$0.purchased && $0.inventoryItemID == linkedID }) { throw APIError.conflict }
        let changedState = purchased != old.purchased
        let item = ShoppingItem(id: id, name: change.name ?? old.name, inventoryItemID: old.inventoryItemID, purchased: purchased, purchaseID: changedState ? (purchased ? UUID().uuidString : nil) : old.purchaseID, restocked: changedState ? false : old.restocked, version: old.version + 1)
        shoppingItems[index] = item
        return item
    }
    func deleteShopping(id: String) async throws { shoppingItems.removeAll { $0.id == id } }
    func clearPurchasedShopping() async throws { shoppingItems.removeAll( where: \.purchased) }
    func restockShopping(id: String, request: ShoppingRestock) async throws -> ShoppingItem {
        guard let index = shoppingItems.firstIndex(where: { $0.id == id }) else { throw APIError.notFound }
        let old = shoppingItems[index]
        guard old.purchased, old.purchaseID == request.purchaseID else { throw APIError.conflict }
        if old.restocked { return old }
        guard let areaIndex = areas.firstIndex(where: { $0.items.contains(where: { $0.id == old.inventoryItemID }) }),
              let inventory = areas[areaIndex].items.first(where: { $0.id == old.inventoryItemID }) else { throw APIError.notFound }
        let updated: InventoryItem
        if inventory.trackingMode == .count, let quantity = request.quantity, quantity > 0, request.level == nil { updated = inventory.adjusting(by: quantity) }
        else if inventory.trackingMode == .level, let level = request.level, (0...100).contains(level), request.quantity == nil { updated = inventory.withLevel(level) }
        else { throw APIError.badRequest(serverMessage: "Enter a valid restock amount.") }
        let area = areas[areaIndex]
        areas[areaIndex] = InventoryArea(id: area.id, name: area.name, icon: area.icon, tint: area.tint, items: area.items.map { $0.id == updated.id ? updated : $0 })
        let item = ShoppingItem(id: id, name: old.name, inventoryItemID: old.inventoryItemID, purchased: true, purchaseID: old.purchaseID, restocked: true, version: old.version + 1)
        shoppingItems[index] = item
        return item
    }
}
