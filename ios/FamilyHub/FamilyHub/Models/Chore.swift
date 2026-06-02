import Foundation
import SwiftUI

enum ChoreStatus: String, Codable {
    case pending = "pending"
    case completed = "completed"
    case overdue = "overdue"
}

struct Chore: Codable, Identifiable {
    let id: String
    let name: String
    let description: String
    let status: ChoreStatus
    let dueDate: String?       // RFC3339 timestamp or nil
    let assignedToUserID: String?

    // Series / scheduling detail (mirrors the web chore form options).
    let categoryID: String?
    let dueTime: String?           // "HH:mm" or nil
    let eligibleAssignees: [String]
    let recurrenceType: String     // none/daily/weekly/monthly/custom
    let recurrenceValue: String    // JSON config: {interval, unit, days, day_of_month}
    let recurOnComplete: Bool
    let seriesID: String?
    let recurrenceUntil: String?   // "YYYY-MM-DD" or nil
    let recurrenceCount: Int?

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case name = "Name"
        case description = "Description"
        case status = "Status"
        case dueDate = "DueDate"
        case assignedToUserID = "AssignedToUserID"
        case categoryID = "CategoryID"
        case dueTime = "DueTime"
        case eligibleAssignees = "EligibleAssignees"
        case recurrenceType = "RecurrenceType"
        case recurrenceValue = "RecurrenceValue"
        case recurOnComplete = "RecurOnComplete"
        case seriesID = "SeriesID"
        case recurrenceUntil = "RecurrenceUntil"
        case recurrenceCount = "RecurrenceCount"
    }

    init(
        id: String,
        name: String,
        description: String,
        status: ChoreStatus,
        dueDate: String? = nil,
        assignedToUserID: String? = nil,
        categoryID: String? = nil,
        dueTime: String? = nil,
        eligibleAssignees: [String] = [],
        recurrenceType: String = "none",
        recurrenceValue: String = "",
        recurOnComplete: Bool = false,
        seriesID: String? = nil,
        recurrenceUntil: String? = nil,
        recurrenceCount: Int? = nil
    ) {
        self.id = id
        self.name = name
        self.description = description
        self.status = status
        self.dueDate = dueDate
        self.assignedToUserID = assignedToUserID
        self.categoryID = categoryID
        self.dueTime = dueTime
        self.eligibleAssignees = eligibleAssignees
        self.recurrenceType = recurrenceType
        self.recurrenceValue = recurrenceValue
        self.recurOnComplete = recurOnComplete
        self.seriesID = seriesID
        self.recurrenceUntil = recurrenceUntil
        self.recurrenceCount = recurrenceCount
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        name = try c.decode(String.self, forKey: .name)
        description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        status = try c.decode(ChoreStatus.self, forKey: .status)
        dueDate = try c.decodeIfPresent(String.self, forKey: .dueDate)
        assignedToUserID = try c.decodeIfPresent(String.self, forKey: .assignedToUserID)
        categoryID = try c.decodeIfPresent(String.self, forKey: .categoryID)
        dueTime = try c.decodeIfPresent(String.self, forKey: .dueTime)
        eligibleAssignees = try c.decodeIfPresent([String].self, forKey: .eligibleAssignees) ?? []
        recurrenceType = try c.decodeIfPresent(String.self, forKey: .recurrenceType) ?? "none"
        recurrenceValue = try c.decodeIfPresent(String.self, forKey: .recurrenceValue) ?? ""
        recurOnComplete = try c.decodeIfPresent(Bool.self, forKey: .recurOnComplete) ?? false
        seriesID = try c.decodeIfPresent(String.self, forKey: .seriesID)
        recurrenceUntil = try c.decodeIfPresent(String.self, forKey: .recurrenceUntil)
        recurrenceCount = try c.decodeIfPresent(Int.self, forKey: .recurrenceCount)
    }
}

enum ChoreBadge: Equatable {
    case overdue
    case dueToday
    case dueSoon

    var label: String {
        switch self {
        case .overdue:  return "Overdue"
        case .dueToday: return "Today"
        case .dueSoon:  return "Due Soon"
        }
    }

    var color: Color {
        switch self {
        case .overdue:            return .red
        case .dueToday, .dueSoon: return .orange
        }
    }
}

struct ChoreRequest: Encodable {
    var name: String
    var description: String
    var assignees: [String]
    var dueDate: String?
    var recurrenceType: String?

    // Full-parity options (match the JSON tags on the server choreAPIBody).
    var categoryId: String?
    var dueTime: String?
    var recurrenceInterval: Int?
    var recurrenceDays: [String]?
    var recurrenceDayOfMonth: Int?
    var recurrenceUnit: String?
    var recurrenceUntil: String?
    var recurrenceCount: Int?
    var recurOnComplete: Bool

