import Foundation

public struct CalendarSchedule {
    public let calendar: Calendar

    public init(calendar: Calendar = .current) {
        self.calendar = calendar
    }

    public func monthStart(containing date: Date) -> Date {
        calendar.dateInterval(of: .month, for: date)!.start
    }

    public func weekStart(containing date: Date) -> Date {
        calendar.dateInterval(of: .weekOfYear, for: date)!.start
    }

    public func adding(_ component: Calendar.Component, value: Int, to date: Date) -> Date {
        calendar.date(byAdding: component, value: value, to: date)!
    }

    public func monthDays(containing date: Date) -> [Date?] {
        let start = monthStart(containing: date)
        let offset = (calendar.component(.weekday, from: start) - calendar.firstWeekday + 7) % 7
        let count = calendar.range(of: .day, in: .month, for: start)!.count
        let trailing = (7 - (offset + count) % 7) % 7
        return Array(repeating: nil, count: offset)
            + (0..<count).map { adding(.day, value: $0, to: start) }
            + Array(repeating: nil, count: trailing)
    }

    public func dayKey(_ date: Date) -> String {
        var wireCalendar = Calendar(identifier: .gregorian)
        wireCalendar.timeZone = calendar.timeZone
        let components = wireCalendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", components.year!, components.month!, components.day!)
    }

    public func monthKey(_ date: Date) -> String {
        String(dayKey(date).prefix(7))
    }

    public func contains(_ event: CalendarEvent, on date: Date) -> Bool {
        let day = calendar.dateInterval(of: .day, for: date)!
        let startTime = event.allDay ? localAllDayDate(event.startTime) : event.startTime
        let endTime = event.endTime.map { event.allDay ? localAllDayDate($0) : $0 }
        if let end = endTime, end > startTime {
            return startTime < day.end && end > day.start
        }
        return startTime >= day.start && startTime < day.end
    }

    private func localAllDayDate(_ date: Date) -> Date {
        var source = Calendar(identifier: .gregorian)
        source.timeZone = TimeZone(secondsFromGMT: 0)!
        var destination = source
        destination.timeZone = calendar.timeZone
        return destination.date(from: source.dateComponents([.year, .month, .day], from: date))!
    }

    public func minuteOfDay(_ date: Date) -> Double {
        let components = calendar.dateComponents([.hour, .minute, .second], from: date)
        return Double(components.hour! * 60 + components.minute!) + Double(components.second!) / 60
    }

    public func placements(for events: [CalendarEvent], on date: Date, minimumMinutes: Double = 30) -> [CalendarPlacement] {
        let day = calendar.dateInterval(of: .day, for: date)!
        let entries = events.filter { !$0.allDay && contains($0, on: date) }
            .sorted { $0.startTime == $1.startTime ? $0.id < $1.id : $0.startTime < $1.startTime }
        var result: [CalendarPlacement] = []
        var group: [CalendarPlacement] = []
        var columnEnds: [Double] = []
        var groupEnd = 0.0

        func appendGroup() {
            result += group.map {
                CalendarPlacement(event: $0.event, startMinute: $0.startMinute, endMinute: $0.endMinute,
                                  column: $0.column, columnCount: columnEnds.count)
            }
        }

        for event in entries {
            let start = event.startTime <= day.start ? 0 : minuteOfDay(event.startTime)
            let eventEnd = event.endTime.flatMap { $0 > event.startTime ? $0 : nil }
            let end = eventEnd.map { $0 >= day.end ? 1440 : minuteOfDay($0) } ?? min(1440, start + 30)
            let visibleEnd = min(1440, max(start + max(1, minimumMinutes), end))
            if !group.isEmpty && start >= groupEnd {
                appendGroup()
                group = []
                columnEnds = []
                groupEnd = 0
            }
            let column = columnEnds.firstIndex(where: { $0 <= start }) ?? columnEnds.count
            if column == columnEnds.count { columnEnds.append(visibleEnd) }
            else { columnEnds[column] = visibleEnd }
            groupEnd = max(groupEnd, visibleEnd)
            group.append(CalendarPlacement(event: event, startMinute: start, endMinute: max(start + 1, end),
                                           column: column, columnCount: 1))
        }
        appendGroup()
        return result
    }
}

public struct CalendarPlacement: Identifiable {
    public var id: String { event.id }
    public let event: CalendarEvent
    public let startMinute: Double
    public let endMinute: Double
    public let column: Int
    public let columnCount: Int
}
