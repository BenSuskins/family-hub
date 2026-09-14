import Foundation
import AppIntents
import SwiftUI
import WidgetKit
import FamilyHubKit

struct TodayChoresView: View {
    @Environment(\.widgetFamily) private var family
    let entry: TodayEntry

    var body: some View {
        switch family {
        case .accessoryInline:      InlineView(entry: entry)
        case .accessoryCircular:    CircularView(entry: entry)
        case .accessoryRectangular: RectangularView(entry: entry)
        case .systemSmall:          SmallView(entry: entry)
        case .systemLarge:          ListView(entry: entry, maxRows: 6)
        default:                    ListView(entry: entry, maxRows: 3)
        }
    }
}

// MARK: - Shared vocabulary

private extension TodayEntry {
    /// The single number the widget leads with. `nil` when there is nothing
    /// truthful to show — an unreachable server is not "nothing to do".
    var outstanding: Int? { hasData ? data.totalCount : nil }

    /// Short explanation for the states where there is no data to draw.
    var problem: String? {
        switch status {
        case .ok, .cached:  return nil
        case .signedOut:    return "Open Family Hub to sign in"
        case .unavailable:  return "Can't reach the hub"
        }
    }

    var showsAge: Bool { status == .cached }
}

private func choreWord(_ count: Int) -> String {
    count == 1 ? "chore" : "chores"
}

// MARK: - Home Screen

private struct SmallView: View {
    let entry: TodayEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Image(systemName: "checklist")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(.secondary)
                Spacer()
                AgeStamp(entry: entry)
            }

            Spacer(minLength: 4)

            if let problem = entry.problem {
                Text(problem)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
            } else if entry.data.isClear {
                Text("All done")
                    .font(.system(size: 22, weight: .bold))
                Text("Nothing due today")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            } else {
                Text("\(entry.data.totalCount)")
                    .font(.system(size: 40, weight: .bold, design: .rounded))
                    .contentTransition(.numericText())
                Text("\(choreWord(entry.data.totalCount)) to do")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                if entry.data.overdueCount > 0 {
                    Text("\(entry.data.overdueCount) overdue")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(.red)
                }
            }

            Spacer(minLength: 0)

            if let first = entry.data.items.first, !entry.data.isClear, entry.problem == nil {
                Text(first.name)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

/// Medium and large: the same list, differing only in how many rows fit.
private struct ListView: View {
    let entry: TodayEntry
    let maxRows: Int

    private var rows: [TodayWidgetData.Item] {
        Array(entry.data.items.prefix(maxRows))
    }

    private var remainder: Int {
        max(0, entry.data.totalCount - rows.count)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
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
                    Text("Nothing due today")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity)
                Spacer(minLength: 0)
            } else {
                ForEach(rows) { item in
                    ChoreRow(item: item)
                }
                if remainder > 0 {
                    Text("+\(remainder) more")
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                        .padding(.top, 2)
                }
                Spacer(minLength: 0)
            }
        }
    }

    private var header: some View {
        HStack(spacing: 6) {
            Text("Today")
                .font(.system(size: 13, weight: .semibold))
            if entry.hasData, entry.data.overdueCount > 0 {
                Text("\(entry.data.overdueCount) overdue")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(.red)
            }
            Spacer()
            AgeStamp(entry: entry)
        }
    }
}

private struct ChoreRow: View {
    let item: TodayWidgetData.Item

    var body: some View {
        HStack(spacing: 9) {
            InitialsBadge(initials: item.assigneeInitials, isOverdue: item.isOverdue)

            VStack(alignment: .leading, spacing: 0) {
                Text(item.name)
                    .font(.system(size: 14, weight: .medium))
                    .lineLimit(1)
                if let subtitle {
                    Text(subtitle)
                        .font(.system(size: 11))
                        .foregroundStyle(item.isOverdue ? Color.red : Color.secondary)
                        .lineLimit(1)
                }
            }

            Spacer(minLength: 0)

            // The tick is the whole point of the widget: the common case is
            // "I've just done that", and it should not need the app.
            Button(intent: CompleteChoreIntent(choreID: item.id)) {
                Image(systemName: "checkmark.circle")
                    .font(.system(size: 19, weight: .regular))
                    .foregroundStyle(.green)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Complete \(item.name)")
        }
    }

    /// "Ben · Overdue Sep 12", with whichever halves exist.
    private var subtitle: String? {
        var parts: [String] = []
        if let name = item.assigneeName { parts.append(name) }
        if item.isOverdue {
            parts.append(item.detail.map { "Overdue \($0)" } ?? "Overdue")
        } else if let detail = item.detail {
            parts.append(detail)
        }
        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }
}

private struct InitialsBadge: View {
    let initials: String?
    let isOverdue: Bool

    private var tint: Color { isOverdue ? .red : .secondary }

    var body: some View {
        ZStack {
            Circle().fill(tint.opacity(0.16))
            Text(initials ?? "?")
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(tint)
        }
        .frame(width: 24, height: 24)
    }
}

/// Marks data the widget could not refresh, so a stale count is never mistaken
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
        } else if entry.data.isClear {
            Text("Chores all done")
        } else if entry.data.overdueCount > 0 {
            Text("\(entry.data.totalCount) \(choreWord(entry.data.totalCount)) · \(entry.data.overdueCount) overdue")
        } else {
            Text("\(entry.data.totalCount) \(choreWord(entry.data.totalCount)) today")
        }
    }
}

private struct CircularView: View {
    let entry: TodayEntry

    var body: some View {
        ZStack {
            AccessoryWidgetBackground()
            if let outstanding = entry.outstanding {
                if outstanding == 0 {
                    Image(systemName: "checkmark")
                        .font(.system(size: 18, weight: .semibold))
                } else {
                    VStack(spacing: -2) {
                        Text("\(outstanding)")
                            .font(.system(size: 20, weight: .bold, design: .rounded))
                        Image(systemName: "checklist")
                            .font(.system(size: 9, weight: .semibold))
                    }
                }
            } else {
                // Signed out or unreachable: say "unknown", never "all clear".
                Image(systemName: "questionmark")
                    .font(.system(size: 16, weight: .semibold))
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
            } else if entry.data.isClear {
                Text("Chores").font(.headline).widgetAccentable()
                Text("Nothing due today").font(.caption)
            } else {
                HStack(spacing: 4) {
                    Text("\(entry.data.totalCount) \(choreWord(entry.data.totalCount))")
                        .font(.headline)
                        .widgetAccentable()
                    if entry.data.overdueCount > 0 {
                        Text("· \(entry.data.overdueCount) overdue").font(.caption)
                    }
                }
                if let first = entry.data.items.first {
                    Text(first.name).font(.caption).lineLimit(1)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}
