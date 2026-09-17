import Foundation

/// Presentation-ready view of the family's day — the agenda, the outstanding
/// chores and the meal plan merged into one ordered list — written by the app
/// and rendered by the widget extension.
///
/// The merge, ordering and labelling live here rather than in the widget's
/// SwiftUI views for two reasons. A widget process gets a very small time and
/// memory budget to produce a view, so it should draw a prepared list rather
/// than join three payloads against the family's users itself. And this layer
/// is the one the Linux CI job can test in seconds — logic buried in a `View`
/// needs a simulator.
///
/// **The chore counts are of outstanding work only.** `GET /api/dashboard`
/// reports *pending* chores due today plus overdue ones; the server's
/// `FindDueToday` filters on `status = 'pending'`, so nothing in the payload
/// says how many were completed today. The widget therefore says "4 to do"
/// rather than inventing a "3 of 7 done" denominator it has no way to source.
///
/// The encoding is not compatible with the chores-only snapshot this replaced.
/// That needs no migration: a snapshot that won't decode reads back as "no
/// cache", which is exactly the state that sends the widget to the network.
public struct TodayWidgetData: Codable, Equatable {

    /// What a timeline row is. Decides the row's tint, and whether it carries a
    /// tick button — an iCal event is someone else's to change.
    public enum Kind: String, Codable {
        case chore
        case event
    }

    /// One row of the day's timeline, flattened so no lookups happen at render
    /// time.
    public struct Item: Codable, Equatable, Identifiable {
        /// Namespaced by kind, because a chore and an event can share a raw id
        /// and `ForEach` would silently drop one of the two.
        public let id: String
        public let kind: Kind
        public let title: String
        /// Second line: the assignee for a chore, the location for an event.
        public let subtitle: String?
        /// Left-hand time column — "18:30", "All day", "Overdue", or nil for a
        /// chore with no time of day, which has nothing truthful to put there.
        public let timeLabel: String?
        public let isOverdue: Bool
        /// Set on chores only: the id ``CompleteChoreIntent`` ticks off. Its
        /// presence is what gives a row a tick button.
        public let choreID: String?
        /// Up to two initials for a chore's assignee. Widgets deliberately
        /// don't fetch avatar images: that would mean a network round-trip and
        /// an image cache inside a process with milliseconds to render.
        public let initials: String?
        /// The source calendar's colour as `RRGGBB`, for events.
        public let colorHex: String?

        public init(
            id: String,
            kind: Kind,
            title: String,
            subtitle: String? = nil,
            timeLabel: String? = nil,
            isOverdue: Bool = false,
            choreID: String? = nil,
            initials: String? = nil,
            colorHex: String? = nil
        ) {
            self.id = id
            self.kind = kind
            self.title = title
            self.subtitle = subtitle
            self.timeLabel = timeLabel
            self.isOverdue = isOverdue
            self.choreID = choreID
            self.initials = initials
            self.colorHex = colorHex
        }

        public static func chore(_ chore: Chore, user: User?, isOverdue: Bool) -> Item {
            Item(
                id: "chore-\(chore.id)",
                kind: .chore,
                title: chore.name,
                subtitle: choreSubtitle(chore, user: user, isOverdue: isOverdue),
                // An overdue chore's own due time is noise: what matters is
                // that it should already have been done.
                timeLabel: isOverdue ? "Overdue" : TodayWidgetData.clockLabel(clock: chore.dueTime),
                isOverdue: isOverdue,
                choreID: chore.id,
                initials: user.map(\.initials)
            )
        }

        public static func event(_ event: CalendarEvent) -> Item {
            Item(
                id: "event-\(event.id)",
                kind: .event,
                title: event.title,
                // Where it is beats what it is about: the widget has one line,
                // and "Leisure centre" is the bit you act on.
                subtitle: nonEmpty(event.location) ?? nonEmpty(event.description),
                timeLabel: event.allDay ? "All day" : TodayWidgetData.clockLabel(event.startTime),
                colorHex: nonEmpty(event.color)
            )
        }

        /// "Ben · Sep 12" for an overdue chore, and just the assignee for one
        /// due today, whose time already sits in the time column.
        private static func choreSubtitle(_ chore: Chore, user: User?, isOverdue: Bool) -> String? {
            var parts: [String] = []
            if let name = user?.name { parts.append(name) }
            if isOverdue, let due = chore.formattedDueDate { parts.append(due) }
            return parts.isEmpty ? nil : parts.joined(separator: " · ")
        }

        private static func nonEmpty(_ value: String) -> String? {
            let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? nil : trimmed
        }
    }

    /// A planned meal.
    ///
    /// Kept out of the timeline rows: meals carry no clock time to sort by, and
    /// dinner is the slot people actually check — pinning it below the agenda
    /// keeps it visible however full the day is, rather than letting breakfast
    /// win the last row at eight in the morning.
    public struct Meal: Codable, Equatable, Identifiable {
        public enum Slot: String, Codable {
            case breakfast
            case lunch
            case dinner

