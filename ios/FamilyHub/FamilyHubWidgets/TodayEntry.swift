import Foundation
import WidgetKit
import FamilyHubKit

/// One rendering of the Today widget.
struct TodayEntry: TimelineEntry {
    enum Status: Equatable {
        /// Real data, either fresh from the server or recently published by
        /// the app.
        case ok
        /// Real data, but older than the refresh window: the server could not
        /// be reached this time.
        case cached
        /// No server URL or API token in the app group — the app has not been
        /// set up, or someone signed out.
        case signedOut
        /// The server is unreachable and there is nothing cached to fall back
        /// on. Distinct from `ok` with an empty day: "we don't know" must never
        /// be drawn as "nothing on".
        case unavailable
    }

    let date: Date
    let data: TodayWidgetData
    let status: Status
    let failedChoreID: String?
    let pendingChoreID: String?

    init(date: Date, data: TodayWidgetData, status: Status, failedChoreID: String? = nil, pendingChoreID: String? = nil) {
        self.date = date
        self.data = data
        self.status = status
        self.failedChoreID = failedChoreID
        self.pendingChoreID = pendingChoreID
    }

    /// Whether the entry carries a day worth drawing.
    var hasData: Bool {
        switch status {
        case .ok, .cached:             return true
        case .signedOut, .unavailable: return false
        }
    }

    var nextEvent: TodayWidgetData.Item? {
        hasData ? data.nextTimedEvent(after: date) : nil
    }

    var chores: [TodayWidgetData.Item] {
        hasData ? data.chores : []
    }

    /// When to ask WidgetKit to come back.
    ///
    /// The budget is only a few dozen refreshes a day, so idle and broken
    /// states back off hard. What actually keeps the widget feeling live is the
    /// app reloading timelines directly whenever a chore changes; these
    /// intervals are the safety net for when the app isn't opened at all.
    var nextRefresh: Date {
        let minutes: Double
        switch status {
        case .ok where !data.isClear: minutes = 30
        case .ok:                     minutes = 120
        case .cached, .unavailable:   minutes = 60
        case .signedOut:              minutes = 240
        }
        return date.addingTimeInterval(minutes * 60)
    }

    /// Sample data for the widget gallery and Xcode previews. The gallery must
    /// not depend on the network, or on whether this phone happens to be
    /// signed in.
    static var preview: TodayEntry {
        let now = Date()
        let swimmingStart = now.addingTimeInterval(60 * 60)
        let bookStart = now.addingTimeInterval(3 * 60 * 60)
        return TodayEntry(
            date: now,
            data: TodayWidgetData(
                items: [
                    .init(
                        id: "chore-1",
                        kind: .chore,
                        title: "Put the bins out",
                        subtitle: "Ben · Sep 15",
                        timeLabel: "Overdue",
                        isOverdue: true,
                        choreID: "1",
                        initials: "B"
                    ),
                    .init(
                        id: "event-1",
                        kind: .event,
                        title: "Swimming lesson",
                        subtitle: "Leisure centre",
                        timeLabel: swimmingStart.formatted(date: .omitted, time: .shortened),
                        colorHex: "3B82F6",
                        eventStart: swimmingStart,
                        eventEnd: swimmingStart.addingTimeInterval(60 * 60),
                        isAllDay: false
                    ),
                    .init(
                        id: "chore-2",
                        kind: .chore,
                        title: "Hoover the front room",
                        subtitle: "Alex",
                        timeLabel: "18:00",
                        choreID: "2",
                        initials: "A"
                    ),
                    .init(
                        id: "event-2",
                        kind: .event,
                        title: "Book club",
                        subtitle: "The Crown",
                        timeLabel: bookStart.formatted(date: .omitted, time: .shortened),
                        colorHex: "F97316",
                        eventStart: bookStart,
                        eventEnd: bookStart.addingTimeInterval(60 * 60),
                        isAllDay: false
                    ),
                    .init(
                        id: "chore-3",
                        kind: .chore,
                        title: "Load the dishwasher",
                        subtitle: "Sam",
                        choreID: "3",
                        initials: "S"
                    ),
                ],
                totalItemCount: 5,
                overdueCount: 1,
                dueTodayCount: 2,
                eventCount: 2,
                meals: [
                    .init(slot: .lunch, name: "Jacket potatoes"),
                    .init(slot: .dinner, name: "Spaghetti bolognese"),
                ],
                capturedAt: now
            ),
            status: .ok
        )
    }
}
