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
            TodayView(entry: entry)
                // The app has no Chores tab — chores and the day's agenda both
                // live on Home — so that is where a tap on the widget lands.
                // Individual rows override this with their own destination.
                .widgetURL(URL(string: "familyhub://today"))
                .containerBackground(.fill.tertiary, for: .widget)
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
