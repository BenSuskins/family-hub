import XCTest
@testable import FamilyHubKit

final class TodayWidgetCacheTests: XCTestCase {

    private func makeSnapshot(capturedAt: Date) -> TodayWidgetData {
        TodayWidgetData(
            items: [TodayWidgetData.Item(id: "c1", name: "Bins", assigneeName: "Ben", assigneeInitials: "B")],
            overdueCount: 1,
            dueTodayCount: 2,
            capturedAt: capturedAt
        )
    }

    // MARK: - Freshness

    func testARecentSnapshotIsFresh() {
        let now = Date()
        let snapshot = makeSnapshot(capturedAt: now.addingTimeInterval(-60))

        XCTAssertTrue(TodayWidgetCache.isFresh(snapshot, now: now))
    }

    func testASnapshotPastTheWindowIsStale() {
        let now = Date()
        let snapshot = makeSnapshot(capturedAt: now.addingTimeInterval(-TodayWidgetCache.freshFor - 1))

        XCTAssertFalse(TodayWidgetCache.isFresh(snapshot, now: now))
    }

    // A snapshot stamped in the future means the clock moved backwards (a
    // timezone change, or NTP correcting the device). Refetch rather than
    // trusting it for the whole window.
    func testAFutureSnapshotIsNotTreatedAsFresh() {
        let now = Date()
        let snapshot = makeSnapshot(capturedAt: now.addingTimeInterval(120))

        XCTAssertFalse(TodayWidgetCache.isFresh(snapshot, now: now))
    }

    func testABlankSnapshotIsStaleSoTheWidgetFetches() {
        XCTAssertFalse(TodayWidgetCache.isFresh(TodayWidgetData.blank(at: .distantPast)))
    }

    // MARK: - Storage

    func testSaveLoadAndClearRoundTripThroughDefaults() throws {
        let suite = "uk.co.suskins.familyhub.tests.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            throw XCTSkip("UserDefaults suites are unavailable on this platform")
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        let snapshot = makeSnapshot(capturedAt: Date())
        TodayWidgetCache.save(snapshot, to: defaults)

        XCTAssertEqual(TodayWidgetCache.load(from: defaults), snapshot)

        TodayWidgetCache.clear(from: defaults)
        XCTAssertNil(TodayWidgetCache.load(from: defaults))
    }

    func testLoadingFromEmptyDefaultsYieldsNilRatherThanABlankSnapshot() throws {
        let suite = "uk.co.suskins.familyhub.tests.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            throw XCTSkip("UserDefaults suites are unavailable on this platform")
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        // nil and "nothing to do" are different states: the first sends the
        // widget to the network, the second draws an all-clear.
        XCTAssertNil(TodayWidgetCache.load(from: defaults))
    }

    func testLoadingIsNilWhenThereAreNoSharedDefaults() {
        XCTAssertNil(TodayWidgetCache.load(from: nil))
    }
}

extension TodayWidgetCacheTests {

    func testInvalidatingKeepsTheChoresButMakesThemStale() throws {
        let suite = "uk.co.suskins.familyhub.tests.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            throw XCTSkip("UserDefaults suites are unavailable on this platform")
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        let snapshot = TodayWidgetData(
            items: [TodayWidgetData.Item(id: "c1", name: "Bins")],
            overdueCount: 0,
            dueTodayCount: 1,
            capturedAt: Date()
        )
        TodayWidgetCache.save(snapshot, to: defaults)

        TodayWidgetCache.invalidate(in: defaults)

        let reloaded = try XCTUnwrap(TodayWidgetCache.load(from: defaults))
        // The chores survive as an offline fallback; only the freshness goes.
        XCTAssertEqual(reloaded.items.map(\.id), ["c1"])
        XCTAssertEqual(reloaded.dueTodayCount, 1)
        XCTAssertFalse(TodayWidgetCache.isFresh(reloaded))
    }

    func testInvalidatingAnEmptyCacheDoesNotInventASnapshot() throws {
        let suite = "uk.co.suskins.familyhub.tests.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            throw XCTSkip("UserDefaults suites are unavailable on this platform")
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        TodayWidgetCache.invalidate(in: defaults)

        XCTAssertNil(TodayWidgetCache.load(from: defaults))
    }
}
