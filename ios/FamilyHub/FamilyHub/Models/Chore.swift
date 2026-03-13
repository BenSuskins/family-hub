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
