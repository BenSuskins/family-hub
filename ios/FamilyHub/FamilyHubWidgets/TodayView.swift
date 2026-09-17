import Foundation
import AppIntents
import SwiftUI
import WidgetKit
import FamilyHubKit

struct TodayView: View {
    @Environment(\.widgetFamily) private var family
    let entry: TodayEntry

    var body: some View {
        switch family {
        case .accessoryInline:      InlineView(entry: entry)
        case .accessoryCircular:    CircularView(entry: entry)
        case .accessoryRectangular: RectangularView(entry: entry)
        case .systemSmall:          SmallView(entry: entry)
        case .systemLarge:          AgendaView(entry: entry, maxRows: 6, showsEveryMeal: true)
        default:                    AgendaView(entry: entry, maxRows: 3, showsEveryMeal: false)
        }
    }
}

// MARK: - Shared vocabulary

private extension TodayEntry {
    /// Short explanation for the states where there is no day to draw.
    var problem: String? {
        switch status {
        case .ok, .cached:  return nil
        case .signedOut:    return "Open Family Hub to sign in"
        case .unavailable:  return "Can't reach the hub"
        }
    }

    var showsAge: Bool { status == .cached }

    /// The next thing the day holds — which, while anything is overdue, is the
    /// overdue chore, because that is what the ordering puts first.
    var nextUp: TodayWidgetData.Item? { hasData ? data.items.first : nil }

    /// What the circular family leads with: the number worth acting on, and a
    /// glyph saying what it counts. `nil` when there is nothing on, or nothing
    /// truthful to show — an unreachable server is not an empty day.
    var circularBadge: (value: Int, symbol: String)? {
        guard hasData else { return nil }
        if data.choreCount > 0 { return (data.choreCount, "checklist") }
        if data.eventCount > 0 { return (data.eventCount, "calendar") }
        return nil
    }
}

private extension TodayWidgetData.Item {
    /// Overdue is red whatever the row is; otherwise an event wears its own
    /// calendar's colour and a chore the widget's amber.
    var tint: Color {
        if isOverdue { return .red }
        switch kind {
        case .event: return colorHex.flatMap { Color(hex: $0) } ?? .blue
        case .chore: return .orange
        }
    }

    /// An event belongs to the calendar tab; chores live on Home, since the app
    /// has no Chores tab.
    var destination: URL {
        switch kind {
        case .event: return WidgetLink.calendar
        case .chore: return WidgetLink.today
        }
    }

    /// One line for the tight families, e.g. "17:30 Swimming lesson".
    var oneLine: String {
        [timeLabel, title].compactMap { $0 }.joined(separator: " ")
    }
}

/// "3 chores · 2 events" — what the day amounts to, for the families with no
/// room to list it.
private func daySummary(_ data: TodayWidgetData) -> String {
    var parts: [String] = []
    if data.choreCount > 0 {
        parts.append("\(data.choreCount) \(data.choreCount == 1 ? "chore" : "chores")")
    }
    if data.eventCount > 0 {
        parts.append("\(data.eventCount) \(data.eventCount == 1 ? "event" : "events")")
    }
    return parts.isEmpty ? "Nothing on" : parts.joined(separator: " · ")
}

// MARK: - Home Screen

/// Medium and large: the same merged day, differing in how many rows fit and
/// how much of the meal plan comes with it.
private struct AgendaView: View {
    let entry: TodayEntry
    let maxRows: Int
    let showsEveryMeal: Bool

    private var rows: [TodayWidgetData.Item] {
        Array(entry.data.items.prefix(maxRows))
    }

    private var hiddenCount: Int {
        entry.data.hiddenCount(shown: rows.count)
    }

