import Foundation

/// Presentation-ready view of "what still needs doing today", written by the
/// app and rendered by the widget extension.
///
/// The selection, ordering and labelling live here rather than in the widget's
/// SwiftUI views for two reasons. A widget process gets a very small time and
/// memory budget to produce a view, so it should draw a prepared list rather
/// than join chores against users itself. And this layer is the one the Linux
/// CI job can test in seconds — logic buried in a `View` needs a simulator.
///
/// **The counts are of outstanding work only.** `GET /api/dashboard` reports
/// *pending* chores due today plus overdue ones; the server's `FindDueToday`
/// filters on `status = 'pending'`, so nothing in the payload says how many
/// were completed today. The widget therefore says "4 to do" rather than
/// inventing a "3 of 7 done" denominator it has no way to source.
public struct TodayWidgetData: Codable, Equatable {

    /// One chore as the widget draws it, flattened so no lookups happen at
    /// render time.
    public struct Item: Codable, Equatable, Identifiable {
        public let id: String
        public let name: String
        public let assigneeName: String?
        /// Up to two initials, drawn in a tinted circle. Widgets deliberately
        /// don't fetch avatar images: that would mean a network round-trip and
        /// an image cache inside a process with a few milliseconds to render.
        public let assigneeInitials: String?
        public let isOverdue: Bool
        /// Secondary line: the due date when overdue, otherwise the due time
        /// ("18:30") if the chore has one.
        public let detail: String?

        public init(
            id: String,
            name: String,
            assigneeName: String? = nil,
            assigneeInitials: String? = nil,
            isOverdue: Bool = false,
            detail: String? = nil
        ) {
            self.id = id
            self.name = name
            self.assigneeName = assigneeName
            self.assigneeInitials = assigneeInitials
            self.isOverdue = isOverdue
            self.detail = detail
        }

        public init(chore: Chore, user: User?, isOverdue: Bool) {
            self.init(
                id: chore.id,
                name: chore.name,
                assigneeName: user?.name,
                assigneeInitials: user.map(\.initials),
                isOverdue: isOverdue,
                detail: isOverdue ? chore.formattedDueDate : Self.time(chore.dueTime)
            )
        }

        private static func time(_ value: String?) -> String? {
            guard let value, !value.isEmpty else { return nil }
            return value
        }
    }

    /// Overdue chores first (oldest first), then chores due today, capped at
    /// ``maxItems``.
    public let items: [Item]
    /// Total overdue, which may exceed the number carried in ``items``.
    public let overdueCount: Int
    /// Total pending and due today, likewise uncapped.
    public let dueTodayCount: Int
    /// When this was built, so the widget can say how stale it is.
    public let capturedAt: Date

    public var totalCount: Int { overdueCount + dueTodayCount }
    public var isClear: Bool { totalCount == 0 }
    /// Chores the widget counted but had no room to list.
    public var hiddenCount: Int { max(0, totalCount - items.count) }

    public init(items: [Item], overdueCount: Int, dueTodayCount: Int, capturedAt: Date) {
        self.items = items
        self.overdueCount = overdueCount
        self.dueTodayCount = dueTodayCount
        self.capturedAt = capturedAt
    }

    /// Nothing outstanding and never refreshed — the placeholder a widget shows
    /// before it has any real data.
    public static func blank(at date: Date = Date()) -> TodayWidgetData {
        TodayWidgetData(items: [], overdueCount: 0, dueTodayCount: 0, capturedAt: date)
    }

    /// How many chores a snapshot carries. The largest widget family lists far
    /// fewer; the cap keeps the app-group payload small, since `UserDefaults`
    /// is a poor home for unbounded data.
    public static let maxItems = 8

    /// Build from a dashboard payload and the family's users.
    ///
    /// Overdue chores lead because they are the ones that need acting on, and
    /// within each group the earliest due date comes first.
    public static func from(
        stats: DashboardStats,
        users: [String: User],
        capturedAt: Date = Date()
    ) -> TodayWidgetData {
        let overdue = stats.choresOverdueList.sorted(by: earliestFirst)
        let dueToday = stats.choresDueTodayList.sorted(by: earliestFirst)

        func item(_ chore: Chore, isOverdue: Bool) -> Item {
            Item(chore: chore, user: chore.assignedToUserID.flatMap { users[$0] }, isOverdue: isOverdue)
        }

        let ordered = overdue.map { item($0, isOverdue: true) }
            + dueToday.map { item($0, isOverdue: false) }

        return TodayWidgetData(
            items: Array(ordered.prefix(maxItems)),
            overdueCount: overdue.count,
            dueTodayCount: dueToday.count,
            capturedAt: capturedAt
        )
    }

    /// A copy without `choreID`, with the matching count decremented.
    ///
    /// Used for the optimistic redraw after a widget tick-off: the server call
    /// has succeeded, but re-fetching the dashboard before drawing would make
    /// the tick feel slow. Unknown ids are returned unchanged, so a duplicate
    /// tap can't drive a count negative.
    public func removing(choreID: String) -> TodayWidgetData {
        guard let removed = items.first(where: { $0.id == choreID }) else { return self }
        return TodayWidgetData(
            items: items.filter { $0.id != choreID },
            overdueCount: removed.isOverdue ? max(0, overdueCount - 1) : overdueCount,
            dueTodayCount: removed.isOverdue ? dueTodayCount : max(0, dueTodayCount - 1),
            capturedAt: capturedAt
        )
    }

    /// A copy stamped as long stale, so the next timeline pass refetches
    /// instead of trusting it.
    ///
    /// Used after a change the app has not recomputed a whole snapshot for —
    /// adding, editing or deleting a chore. Discarding the snapshot outright
    /// would work too, but this keeps real data around as a fallback for when
    /// the refetch fails because the hub is out of reach.
    public func markedStale() -> TodayWidgetData {
        TodayWidgetData(
            items: items,
            overdueCount: overdueCount,
            dueTodayCount: dueTodayCount,
            capturedAt: .distantPast
        )
    }

    /// Due date ascending, with undated chores last so they never displace a
    /// chore that has a real deadline, and id as a tie-break.
    ///
    /// The tie-break is not cosmetic. `sorted(by:)` is not stable, and chores
    /// sharing a due date are common — a whole recurring set can land on the
    /// same midnight. Without a total order the widget's rows would shuffle
    /// between refreshes for no reason the user could see.
    private static func earliestFirst(_ lhs: Chore, _ rhs: Chore) -> Bool {
        switch (APIDate.parse(lhs.dueDate), APIDate.parse(rhs.dueDate)) {
        case let (left?, right?): return left == right ? lhs.id < rhs.id : left < right
        case (nil, _?):           return false
        case (_?, nil):           return true
        case (nil, nil):          return lhs.id < rhs.id
        }
    }
}
