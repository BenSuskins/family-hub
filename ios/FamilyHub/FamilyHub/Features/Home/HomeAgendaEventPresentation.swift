import Foundation
import FamilyHubKit

enum HomeAgendaEventPresentation {
    static func isPast(event: CalendarEvent, at date: Date) -> Bool {
        guard let endTime = event.endTime else { return false }
        return endTime <= date
    }
}