    private var meals: [TodayWidgetData.Meal] {
        guard entry.hasData else { return [] }
        if showsEveryMeal { return entry.data.meals }
        // One line to spare, so lead with dinner — but a day where only lunch
        // is planned should still say so rather than show nothing.
        return [entry.data.dinner ?? entry.data.meals.last].compactMap { $0 }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            header

            if let problem = entry.problem {
                Spacer(minLength: 0)
                Text(problem)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(.secondary)
                Spacer(minLength: 0)
            } else if entry.data.isClear {
                Spacer(minLength: 0)
                HStack(spacing: 8) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                    Text("Nothing on today")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity)
                Spacer(minLength: 0)
            } else {
                ForEach(rows) { row in
                    TimelineRow(item: row)
                }

                if hiddenCount > 0 {
                    Text("+\(hiddenCount) more")
                        .font(.system(size: 11))
                        .foregroundStyle(.secondary)
                }

                Spacer(minLength: 0)
                MealFooter(meals: meals)
            }
        }
    }

    private var header: some View {
        HStack(spacing: 6) {
            Text("Today")
                .font(.system(size: 13, weight: .semibold))
            Text(entry.date, format: .dateTime.weekday(.abbreviated).day().month(.abbreviated))
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
            Spacer(minLength: 0)
            if entry.hasData, entry.data.overdueCount > 0 {
                Text("\(entry.data.overdueCount) overdue")
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(.red)
            }
            AgeStamp(entry: entry)
        }
    }
}

/// One row of the day. The row body is a link to the right tab; the tick, where
/// there is one, is a sibling rather than nested inside it, so a tap either
/// completes the chore or opens the app and never has to guess.
private struct TimelineRow: View {
    let item: TodayWidgetData.Item

