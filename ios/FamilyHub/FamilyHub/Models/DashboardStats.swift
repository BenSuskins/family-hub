import Foundation

struct DashboardStats: Decodable {
    let choresDueToday: Int
    let choresOverdue: Int
    let choresDueTodayList: [Chore]
    let choresOverdueList: [Chore]

    enum CodingKeys: String, CodingKey {
        case choresDueToday = "chores_due_today"
        case choresOverdue = "chores_overdue"
        case choresDueTodayList = "chores_due_today_list"
        case choresOverdueList = "chores_overdue_list"
    }
}