            public var label: String {
                switch self {
                case .breakfast: return "Breakfast"
                case .lunch:     return "Lunch"
                case .dinner:    return "Dinner"
                }
            }

            /// Order within the day.
            var order: Int {
                switch self {
                case .breakfast: return 0
                case .lunch:     return 1
                case .dinner:    return 2
                }
            }
        }

        public let slot: Slot
        public let name: String

        public var id: String { slot.rawValue }

        public init(slot: Slot, name: String) {
            self.slot = slot
            self.name = name
        }
    }

    /// The day's timeline: overdue chores, then all-day events, then everything
    /// with a clock time in order, then chores with no time. Capped at
    /// ``maxItems``.
    public let items: [Item]
    /// How many rows the day holds before the cap, so a view can honestly say
    /// "+3 more".
    public let totalItemCount: Int
    /// Total overdue chores, which may exceed the number carried in ``items``.
    public let overdueCount: Int
    /// Total chores pending and due today, likewise uncapped.
    public let dueTodayCount: Int
    /// Total events today, likewise uncapped.
    public let eventCount: Int
    /// Today's meals, breakfast first.
    public let meals: [Meal]
    /// When this was built, so the widget can say how stale it is.
    public let capturedAt: Date

    public var choreCount: Int { overdueCount + dueTodayCount }

    /// Nothing planned, nothing outstanding, nothing cooking — the one state
    /// where the widget has nothing to list.
    public var isClear: Bool { totalItemCount == 0 && meals.isEmpty }

    /// The meal the widget leads with when it has room for only one.
    public var dinner: Meal? { meals.first { $0.slot == .dinner } }

    /// Rows the day holds that a view showing `shown` of them had no room for.
    public func hiddenCount(shown: Int) -> Int { max(0, totalItemCount - shown) }

    public init(
        items: [Item],
        totalItemCount: Int,
        overdueCount: Int,
        dueTodayCount: Int,
        eventCount: Int,
        meals: [Meal],
        capturedAt: Date
    ) {
        self.items = items
        self.totalItemCount = totalItemCount
        self.overdueCount = overdueCount
        self.dueTodayCount = dueTodayCount
        self.eventCount = eventCount
        self.meals = meals
        self.capturedAt = capturedAt
    }

    /// An empty day, never refreshed — the placeholder a widget shows before it
    /// has any real data.
    public static func blank(at date: Date = Date()) -> TodayWidgetData {
        TodayWidgetData(
            items: [],
            totalItemCount: 0,
            overdueCount: 0,
            dueTodayCount: 0,
            eventCount: 0,
            meals: [],
            capturedAt: date
        )
    }

    /// How many rows a snapshot carries. The largest widget family lists fewer;
    /// the cap keeps the app-group payload small, since `UserDefaults` is a
    /// poor home for unbounded data.
    public static let maxItems = 10

    /// Build a day from a dashboard payload, the day's calendar events and the
    /// family's users.
    ///
    /// `events` is expected to be a single day's worth — what
    /// `GET /api/calendar?view=day` returns — and is not filtered again here.
    /// Both callers ask for today, and a snapshot that outlives its day goes
    /// stale by ``capturedAt`` long before its contents mislead anyone.
    public static func from(
        stats: DashboardStats,
        events: [CalendarEvent] = [],
        users: [String: User] = [:],
        capturedAt: Date = Date()
    ) -> TodayWidgetData {
        var placed: [Placed] = []

        // Overdue leads: it is the part of the day that needed acting on
        // yesterday. Pre-sorted earliest-first, and the resulting index carries
        // that order through the merge.
        for (index, chore) in stats.choresOverdueList.sorted(by: earliestFirst).enumerated() {
            placed.append(Placed(
                item: .chore(chore, user: assignee(of: chore, in: users), isOverdue: true),
                band: Band.overdue,
                minute: index,
                tiebreak: chore.id
            ))
        }

        for event in events {
            placed.append(Placed(
                item: .event(event),
                band: event.allDay ? Band.allDay : Band.timed,
                minute: event.allDay ? 0 : minutesFromMidnight(event.startTime),
                // Events before chores at the same minute: somewhere you have
                // to be outranks something you could do whenever.
                tiebreak: "0-\(event.title)-\(event.id)"
            ))
        }

        for chore in stats.choresDueTodayList {
            // A chore's time of day lives only in `DueTime`; the server parses
            // `DueDate` as a bare date, so it is always local midnight and says
            // nothing about when in the day the chore belongs.
            let minute = minutesFromMidnight(clock: chore.dueTime)
            placed.append(Placed(
                item: .chore(chore, user: assignee(of: chore, in: users), isOverdue: false),
                band: minute == nil ? Band.untimed : Band.timed,
                minute: minute ?? 0,
                tiebreak: "1-\(chore.name)-\(chore.id)"
            ))
        }

        let ordered = placed.sorted(by: inDayOrder).map(\.item)

        return TodayWidgetData(
            items: Array(ordered.prefix(maxItems)),
            totalItemCount: ordered.count,
            overdueCount: stats.choresOverdueList.count,
            dueTodayCount: stats.choresDueTodayList.count,
            eventCount: events.count,
            meals: plannedMeals(from: stats.todayMeals),
            capturedAt: capturedAt
        )
    }

