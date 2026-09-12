import Foundation

public struct DashboardStats: Decodable {
    public let choresDueToday: Int
    public let choresOverdue: Int
    public let choresDueTodayList: [Chore]
    public let choresOverdueList: [Chore]
    public let mealsThisWeek: Int
    public let todayMeals: [MealPlan]

    public init(
        choresDueToday: Int,
        choresOverdue: Int,
        choresDueTodayList: [Chore],
        choresOverdueList: [Chore],
        mealsThisWeek: Int,
        todayMeals: [MealPlan]
    ) {
        self.choresDueToday = choresDueToday
        self.choresOverdue = choresOverdue
        self.choresDueTodayList = choresDueTodayList
        self.choresOverdueList = choresOverdueList
        self.mealsThisWeek = mealsThisWeek
        self.todayMeals = todayMeals
    }

    enum CodingKeys: String, CodingKey {
        case choresDueToday = "chores_due_today"
        case choresOverdue = "chores_overdue"
        case choresDueTodayList = "chores_due_today_list"
        case choresOverdueList = "chores_overdue_list"
        case mealsThisWeek = "meals_this_week"
        case todayMeals = "today_meals"
    }
}
