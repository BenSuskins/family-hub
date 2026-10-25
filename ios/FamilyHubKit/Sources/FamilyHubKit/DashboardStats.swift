import Foundation

public struct BinCollection: Codable, Identifiable, Equatable {
    public let date: String
    public let type: String
    public let name: String

    public var id: String { "\(date)-\(type)" }

    public var imageName: String {
        switch type {
        case "food_caddy": "BinFoodCaddy"
        case "garden": "BinGarden"
        case "mixed_recycling": "BinMixedRecycling"
        case "paper_card": "BinPaperCard"
        default: "BinGrey"
        }
    }

    public init(date: String, type: String, name: String) {
        self.date = date
        self.type = type
        self.name = name
    }
}

public struct DashboardStats: Codable {
    public let choresDueToday: Int
    public let choresOverdue: Int
    public let choresDueTodayList: [Chore]
    public let choresOverdueList: [Chore]
    public let mealsThisWeek: Int
    public let todayMeals: [MealPlan]
    public let binCollectionsTomorrow: [BinCollection]

    public init(
        choresDueToday: Int,
        choresOverdue: Int,
        choresDueTodayList: [Chore],
        choresOverdueList: [Chore],
        mealsThisWeek: Int,
        todayMeals: [MealPlan],
        binCollectionsTomorrow: [BinCollection] = []
    ) {
        self.choresDueToday = choresDueToday
        self.choresOverdue = choresOverdue
        self.choresDueTodayList = choresDueTodayList
        self.choresOverdueList = choresOverdueList
        self.mealsThisWeek = mealsThisWeek
        self.todayMeals = todayMeals
        self.binCollectionsTomorrow = binCollectionsTomorrow
    }

    enum CodingKeys: String, CodingKey {
        case choresDueToday = "chores_due_today"
        case choresOverdue = "chores_overdue"
        case choresDueTodayList = "chores_due_today_list"
        case choresOverdueList = "chores_overdue_list"
        case mealsThisWeek = "meals_this_week"
        case todayMeals = "today_meals"
        case binCollectionsTomorrow = "bin_collections_tomorrow"
    }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        choresDueToday = try values.decode(Int.self, forKey: .choresDueToday)
        choresOverdue = try values.decode(Int.self, forKey: .choresOverdue)
        choresDueTodayList = try values.decode([Chore].self, forKey: .choresDueTodayList)
        choresOverdueList = try values.decode([Chore].self, forKey: .choresOverdueList)
        mealsThisWeek = try values.decode(Int.self, forKey: .mealsThisWeek)
        todayMeals = try values.decode([MealPlan].self, forKey: .todayMeals)
        binCollectionsTomorrow = try values.decodeIfPresent([BinCollection].self, forKey: .binCollectionsTomorrow) ?? []
    }
}
