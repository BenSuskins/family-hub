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

    /// Whether the entry carries a day worth drawing.
    var hasData: Bool {
        switch status {
        case .ok, .cached:             return true
        case .signedOut, .unavailable: return false
        }
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
        TodayEntry(
            date: Date(),
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
                        timeLabel: "17:30",
                        colorHex: "3B82F6"
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
                        timeLabel: "20:00",
                        colorHex: "F97316"
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
                capturedAt: Date()
            ),
            status: .ok
        )
    }
}
