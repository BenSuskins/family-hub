import Foundation
import WidgetKit
import FamilyHubKit

/// Supplies entries for the Today widget.
///
/// Reads the app's published snapshot first and only goes to the network when
/// that is stale. WidgetKit allows a widget a few dozen refreshes a day, while
/// the app loads the dashboard and today's calendar every time someone opens
/// Home — so the cheapest and freshest source is usually the app's own last
/// load, not three requests from here.
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
            let safetyRefresh = entry.nextRefresh
            let boundaryLimit = min(safetyRefresh, entry.date.addingTimeInterval(4 * 60 * 60))
            let nextBoundary = entry.data.eventBoundaries(
                after: entry.date,
                before: boundaryLimit.addingTimeInterval(1)
            ).first
            let timelineEntries = nextBoundary.map {
                [entry, TodayEntry(
                    date: $0,
                    data: entry.data,
                    status: entry.status,
                    failedChoreID: entry.failedChoreID,
                    pendingChoreID: entry.pendingChoreID
                )]
            } ?? [entry]
            completion(Timeline(entries: timelineEntries, policy: .after(nextBoundary ?? safetyRefresh)))
        }
    }

    private func currentEntry() async -> TodayEntry {
        let now = Date()
        let failedChoreID = TodayWidgetCache.failedChoreID()
        let pendingChoreID = TodayWidgetCache.pendingChoreID(now: now)

        guard SharedContainer.isConfigured else {
            return TodayEntry(date: now, data: .blank(at: now), status: .signedOut, failedChoreID: failedChoreID, pendingChoreID: pendingChoreID)
        }

        // A snapshot written by a build with a different shape won't decode
        // and reads back as nil, which is the same as no cache: fetch.
        let cached = TodayWidgetCache.load()
        if let cached, TodayWidgetCache.isFresh(cached, now: now) {
            return TodayEntry(date: now, data: cached, status: .ok, failedChoreID: failedChoreID, pendingChoreID: pendingChoreID)
        }

        if let fetched = await WidgetEnvironment.fetchSnapshot() {
            TodayWidgetCache.save(fetched)
            return TodayEntry(date: now, data: fetched, status: .ok, failedChoreID: failedChoreID, pendingChoreID: pendingChoreID)
        }

        // Unreachable. Routine for a self-hosted hub away from the home
        // network, so show what we last knew and let the view stamp its age.
        if let cached {
            return TodayEntry(date: now, data: cached, status: .cached, failedChoreID: failedChoreID, pendingChoreID: pendingChoreID)
        }
        return TodayEntry(date: now, data: .blank(at: now), status: .unavailable, failedChoreID: failedChoreID, pendingChoreID: pendingChoreID)
    }
}
