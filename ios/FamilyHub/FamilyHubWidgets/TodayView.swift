import AppIntents
import SwiftUI
import WidgetKit
import FamilyHubKit

enum WidgetPresentation: Equatable {
    case today
    case tonight
    case mealPlan
    case upNext
    case chores
}

struct TodayView: View {
    @Environment(\.widgetFamily) private var family
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    let entry: TodayEntry
    var presentation: WidgetPresentation = .today

    private var largeText: Bool { dynamicTypeSize >= .xxLarge }

    var body: some View {
        Group {
            switch presentation {
            case .today: today
            case .tonight: tonight
            case .mealPlan: mealPlan
            case .upNext: upNext
            case .chores: chores
            }
        }
        .foregroundStyle(WidgetPalette.ink)
    }

    private var today: some View {
        Group {
            switch family {
            case .accessoryInline: inlineDinner
            case .accessoryCircular: circularChores
            case .accessoryRectangular: rectangularEvent
            case .systemLarge: largeToday
            case .systemSmall: smallToday
            default: mediumToday
            }
        }
    }

    private var mediumToday: some View {
        VStack(alignment: .leading, spacing: 8) {
            header(title: "Today")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else {
                if entry.data.isClear {
                    EmptyDayMessage()
                } else {
                HStack(alignment: .top, spacing: 8) {
                    EventHighlight(item: entry.nextEvent)
                    Rectangle().fill(WidgetPalette.border).frame(width: 1)
                    MealHighlight(meal: entry.data.dinner, title: "Tonight")
                }
                .frame(maxHeight: .infinity)
                ChoreSummary(entry: entry)
                if entry.failedChoreID != nil {
                    Text("Could not complete the chore. It stays unchecked.")
                        .font(.caption2)
                        .foregroundStyle(WidgetPalette.clay)
                }
                if entry.pendingChoreID != nil {
                    Text("Saving chore…")
                        .font(.caption2)
                        .foregroundStyle(WidgetPalette.secondary)
                }
                }
            }
        }
    }

    private var largeToday: some View {
        VStack(alignment: .leading, spacing: largeText ? 5 : 8) {
            header(title: "Today")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else if entry.data.isClear {
                EmptyDayMessage()
            } else {
                MealHighlight(meal: entry.data.dinner, title: "Tonight")
                sectionTitle("Up next")
                let events = entry.data.events
                    .filter { $0.isAllDay != true && ($0.eventEnd ?? $0.eventStart ?? .distantPast) > entry.date }
                    .sorted { ($0.eventStart ?? .distantFuture) < ($1.eventStart ?? .distantFuture) }
                    .prefix(largeText ? 1 : 2)
                if events.isEmpty {
                    Text("Nothing else scheduled today")
                        .font(.caption)
                        .foregroundStyle(WidgetPalette.secondary)
                } else {
                    ForEach(Array(events)) { item in
                        EventRow(item: item)
                    }
                }
                sectionTitle("Chores")
                let chores = Array(entry.chores.prefix(largeText ? 1 : 2))
                if chores.isEmpty {
                    Text("No chores due today")
                        .font(.caption)
                        .foregroundStyle(WidgetPalette.secondary)
                } else {
                    ForEach(chores) { item in
                        ChoreRow(
                            item: item,
                            failed: item.choreID == entry.failedChoreID,
                            pending: item.choreID == entry.pendingChoreID
                        )
                    }
                }
                Spacer(minLength: 0)
            }
        }
    }

    private var smallToday: some View {
        VStack(alignment: .leading, spacing: 8) {
            header(title: "Today")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else {
                EventHighlight(item: entry.nextEvent)
                Spacer(minLength: 0)
                MealHighlight(meal: entry.data.dinner, title: "Tonight")
            }
        }
    }

