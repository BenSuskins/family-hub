import Foundation
import SwiftUI
import WidgetKit
import FamilyHubKit

/// The family's day on the Home Screen: the agenda, what still needs doing and
/// what's for dinner, in one widget.
///
/// A chores-only widget shipped first and answered half the question — it told
/// you there were four chores while saying nothing about the swimming lesson at
/// six that makes two of them impossible. The day is the unit people actually
/// plan in, so that is the unit the widget shows.
struct TodayWidget: Widget {
    // Shared with the app, which reloads this widget by name.
    static let kind = WidgetKind.today

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: Self.kind, provider: TodayProvider()) { entry in
            TodayView(entry: entry, presentation: .today)
                // Preserve the existing URL when the app's navigation changes.
                .widgetURL(URL(string: "familyhub://today"))
                .containerBackground(for: .widget) { WidgetPalette.background }
        }
        .configurationDisplayName("Today")
        .description("Your family's day: calendar events, chores due, and tonight's meal.")
        .supportedFamilies([
            .systemSmall,
            .systemMedium,
            .systemLarge,
            .accessoryCircular,
            .accessoryRectangular,
            .accessoryInline,
        ])
    }
}

private func focusedFamilyConfiguration(
    kind: String,
    name: String,
    description: String,
    presentation: WidgetPresentation,
    families: [WidgetFamily],
    destination: URL
) -> some WidgetConfiguration {
    StaticConfiguration(kind: kind, provider: TodayProvider()) { entry in
        TodayView(entry: entry, presentation: presentation)
            .widgetURL(destination)
            .containerBackground(for: .widget) { WidgetPalette.background }
    }
    .configurationDisplayName(name)
    .description(description)
    .supportedFamilies(families)
}

struct TonightWidget: Widget {
    var body: some WidgetConfiguration {
        focusedFamilyConfiguration(
            kind: WidgetKind.tonight,
            name: "Tonight",
            description: "See what is for dinner.",
            presentation: .tonight,
            families: [.systemSmall, .accessoryInline],
            destination: WidgetLink.meals
        )
    }
}

struct MealPlanWidget: Widget {
    var body: some WidgetConfiguration {
        focusedFamilyConfiguration(
            kind: WidgetKind.mealPlan,
            name: "Meal plan",
            description: "See today's breakfast, lunch, and dinner.",
            presentation: .mealPlan,
            families: [.systemMedium],
            destination: WidgetLink.meals
        )
    }
}

struct UpNextWidget: Widget {
    var body: some WidgetConfiguration {
        focusedFamilyConfiguration(
            kind: WidgetKind.upNext,
            name: "Up next",
            description: "See the next event and where it is.",
            presentation: .upNext,
            families: [.systemSmall, .accessoryRectangular],
            destination: WidgetLink.calendar
        )
    }
}

struct ChoresWidget: Widget {
    var body: some WidgetConfiguration {
        focusedFamilyConfiguration(
            kind: WidgetKind.chores,
            name: "Chores",
            description: "See outstanding chores and complete them from the widget.",
            presentation: .chores,
            families: [.systemSmall, .systemMedium, .accessoryCircular],
            destination: WidgetLink.today
        )
    }
}
