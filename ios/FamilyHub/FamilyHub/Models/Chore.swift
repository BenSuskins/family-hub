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

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case name = "Name"
        case description = "Description"
        case status = "Status"
        case dueDate = "DueDate"
        case assignedToUserID = "AssignedToUserID"
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

extension Chore {
    var badge: ChoreBadge? {
        switch status {
        case .overdue:   return .overdue
        case .completed: return nil
        case .pending:
            guard let dueDate else { return .dueSoon }
            let date = ISO8601DateFormatter().date(from: dueDate)
                ?? parseShortDate(dueDate)
            guard let date else { return .dueSoon }
            return Calendar.current.isDateInToday(date) ? .dueToday : .dueSoon
        }
    }

    private func parseShortDate(_ string: String) -> Date? {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        return formatter.date(from: String(string.prefix(10)))
    }
}

extension Chore {
    var formattedDueDate: String? {
        guard let dueDate else { return nil }
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        let iso = ISO8601DateFormatter()
        guard let date = iso.date(from: dueDate) ?? formatter.date(from: String(dueDate.prefix(10))) else {
            return nil
        }
        let display = DateFormatter()
        display.dateFormat = "MMM d"
        return display.string(from: date)
    }
}