    init(
        name: String,
        description: String,
        assignees: [String],
        dueDate: String? = nil,
        recurrenceType: String? = nil,
        categoryId: String? = nil,
        dueTime: String? = nil,
        recurrenceInterval: Int? = nil,
        recurrenceDays: [String]? = nil,
        recurrenceDayOfMonth: Int? = nil,
        recurrenceUnit: String? = nil,
        recurrenceUntil: String? = nil,
        recurrenceCount: Int? = nil,
        recurOnComplete: Bool = false
    ) {
        self.name = name
        self.description = description
        self.assignees = assignees
        self.dueDate = dueDate
        self.recurrenceType = recurrenceType
        self.categoryId = categoryId
        self.dueTime = dueTime
        self.recurrenceInterval = recurrenceInterval
        self.recurrenceDays = recurrenceDays
        self.recurrenceDayOfMonth = recurrenceDayOfMonth
        self.recurrenceUnit = recurrenceUnit
        self.recurrenceUntil = recurrenceUntil
        self.recurrenceCount = recurrenceCount
        self.recurOnComplete = recurOnComplete
    }
}

/// Decoded form of a chore's `recurrenceValue` JSON config, used to pre-fill the
/// edit form. Mirrors the server's `recurrenceConfigJSON`.
struct RecurrenceConfig {
    var interval: Int = 1
    var unit: String = "days"
    var days: [String] = []
    var dayOfMonth: Int = 1

    init() {}

    init(json: String) {
        guard let data = json.data(using: .utf8),
              let obj = try? JSONDecoder().decode(Raw.self, from: data) else { return }
        if let i = obj.interval, i > 0 { interval = i }
        if let u = obj.unit, !u.isEmpty { unit = u }
        if let d = obj.days { days = d }
        if let dom = obj.dayOfMonth, dom >= 1, dom <= 31 { dayOfMonth = dom }
    }

    private struct Raw: Decodable {
        let interval: Int?
        let unit: String?
        let days: [String]?
        let dayOfMonth: Int?
        enum CodingKeys: String, CodingKey {
            case interval, unit, days
            case dayOfMonth = "day_of_month"
        }
    }
}

extension Chore {
    var badge: ChoreBadge? {
        switch status {
        case .overdue:   return .overdue
        case .completed: return nil
        case .pending:
            guard let date = APIDate.parse(dueDate) else { return .dueSoon }
            return Calendar.current.isDateInToday(date) ? .dueToday : .dueSoon
        }
    }

    /// Returns a copy with a different status, preserving every other field.
    /// Avoids re-listing all stored properties at each call site.
    func with(status: ChoreStatus) -> Chore {
        Chore(
            id: id,
            name: name,
            description: description,
            status: status,
            dueDate: dueDate,
            assignedToUserID: assignedToUserID,
            categoryID: categoryID,
            dueTime: dueTime,
            eligibleAssignees: eligibleAssignees,
            recurrenceType: recurrenceType,
            recurrenceValue: recurrenceValue,
            recurOnComplete: recurOnComplete,
            seriesID: seriesID,
            recurrenceUntil: recurrenceUntil,
            recurrenceCount: recurrenceCount
        )
    }

    var completed: Chore { with(status: .completed) }
}

extension Chore {
    private static let dueDateDisplayFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "MMM d"
        return f
    }()

    var formattedDueDate: String? {
        guard let date = APIDate.parse(dueDate) else { return nil }
        return Self.dueDateDisplayFormatter.string(from: date)
    }
}

extension Chore {
    var isRecurring: Bool { recurrenceType != "none" && !recurrenceType.isEmpty }

    var recurrenceConfig: RecurrenceConfig {
        RecurrenceConfig(json: recurrenceValue)
    }

    /// Short human-readable recurrence description for list rows,
    /// e.g. "Weekly · Mon, Wed" or "One-time".
    var recurrenceSummary: String {
        guard isRecurring else { return "One-time" }
        let config = recurrenceConfig
        let interval = max(config.interval, 1)

        func plural(_ unit: String) -> String {
            interval == 1 ? unit.capitalized : "Every \(interval) \(unit)s"
        }

        switch recurrenceType {
        case "daily":
            return interval == 1 ? "Daily" : "Every \(interval) days"
        case "weekly":
            let base = interval == 1 ? "Weekly" : "Every \(interval) weeks"
            guard !config.days.isEmpty else { return base }
            let labels = config.days.compactMap { Self.dayShortLabels[$0.lowercased()] }
            return labels.isEmpty ? base : "\(base) · \(labels.joined(separator: ", "))"
        case "monthly":
            let base = interval == 1 ? "Monthly" : "Every \(interval) months"
            return config.dayOfMonth >= 1 ? "\(base) · day \(config.dayOfMonth)" : base
        case "custom":
            return "Every \(interval) \(config.unit)"
        default:
            return plural(recurrenceType)
        }
    }

    static let weekdayKeys = ["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"]

    static let dayShortLabels: [String: String] = [
        "monday": "Mon", "tuesday": "Tue", "wednesday": "Wed", "thursday": "Thu",
        "friday": "Fri", "saturday": "Sat", "sunday": "Sun",
    ]
}
