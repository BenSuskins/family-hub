import Foundation
import WidgetKit
import FamilyHubKit

/// Supplies entries for the Today's Chores widget.
///
/// Reads the app's published snapshot first and only goes to the network when
/// that is stale. WidgetKit allows a widget a few dozen refreshes a day, while
/// the app loads the dashboard every time someone opens it — so the cheapest
/// and freshest source is usually the app's own last load, not a request from
/// here.
struct TodayProvider: TimelineProvider {

    func placeholder(in context: Context) -> TodayEntry {
        .preview
    }

    func getSnapshot(in context: Context, completion: @escaping (TodayEntry) -> Void) {
        if context.isPreview {
            completion(.preview)
            return
        }
        Task {
            completion(await currentEntry())
        }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<TodayEntry>) -> Void) {
        Task {
            let entry = await currentEntry()
            completion(Timeline(entries: [entry], policy: .after(entry.nextRefresh)))
        }
    }

    private func currentEntry() async -> TodayEntry {
        let now = Date()

        guard SharedContainer.isConfigured else {
            return TodayEntry(date: now, data: .blank(at: now), status: .signedOut)
        }

        let cached = TodayWidgetCache.load()
        if let cached, TodayWidgetCache.isFresh(cached, now: now) {
            return TodayEntry(date: now, data: cached, status: .ok)
        }

        if let fetched = await WidgetEnvironment.fetchSnapshot() {
            TodayWidgetCache.save(fetched)
            return TodayEntry(date: now, data: fetched, status: .ok)
        }

        // Unreachable. Routine for a self-hosted hub away from the home
        // network, so show what we last knew and let the view stamp its age.
        if let cached {
            return TodayEntry(date: now, data: cached, status: .cached)
        }
        return TodayEntry(date: now, data: .blank(at: now), status: .unavailable)
    }
}
