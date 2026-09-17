import XCTest
@testable import FamilyHubKit

final class TodayWidgetCacheTests: XCTestCase {

    private func makeSnapshot(capturedAt: Date) -> TodayWidgetData {
        TodayWidgetData(
            items: [
                TodayWidgetData.Item(
                    id: "chore-c1",
                    kind: .chore,
                    title: "Bins",
                    subtitle: "Ben",
                    timeLabel: "08:00",
                    choreID: "c1",
                    initials: "B"
                ),
                TodayWidgetData.Item(
                    id: "event-e1",
                    kind: .event,
                    title: "Swimming",
                    subtitle: "Pool",
                    timeLabel: "17:30",
                    colorHex: "3B82F6"
                ),
            ],
            totalItemCount: 2,
            overdueCount: 1,
            dueTodayCount: 2,
            eventCount: 1,
            meals: [TodayWidgetData.Meal(slot: .dinner, name: "Bolognese")],
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

        // nil and "nothing on" are different states: the first sends the
        // widget to the network, the second draws an all-clear.
        XCTAssertNil(TodayWidgetCache.load(from: defaults))
    }

    func testLoadingIsNilWhenThereAreNoSharedDefaults() {
        XCTAssertNil(TodayWidgetCache.load(from: nil))
    }

    // A snapshot written by a build with a different shape must read back as
    // "no cache" rather than throwing or half-decoding: that is what makes a
    // changed snapshot shape a non-event for anyone upgrading.
    func testASnapshotInAnUnrecognisedShapeReadsBackAsNoCache() throws {
        let suite = "uk.co.suskins.familyhub.tests.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            throw XCTSkip("UserDefaults suites are unavailable on this platform")
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        // The chores-only snapshot this widget replaced.
        let legacy = #"{"items":[{"id":"c1","name":"Bins"}],"overdueCount":1,"dueTodayCount":0,"capturedAt":0}"#
        defaults.set(Data(legacy.utf8), forKey: SharedContainer.Key.todaySnapshot)

        XCTAssertNil(TodayWidgetCache.load(from: defaults))
    }
}

extension TodayWidgetCacheTests {

    func testInvalidatingKeepsTheDayButMakesItStale() throws {
        let suite = "uk.co.suskins.familyhub.tests.\(UUID().uuidString)"
        guard let defaults = UserDefaults(suiteName: suite) else {
            throw XCTSkip("UserDefaults suites are unavailable on this platform")
        }
        defer { defaults.removePersistentDomain(forName: suite) }

        TodayWidgetCache.save(makeSnapshot(capturedAt: Date()), to: defaults)

        TodayWidgetCache.invalidate(in: defaults)

        let reloaded = try XCTUnwrap(TodayWidgetCache.load(from: defaults))
        // The day survives as an offline fallback; only the freshness goes.
        XCTAssertEqual(reloaded.items.map(\.id), ["chore-c1", "event-e1"])
        XCTAssertEqual(reloaded.dueTodayCount, 2)
        XCTAssertEqual(reloaded.meals.map(\.name), ["Bolognese"])
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