    private var tonight: some View {
        VStack(alignment: .leading, spacing: 8) {
            header(title: "Dinner · Today")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else if let dinner = entry.data.dinner {
                Link(destination: WidgetLink.meals) {
                    VStack(alignment: .leading, spacing: 10) {
                        Image(systemName: "fork.knife")
                            .font(.title2.weight(.medium))
                            .foregroundStyle(WidgetPalette.accent)
                            .frame(width: 42, height: 42)
                            .background(WidgetPalette.sage, in: Circle())
                        Text(dinner.name)
                            .font(.title3.weight(.semibold))
                            .lineLimit(3)
                            .privacySensitive()
                        Spacer(minLength: 0)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
                }
            } else {
                Text("No dinner planned")
                    .font(.subheadline)
                    .foregroundStyle(WidgetPalette.secondary)
                Spacer(minLength: 0)
            }
        }
    }

    private var mealPlan: some View {
        VStack(alignment: .leading, spacing: 5) {
            header(title: "Meal plan")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else if entry.data.meals.isEmpty {
                Text("No meals planned today")
                    .font(.subheadline)
                    .foregroundStyle(WidgetPalette.secondary)
                Spacer(minLength: 0)
            } else {
                let meals = largeText
                    ? [entry.data.dinner ?? entry.data.meals.last].compactMap { $0 }
                    : entry.data.meals
                ForEach(meals) { meal in
                    Link(destination: WidgetLink.meals) {
                        HStack(spacing: 8) {
                            Image(systemName: "fork.knife")
                                .font(.caption)
                                .foregroundStyle(WidgetPalette.accent)
                                .frame(width: 22)
                            Text(meal.slot.label)
                                .font(.caption.weight(.medium))
                                .foregroundStyle(WidgetPalette.secondary)
                                .frame(width: 66, alignment: .leading)
                            Text(meal.name)
                                .font(meal.slot == .dinner ? .subheadline.weight(.semibold) : .subheadline)
                                .lineLimit(1)
                                .privacySensitive()
                            Spacer(minLength: 0)
                        }
                        .frame(minHeight: 36)
                        .padding(.horizontal, meal.slot == .dinner ? 8 : 0)
                        .background(meal.slot == .dinner ? WidgetPalette.sage : .clear, in: RoundedRectangle(cornerRadius: 10))
                    }
                }
                if largeText && entry.data.meals.count > meals.count {
                    Text("+\(entry.data.meals.count - meals.count) more meals")
                        .font(.caption)
                        .foregroundStyle(WidgetPalette.secondary)
                }
                Spacer(minLength: 0)
            }
        }
    }

    private var upNext: some View {
        VStack(alignment: .leading, spacing: 5) {
            header(title: "Up next")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else if let event = entry.nextEvent {
                EventHighlight(item: event)
                Spacer(minLength: 0)
            } else {
                Text("Nothing else today")
                    .font(.subheadline)
                    .foregroundStyle(WidgetPalette.secondary)
                Spacer(minLength: 0)
            }
        }
    }

    private var chores: some View {
        Group {
            switch family {
            case .accessoryCircular: circularChores
            case .systemSmall: smallChores
            default: mediumChores
            }
        }
    }

    private var smallChores: some View {
        VStack(alignment: .leading, spacing: 8) {
            header(title: "Chores")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else {
                Text("\(entry.data.choreCount)")
                    .font(.system(.largeTitle, design: .rounded, weight: .bold))
                    .foregroundStyle(WidgetPalette.accent)
                Text(entry.data.overdueCount > 0 ? "to do · \(entry.data.overdueCount) overdue" : "chores to do")
                    .font(.caption)
                    .foregroundStyle(WidgetPalette.secondary)
                if entry.failedChoreID != nil {
                    Text("Could not complete · still open")
                        .font(.caption2)
                        .foregroundStyle(WidgetPalette.clay)
                }
                if entry.pendingChoreID != nil {
                    Text("Saving chore…")
                        .font(.caption2)
                        .foregroundStyle(WidgetPalette.secondary)
                }
                Spacer(minLength: 0)
            }
        }
    }

    private var mediumChores: some View {
        VStack(alignment: .leading, spacing: 7) {
            header(title: "Chores")
            if let problem = entry.problem {
                ProblemMessage(text: problem)
            } else if entry.chores.isEmpty {
                Text("No chores to do")
                    .font(.subheadline)
                    .foregroundStyle(WidgetPalette.secondary)
                Spacer(minLength: 0)
            } else {
                ForEach(entry.chores.prefix(2)) { item in
                    ChoreRow(
                        item: item,
                        failed: item.choreID == entry.failedChoreID,
                        pending: item.choreID == entry.pendingChoreID
                    )
                }
                if entry.data.choreCount > 2 {
                    Text("+\(entry.data.choreCount - 2) more")
                        .font(.caption)
                        .foregroundStyle(WidgetPalette.secondary)
                }
                Spacer(minLength: 0)
            }
        }
    }

    private var inlineDinner: some View {
        Group {
            if let problem = entry.problem {
                Text(problem)
            } else if let dinner = entry.data.dinner {
                Text("Dinner: \(dinner.name)").privacySensitive()
            } else {
                Text("No dinner planned")
            }
        }
    }

    private var circularChores: some View {
        Group {
            if let problem = entry.problem {
                Image(systemName: "exclamationmark")
                    .accessibilityLabel(problem)
            } else {
                Text("\(entry.data.choreCount)")
                    .font(.system(.title2, design: .rounded, weight: .bold))
                    .foregroundStyle(WidgetPalette.accent)
                    .accessibilityLabel("\(entry.data.choreCount) chores to do")
            }
        }
    }

    private var rectangularEvent: some View {
        Group {
            if let problem = entry.problem {
                Text(problem).font(.caption)
            } else if let event = entry.nextEvent {
                VStack(alignment: .leading, spacing: 2) {
                    Text(event.eventStart ?? entry.date, format: .dateTime.hour().minute())
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(WidgetPalette.accent)
                    Text(event.title).font(.caption.weight(.semibold)).lineLimit(1).privacySensitive()
                    if let location = event.subtitle {
                        Text(location).font(.caption2).lineLimit(1).foregroundStyle(WidgetPalette.secondary).privacySensitive()
                    }
                }
            } else {
                Text("Nothing else today").font(.caption)
            }
        }
    }

    private var header: some View { header(title: "Today") }

    private func header(title: String) -> some View {
        HStack(spacing: 5) {
            Text(title).font(.footnote.weight(.semibold))
            if presentation == .today || presentation == .tonight {
                Text(entry.date, format: .dateTime.weekday(.abbreviated).day().month(.abbreviated))
                    .font(.caption)
                    .foregroundStyle(WidgetPalette.secondary)
            }
            Spacer(minLength: 0)
            if entry.status == .cached {
                HStack(spacing: 3) {
                    Image(systemName: "clock")
                    Text(entry.data.capturedAt, style: .time)
                }
                .font(.caption2)
                .foregroundStyle(WidgetPalette.secondary)
                .accessibilityLabel("Saved at \(entry.data.capturedAt.formatted(date: .omitted, time: .shortened))")
            }
        }
    }

    private func sectionTitle(_ title: String) -> some View {
        Text(title.uppercased())
            .font(.caption2.weight(.semibold))
            .foregroundStyle(WidgetPalette.secondary)
            .padding(.top, 2)
    }
}

private extension TodayEntry {
    var problem: String? {
        switch status {
        case .ok, .cached: nil
        case .signedOut: "Open Family Hub to sign in"
        case .unavailable: "Can't reach the hub"
        }
    }
}

private struct ProblemMessage: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.subheadline.weight(.medium))
            .foregroundStyle(WidgetPalette.secondary)
            .fixedSize(horizontal: false, vertical: true)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
    }
}