    /// A copy without `choreID`, with the matching counts decremented.
    ///
    /// Used for the optimistic redraw after a widget tick-off: the server call
    /// has succeeded, but re-fetching before drawing would make the tick feel
    /// slow. Unknown ids are returned unchanged, so a duplicate tap can't drive
    /// a count negative.
    public func removing(choreID: String) -> TodayWidgetData {
        guard let removed = items.first(where: { $0.choreID == choreID }) else { return self }
        return TodayWidgetData(
            items: items.filter { $0.choreID != choreID },
            totalItemCount: max(0, totalItemCount - 1),
            overdueCount: removed.isOverdue ? max(0, overdueCount - 1) : overdueCount,
            dueTodayCount: removed.isOverdue ? dueTodayCount : max(0, dueTodayCount - 1),
            eventCount: eventCount,
            meals: meals,
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
            totalItemCount: totalItemCount,
            overdueCount: overdueCount,
            dueTodayCount: dueTodayCount,
            eventCount: eventCount,
            meals: meals,
            capturedAt: .distantPast
        )
    }

    // MARK: - Ordering

    /// Where a row sits in the day, before any clock time is compared.
    private enum Band {
        static let overdue = 0
        static let allDay = 1
        static let timed = 2
        static let untimed = 3
    }

    /// A row plus the keys that place it in the day.
    ///
    /// The order has to be a total one. `sorted(by:)` is not stable, and rows
    /// sharing a minute are common — a whole recurring set can land on the same
    /// time — so without a tie-break the widget's rows would shuffle between
    /// refreshes for no reason the user could see.
    private struct Placed {
        let item: Item
        let band: Int
        /// Minutes from local midnight for timed rows; for overdue chores, the
        /// index that carries their earliest-first order through the merge.
        let minute: Int
        let tiebreak: String
    }

    private static func inDayOrder(_ lhs: Placed, _ rhs: Placed) -> Bool {
        if lhs.band != rhs.band { return lhs.band < rhs.band }
        if lhs.minute != rhs.minute { return lhs.minute < rhs.minute }
        return lhs.tiebreak < rhs.tiebreak
    }

    /// Due date ascending, with undated chores last so they never displace one
    /// that has a real deadline, and id as a tie-break.
    private static func earliestFirst(_ lhs: Chore, _ rhs: Chore) -> Bool {
        switch (APIDate.parse(lhs.dueDate), APIDate.parse(rhs.dueDate)) {
        case let (left?, right?): return left == right ? lhs.id < rhs.id : left < right
        case (nil, _?):           return false
        case (_?, nil):           return true
        case (nil, nil):          return lhs.id < rhs.id
        }
    }

    // MARK: - Building blocks

    private static func assignee(of chore: Chore, in users: [String: User]) -> User? {
        chore.assignedToUserID.flatMap { users[$0] }
    }

    /// Today's meals as the widget draws them, breakfast first. Slots the
    /// server doesn't recognise, and blank names, are dropped rather than
    /// rendered as an empty row.
    private static func plannedMeals(from plans: [MealPlan]) -> [Meal] {
        plans.compactMap { plan -> Meal? in
            guard let slot = Meal.Slot(rawValue: plan.mealType.lowercased()) else { return nil }
            let name = plan.name.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !name.isEmpty else { return nil }
            return Meal(slot: slot, name: name)
        }
        .sorted { $0.slot.order < $1.slot.order }
    }

    /// Minutes from local midnight. Used only to order rows against each other.
    private static func minutesFromMidnight(_ date: Date) -> Int {
        let parts = Calendar.current.dateComponents([.hour, .minute], from: date)
        return (parts.hour ?? 0) * 60 + (parts.minute ?? 0)
    }

    /// Minutes from midnight for an `HH:mm` chore due time, or nil when the
    /// chore has no time of day and belongs in the untimed band.
    private static func minutesFromMidnight(clock value: String?) -> Int? {
        guard let value, !value.isEmpty, let date = APIDate.time.date(from: value) else { return nil }
        return minutesFromMidnight(date)
    }

    /// A clock time written the way the phone writes it — 24-hour in the UK,
    /// 12-hour where that is the convention. Built once: `DateFormatter` is
    /// expensive and a widget has milliseconds to render.
    private static let clockFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .none
        formatter.timeStyle = .short
        return formatter
    }()

    fileprivate static func clockLabel(_ date: Date) -> String {
        clockFormatter.string(from: date)
    }

    /// Re-writes a chore's wire-format `HH:mm` due time in the phone's clock
    /// format, falling back to the raw value if it won't parse.
    fileprivate static func clockLabel(clock value: String?) -> String? {
        guard let value, !value.isEmpty else { return nil }
        guard let date = APIDate.time.date(from: value) else { return value }
        return clockLabel(date)
    }
}