    var body: some View {
        HStack(spacing: 8) {
            Link(destination: item.destination) {
                HStack(spacing: 8) {
                    TimeColumn(item: item)
                    KindMarker(item: item)
                    VStack(alignment: .leading, spacing: 0) {
                        Text(item.title)
                            .font(.system(size: 13, weight: .medium))
                            // Explicit, because a Text inside a Link would
                            // otherwise pick up the accent tint.
                            .foregroundStyle(.primary)
                            .lineLimit(1)
                        if let subtitle = item.subtitle {
                            Text(subtitle)
                                .font(.system(size: 10))
                                .foregroundStyle(item.isOverdue ? Color.red : Color.secondary)
                                .lineLimit(1)
                        }
                    }
                    Spacer(minLength: 0)
                }
            }

            // The tick is the best bit of the widget: the common case is "I've
            // just done that", and it should not need the app. Events are
            // someone else's calendar, so their rows have no button.
            if let choreID = item.choreID {
                Button(intent: CompleteChoreIntent(choreID: choreID)) {
                    Image(systemName: "checkmark.circle")
                        .font(.system(size: 18, weight: .regular))
                        .foregroundStyle(.green)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Complete \(item.title)")
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

/// The left-hand clock column. Fixed width so titles line up down the day even
/// when a row has no time to show.
private struct TimeColumn: View {
    let item: TodayWidgetData.Item

    var body: some View {
        Text(item.timeLabel ?? "")
            .font(.system(size: 11, weight: .semibold).monospacedDigit())
            .foregroundStyle(item.isOverdue ? Color.red : Color.secondary)
            .lineLimit(1)
            .minimumScaleFactor(0.75)
            .frame(width: 46, alignment: .trailing)
    }
}

/// Circle for an event, square for a chore — the same shape language the app's
/// calendar uses, so the two kinds stay apart even for a colour-blind reader.
private struct KindMarker: View {
    let item: TodayWidgetData.Item

    var body: some View {
        Group {
            switch item.kind {
            case .event:
                Circle().fill(item.tint)
            case .chore:
                RoundedRectangle(cornerRadius: 1.5, style: .continuous).fill(item.tint)
            }
        }
        .frame(width: 7, height: 7)
    }
}

/// The meal plan, pinned below the agenda rather than merged into it: meals
/// have no clock time to sort by, and dinner should stay visible however busy
/// the day above it gets.
private struct MealFooter: View {
    let meals: [TodayWidgetData.Meal]

    var body: some View {
        if !meals.isEmpty {
            Link(destination: WidgetLink.meals) {
                VStack(alignment: .leading, spacing: 2) {
                    Divider().opacity(0.4)
                    ForEach(meals) { meal in
                        HStack(spacing: 5) {
                            Image(systemName: "fork.knife")
                                .font(.system(size: 9, weight: .semibold))
                                .foregroundStyle(.secondary)
                                .frame(width: 11)
                            Text(meal.slot.label)
                                .font(.system(size: 10, weight: .semibold))
                                .foregroundStyle(.secondary)
                            Text(meal.name)
                                .font(.system(size: 11))
                                .foregroundStyle(.primary)
                                .lineLimit(1)
                            Spacer(minLength: 0)
                        }
                    }
                }
            }
        }
    }
}

private struct SmallView: View {
    let entry: TodayEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 4) {
                Text(entry.date, format: .dateTime.weekday(.abbreviated).day())
                    .font(.system(size: 12, weight: .semibold))
                Spacer(minLength: 0)
                AgeStamp(entry: entry)
            }

            if let problem = entry.problem {
                Spacer(minLength: 4)
                Text(problem)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
                Spacer(minLength: 0)
            } else {
                Text(daySummary(entry.data))
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(entry.data.overdueCount > 0 ? Color.red : Color.secondary)
                    .lineLimit(1)

                Spacer(minLength: 6)

                // Small has room for one thing, so it shows the thing that's
                // next rather than a count with no context.
                if let next = entry.nextUp {
                    VStack(alignment: .leading, spacing: 2) {
                        HStack(spacing: 5) {
                            KindMarker(item: next)
                            Text(next.timeLabel ?? "Any time")
                                .font(.system(size: 11, weight: .semibold).monospacedDigit())
                                .foregroundStyle(next.isOverdue ? Color.red : Color.secondary)
                                .lineLimit(1)
                        }
                        Text(next.title)
                            .font(.system(size: 15, weight: .semibold))
                            .lineLimit(2)
                    }
                } else {
                    Text("Nothing scheduled")
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(.secondary)
                }

                Spacer(minLength: 4)

                if let dinner = entry.data.dinner {
                    HStack(spacing: 5) {
                        Image(systemName: "fork.knife")
                            .font(.system(size: 9, weight: .semibold))
                        Text(dinner.name)
                            .font(.system(size: 11))
                            .lineLimit(1)
                    }
                    .foregroundStyle(.secondary)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

/// Marks data the widget could not refresh, so a stale day is never mistaken
/// for a current one.
private struct AgeStamp: View {
    let entry: TodayEntry

    var body: some View {
        if entry.showsAge {
            HStack(spacing: 2) {
                Image(systemName: "clock")
                Text(entry.data.capturedAt, style: .time)
            }
            .font(.system(size: 10))
            .foregroundStyle(.tertiary)
        }
    }
}

// MARK: - Lock Screen

private struct InlineView: View {
    let entry: TodayEntry

    var body: some View {
        if let problem = entry.problem {
            Text(problem)
        } else if let next = entry.nextUp {
            Text(next.oneLine)
        } else if let dinner = entry.data.dinner {
            Text("Dinner · \(dinner.name)")
        } else {
            Text("Nothing on today")
        }
    }
}

private struct CircularView: View {
    let entry: TodayEntry

    var body: some View {
        ZStack {
            AccessoryWidgetBackground()
            if !entry.hasData {
                // Signed out or unreachable: say "unknown", never "all clear".
                Image(systemName: "questionmark")
                    .font(.system(size: 16, weight: .semibold))
            } else if let badge = entry.circularBadge {
                VStack(spacing: -2) {
                    Text("\(badge.value)")
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                    Image(systemName: badge.symbol)
                        .font(.system(size: 9, weight: .semibold))
                }
            } else {
                Image(systemName: "checkmark")
                    .font(.system(size: 18, weight: .semibold))
            }
        }
        .widgetAccentable()
    }
}

private struct RectangularView: View {
    let entry: TodayEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 1) {
            if let problem = entry.problem {
                Text("Family Hub").font(.headline).widgetAccentable()
                Text(problem).font(.caption)
            } else {
                HStack(spacing: 4) {
                    Text(daySummary(entry.data))
                        .font(.headline)
                        .widgetAccentable()
                        .lineLimit(1)
                    if entry.data.overdueCount > 0 {
                        Text("· \(entry.data.overdueCount) late").font(.caption)
                    }
                }
                if let next = entry.nextUp {
                    Text(next.oneLine).font(.caption).lineLimit(1)
                }
                if let dinner = entry.data.dinner {
                    Text("Dinner · \(dinner.name)").font(.caption2).lineLimit(1)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}
