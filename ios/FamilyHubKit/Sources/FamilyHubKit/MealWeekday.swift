import Foundation

public enum MealWeekday: String, Codable, CaseIterable, Sendable {
    case monday, tuesday, wednesday, thursday, friday, saturday, sunday

    public var calendarWeekday: Int {
        switch self {
        case .sunday: 1
        case .monday: 2
        case .tuesday: 3
        case .wednesday: 4
        case .thursday: 5
        case .friday: 6
        case .saturday: 7
        }
    }

    public var title: String {
        Calendar.current.weekdaySymbols[calendarWeekday - 1]
    }

    public var rangeTitle: String {
        let endIndex = (calendarWeekday + 5) % 7
        return "\(title)–\(Calendar.current.weekdaySymbols[endIndex])"
    }

    public func weekStart(containing date: Date, calendar: Calendar = Calendar(identifier: .gregorian)) -> Date {
        let offset = (calendar.component(.weekday, from: date) - calendarWeekday + 7) % 7
        return calendar.date(byAdding: .day, value: -offset, to: calendar.startOfDay(for: date))!
    }
}
