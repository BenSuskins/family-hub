import Foundation
import SwiftUI
import WidgetKit
import FamilyHubKit

struct TodayChoresWidget: Widget {
    // Shared with the app, which reloads this widget by name.
    static let kind = WidgetKind.todayChores

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: Self.kind, provider: TodayProvider()) { entry in
            TodayChoresView(entry: entry)
                // The app has no Chores tab — chores live on Home — so that is
                // where a tap lands.
                .widgetURL(URL(string: "familyhub://today"))
                .containerBackground(.fill.tertiary, for: .widget)
        }
        .configurationDisplayName("Today's Chores")
        .description("Chores due today and anything overdue, with a tick to complete them.")
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
