import Foundation
import FamilyHubKit

enum CalendarDemoData {
    static var isEnabled: Bool { ProcessInfo.processInfo.arguments.contains("-calendar-fixtures") }
    static var now: Date { date(day: 15, hour: 8) }

    static func response(view: String, date: Date) -> CalendarResponse {
        let schedule = CalendarSchedule()
        let calendar = schedule.calendar
        let interval: DateInterval
        switch view {
        case "day": interval = calendar.dateInterval(of: .day, for: date)!
        case "week": interval = calendar.dateInterval(of: .weekOfYear, for: date)!
        default: interval = calendar.dateInterval(of: .month, for: date)!
        }
        let days = calendar.range(of: .day, in: .month, for: interval.start)!.count
        let dates = (0...days).map { schedule.adding(.day, value: $0, to: interval.start) }.filter { $0 < interval.end }
        let events = [
            event("school", "School run", day: 15, hour: 9, duration: 60, color: "#007AFF"),
            event("dentist", "Dentist", day: 15, hour: 9, minute: 30, duration: 90, color: "#AF52DE"),
            event("lunch", "Lunch with Sam", day: 15, hour: 12, duration: 60, color: "#FF9500"),
            CalendarEvent(id: "fixture-trip", title: "Family trip", description: "A few days together.", location: "",
                          startTime: APIDate.iso.date(from: "2026-03-15T00:00:00Z")!,
                          endTime: APIDate.iso.date(from: "2026-03-17T00:00:00Z"), allDay: true, color: "#34C759"),
            CalendarEvent(id: "fixture-month-boundary", title: "Weekend away", description: "Across the month boundary.", location: "",
                          startTime: APIDate.iso.date(from: "2026-02-28T00:00:00Z")!,
                          endTime: APIDate.iso.date(from: "2026-03-02T00:00:00Z"), allDay: true, color: "#FF2D55"),
            CalendarEvent(id: "fixture-no-end", title: "Call Alex", description: "", location: "",
                          startTime: self.date(day: 17, hour: 10), endTime: nil, allDay: false, color: "#007AFF")
        ] + (0..<6).map { index in
            event("busy-\(index)", "Family activity \(index + 1)", day: 16, hour: 9 + index, duration: 45, color: "#5856D6")
        }
        let chores = [
            Chore(id: "fixture-dishes", name: "Do the dishes", description: "", status: .pending,
                  dueDate: "2026-03-15T00:00:00Z", assignedToUserID: nil),
            Chore(id: "fixture-bins", name: "Take out the bins", description: "", status: .pending,
                  dueDate: "2026-03-17T00:00:00Z", assignedToUserID: nil)
        ]
        let meals = ["breakfast", "lunch", "dinner"].map {
            MealPlan(date: "2026-03-15", mealType: $0, name: $0 == "dinner" ? "Family pasta" : $0.capitalized,
                     notes: "", recipeID: nil)
        }
        let keys = Set(dates.map { schedule.dayKey($0) })
        return CalendarResponse(
            chores: chores.filter { keys.contains(String(($0.dueDate ?? "").prefix(10))) },
            events: events.filter { event in dates.contains { schedule.contains(event, on: $0) } },
            meals: meals.filter { keys.contains($0.date) }
        )
    }

    private static func date(day: Int, hour: Int, minute: Int = 0) -> Date {
        Calendar.current.date(from: DateComponents(year: 2026, month: 3, day: day, hour: hour, minute: minute))!
    }

    private static func event(_ identifier: String, _ title: String, day: Int, hour: Int, minute: Int = 0,
                              duration: Int, color: String) -> CalendarEvent {
        let start = date(day: day, hour: hour, minute: minute)
        return CalendarEvent(id: "fixture-\(identifier)", title: title, description: "Calendar preview", location: "",
                             startTime: start, endTime: start.addingTimeInterval(Double(duration) * 60), allDay: false, color: color)
    }
}