private struct EmptyDayMessage: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Image(systemName: "checkmark.circle.fill")
                .font(.title3)
                .foregroundStyle(WidgetPalette.accent)
            Text("Nothing on today")
                .font(.subheadline.weight(.medium))
                .foregroundStyle(WidgetPalette.secondary)
            Spacer(minLength: 0)
        }
    }
}

private struct EventHighlight: View {
    let item: TodayWidgetData.Item?

    var body: some View {
        Group {
            if let item {
                Link(destination: WidgetLink.calendar) {
                    VStack(alignment: .leading, spacing: 5) {
                        Label("Up next", systemImage: "calendar")
                            .font(.caption.weight(.medium))
                            .foregroundStyle(WidgetPalette.accent)
                        if let start = item.eventStart {
                            Text(start, format: .dateTime.hour().minute())
                                .font(.caption2.weight(.semibold).monospacedDigit())
                                .foregroundStyle(WidgetPalette.secondary)
                        }
                        Text(item.title)
                            .font(.subheadline.weight(.semibold))
                            .lineLimit(2)
                            .privacySensitive()
                        if let subtitle = item.subtitle {
                            Text(subtitle)
                                .font(.caption)
                                .foregroundStyle(WidgetPalette.secondary)
                                .lineLimit(1)
                        }
                        Spacer(minLength: 0)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
                    .padding(8)
                    .background(WidgetPalette.sage, in: RoundedRectangle(cornerRadius: 12))
                }
            } else {
                VStack(alignment: .leading, spacing: 5) {
                    Label("Up next", systemImage: "calendar")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(WidgetPalette.accent)
                    Text("Nothing else today")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(WidgetPalette.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
                .padding(8)
                .background(WidgetPalette.sage, in: RoundedRectangle(cornerRadius: 12))
            }
        }
    }
}

private struct MealHighlight: View {
    let meal: TodayWidgetData.Meal?
    let title: String

    var body: some View {
        Link(destination: WidgetLink.meals) {
            VStack(alignment: .leading, spacing: 5) {
                Label(title, systemImage: "fork.knife")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(WidgetPalette.clay)
                if let meal {
                    Text(meal.name)
                        .font(.subheadline.weight(.semibold))
                        .lineLimit(3)
                        .privacySensitive()
                } else {
                    Text("No dinner planned")
                        .font(.subheadline)
                        .foregroundStyle(WidgetPalette.secondary)
                }
                Spacer(minLength: 0)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
            .padding(8)
            .background(WidgetPalette.departure, in: RoundedRectangle(cornerRadius: 12))
        }
    }
}

private struct ChoreSummary: View {
    let entry: TodayEntry

    var body: some View {
        if entry.data.choreCount > 0 {
            Link(destination: WidgetLink.today) {
                HStack(spacing: 5) {
                    Image(systemName: "checkmark.circle")
                    Text("\(entry.data.choreCount) chores to do")
                    if entry.data.overdueCount > 0 {
                        Text("· \(entry.data.overdueCount) overdue")
                            .foregroundStyle(WidgetPalette.clay)
                    }
                    Spacer(minLength: 0)
                }
                .font(.caption.weight(.medium))
                .foregroundStyle(WidgetPalette.secondary)
            }
        }
    }
}

private struct EventRow: View {
    let item: TodayWidgetData.Item

    var body: some View {
        Link(destination: WidgetLink.calendar) {
            HStack(spacing: 8) {
                if let start = item.eventStart {
                    Text(start, format: .dateTime.hour().minute())
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(WidgetPalette.accent)
                        .frame(width: 48, alignment: .leading)
                }
                VStack(alignment: .leading, spacing: 1) {
                    Text(item.title).font(.caption.weight(.medium)).lineLimit(1).privacySensitive()
                    if let subtitle = item.subtitle {
                        Text(subtitle).font(.caption2).foregroundStyle(WidgetPalette.secondary).lineLimit(1).privacySensitive()
                    }
                }
                Spacer(minLength: 0)
            }
            .frame(minHeight: 28)
        }
    }
}

private struct ChoreRow: View {
    let item: TodayWidgetData.Item
    var failed = false
    var pending = false

    var body: some View {
        HStack(spacing: 5) {
            Link(destination: WidgetLink.today) {
                HStack(spacing: 7) {
                    Image(systemName: "circle")
                        .font(.body)
                        .foregroundStyle(WidgetPalette.clay)
                    VStack(alignment: .leading, spacing: 1) {
                        Text(item.title).font(.caption.weight(.medium)).lineLimit(1)
                        if let subtitle = item.subtitle {
                            Text(subtitle).font(.caption2).foregroundStyle(WidgetPalette.secondary).lineLimit(1)
                        }
                        if failed {
                            Text("Could not save · still open")
                                .font(.caption2)
                                .foregroundStyle(WidgetPalette.clay)
                        }
                        if pending {
                            Text("Saving…")
                                .font(.caption2)
                                .foregroundStyle(WidgetPalette.secondary)
                        }
                    }
                    Spacer(minLength: 0)
                }
                .contentShape(Rectangle())
            }
            if let choreID = item.choreID {
                Button(intent: CompleteChoreIntent(choreID: choreID)) {
                    Group {
                        if pending {
                            ProgressView()
                                .tint(WidgetPalette.accent)
                        } else {
                            Image(systemName: "checkmark.circle")
                                .font(.title3)
                                .foregroundStyle(WidgetPalette.accent)
                        }
                    }
                    .frame(width: 44, height: 44)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .disabled(pending)
                .accessibilityLabel("Complete \(item.title)")
            }
        }
        .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
    }
}
