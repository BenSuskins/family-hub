import Foundation
import AppIntents
import WidgetKit
import FamilyHubKit

/// Ticks a chore off straight from the widget.
///
/// Not discoverable in Shortcuts: it takes a raw chore id, which is meaningless
/// to type by hand, and the app has no Shortcuts surface yet to make it
/// coherent. It exists solely to back the widget's tick buttons.
struct CompleteChoreIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Complete Chore" }
    nonisolated static var description: IntentDescription {
        IntentDescription("Marks a chore as done from the Today's Chores widget.")
    }
    nonisolated static var isDiscoverable: Bool { false }

    @Parameter(title: "Chore")
    var choreID: String

    init() {}

    init(choreID: String) {
        self.choreID = choreID
    }

    func perform() async throws -> some IntentResult {
        guard let client = WidgetEnvironment.client() else {
            // Signed out. The widget is already showing a "open the app"
            // state, so there is nothing useful to throw at the user here.
            return .result()
        }

        try await client.completeChore(id: choreID)

        // Drop the chore from the cached snapshot before asking for a reload.
        // Re-fetching the dashboard first would make the tick feel sluggish,
        // and the next scheduled refresh will reconcile anyway.
        if let cached = TodayWidgetCache.load() {
            TodayWidgetCache.save(cached.removing(choreID: choreID))
        }
        WidgetCenter.shared.reloadTimelines(ofKind: WidgetKind.todayChores)

        return .result()
    }
}
