import Foundation

public struct CalendarResponse: Decodable {
    public let chores: [Chore]
    public let events: [CalendarEvent]
    public let meals: [MealPlan]

    public init(chores: [Chore], events: [CalendarEvent], meals: [MealPlan]) {
        self.chores = chores
        self.events = events
        self.meals = meals
    }
}

public struct CalendarEvent: Codable, Identifiable {
    public let id: String
    public let title: String
    public let description: String
    public let location: String
    public let startTime: Date
    public let endTime: Date?
    public let allDay: Bool
    public let color: String

    public init(
        id: String,
        title: String,
        description: String,
        location: String,
        startTime: Date,
        endTime: Date?,
        allDay: Bool,
        color: String
    ) {
        self.id = id
        self.title = title
        self.description = description
        self.location = location
        self.startTime = startTime
        self.endTime = endTime
        self.allDay = allDay
        self.color = color
    }

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
