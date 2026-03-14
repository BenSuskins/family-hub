import Foundation

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
