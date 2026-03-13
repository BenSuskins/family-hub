import Foundation

struct CalendarResponse: Decodable {
    let chores: [Chore]   // backend guarantees [] not null (nil guard in handler)
}
