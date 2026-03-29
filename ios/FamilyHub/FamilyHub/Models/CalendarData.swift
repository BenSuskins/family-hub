import Foundation

struct CalendarResponse: Decodable {
    let chores: [Chore]
    let events: [CalendarEvent]
    let meals: [MealPlan]
}

struct CalendarEvent: Codable, Identifiable {
    let id: String
    let title: String
    let description: String
    let location: String
    let startTime: Date
    let endTime: Date?
    let allDay: Bool
    let color: String

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case title = "Title"
        case description = "Description"
        case location = "Location"
        case startTime = "StartTime"
        case endTime = "EndTime"
        case allDay = "AllDay"
        case color = "Color"
    }
}
