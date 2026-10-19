import Foundation

public enum SiriActionError: Error, LocalizedError {
    case invalid(String)
    case uncertain

    public var errorDescription: String? {
        switch self {
        case .invalid(let message): return message
        case .uncertain: return "The server did not confirm the change. Check Family Hub before you repeat it."
        }
    }
}

public struct SiriDay {
    public let key: String
    public let spoken: String
    public let queryDate: Date
}

public struct SiriDateRules {
    private let calendar: Calendar

    public init(timezone: String) throws {
        guard let zone = TimeZone(identifier: timezone) else {
            throw SiriActionError.invalid("Set a valid household timezone in Family Hub.")
        }
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = zone
        self.calendar = calendar
    }

    public func day(_ components: DateComponents?, now: Date) throws -> SiriDay {
        let selected = components ?? calendar.dateComponents([.year, .month, .day], from: now)
        guard let year = selected.year, let month = selected.month, let day = selected.day,
              let date = calendar.date(from: DateComponents(year: year, month: month, day: day, hour: 12)),
              calendar.component(.year, from: date) == year,
              calendar.component(.month, from: date) == month,
              calendar.component(.day, from: date) == day else {
            throw SiriActionError.invalid("Choose a valid calendar date.")
        }
        let formatter = DateFormatter()
        formatter.calendar = calendar
        formatter.timeZone = calendar.timeZone
        formatter.locale = Locale(identifier: "en_GB")
        formatter.dateFormat = "EEEE, d MMMM yyyy"
        // The existing meal client encodes dates in the device timezone.
        var queryCalendar = Calendar(identifier: .gregorian)
        queryCalendar.timeZone = APIDate.day.timeZone
        guard let queryDate = queryCalendar.date(from: DateComponents(year: year, month: month, day: day, hour: 12)) else {
            throw SiriActionError.invalid("Choose a valid calendar date.")
        }
        return SiriDay(key: String(format: "%04d-%02d-%02d", year, month, day), spoken: formatter.string(from: date), queryDate: queryDate)
    }

    public func eligibleChores(_ chores: [Chore], userID: String, now: Date) -> [Chore] {
        guard let today = try? day(nil, now: now).key else { return [] }
        return chores.filter { chore in
            guard chore.assignedToUserID == userID, !chore.isForecast,
                  chore.status == .pending || chore.status == .overdue,
                  let due = chore.dueDate, !due.isEmpty else { return false }
            if chore.status == .overdue { return true }
            // The server stores the due day as a timestamp and compares its calendar components.
            let key = String(due.prefix(10))
            let parts = key.split(separator: "-").compactMap { Int($0) }
            guard key.count == 10, parts.count == 3,
                  let validated = try? day(DateComponents(year: parts[0], month: parts[1], day: parts[2]), now: now),
                  validated.key == key else { return false }
            return key <= today
        }.sorted {
            if $0.dueDate != $1.dueDate { return ($0.dueDate ?? "") < ($1.dueDate ?? "") }
            return $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending
        }
    }
}

public enum SiriNames {
    public static func normalized(_ name: String) -> String {
        name.split(whereSeparator: \.isWhitespace).joined(separator: " ").lowercased()
    }

    public static func recipes(named name: String, in recipes: [Recipe]) -> [Recipe] {
        recipes.filter { normalized($0.title) == normalized(name) }
    }

    public static func summary(_ lines: [String], empty: String) -> String {
        guard !lines.isEmpty else { return empty }
        let spoken = lines.prefix(5).joined(separator: ". ") + "."
        return lines.count > 5 ? "There are \(lines.count) items. The first five are: \(spoken)" : spoken
    }
}

public enum InventoryStockChange: Encodable {
    case quantity(Int)
    case quantityDelta(Int)
    case level(Int)

    private enum CodingKeys: String, CodingKey { case quantity, quantityDelta, level }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case .quantity(let value): try container.encode(value, forKey: .quantity)
        case .quantityDelta(let value): try container.encode(value, forKey: .quantityDelta)
        case .level(let value): try container.encode(value, forKey: .level)
        }
    }

    public func applying(to item: InventoryItem) throws -> InventoryItem {
        switch self {
        case .quantity(let value):
            guard item.trackingMode == .count, value >= 0 else { throw SiriActionError.invalid("Use a nonnegative whole count for this item.") }
            return InventoryItem(id: item.id, areaID: item.areaID, name: item.name, trackingMode: item.trackingMode, quantity: value, level: item.level, unit: item.unit, lowAt: item.lowAt)
        case .quantityDelta(let value):
            guard item.trackingMode == .count else { throw SiriActionError.invalid("This item uses a percentage, not a count.") }
            let (total, overflow) = item.quantity.addingReportingOverflow(value)
            guard !overflow else { throw SiriActionError.invalid("The count is too large.") }
            return try InventoryStockChange.quantity(max(0, total)).applying(to: item)
        case .level(let value):
            guard item.trackingMode == .level, (0...100).contains(value) else { throw SiriActionError.invalid("Use a percentage from 0 to 100 for this item.") }
            return item.withLevel(value)
        }
    }
}
