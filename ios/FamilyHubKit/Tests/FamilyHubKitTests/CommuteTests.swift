import XCTest
@testable import FamilyHubKit

final class CommuteTests: XCTestCase {
    @MainActor
    func testScheduleUsesLondonTimeAndExcludesTheEnd() async {
        let schedule = CommuteSchedule(enabled: true, weekdays: [2], start: "06:30", end: "09:30")
        XCTAssertFalse(schedule.contains(date("2026-10-19T05:29:00Z")))
        XCTAssertTrue(schedule.contains(date("2026-10-19T05:30:00Z")))
        XCTAssertFalse(schedule.contains(date("2026-10-19T08:30:00Z")))
        XCTAssertFalse(schedule.contains(date("2026-10-20T06:30:00Z")))
    }

    @MainActor
    func testOvernightScheduleBelongsToItsStartDay() async {
        let schedule = CommuteSchedule(enabled: true, weekdays: [6], start: "23:00", end: "02:00")
        XCTAssertTrue(schedule.contains(date("2026-10-23T22:00:00Z")))
        XCTAssertTrue(schedule.contains(date("2026-10-24T00:59:00Z")))
        XCTAssertFalse(schedule.contains(date("2026-10-24T01:00:00Z")))
        XCTAssertFalse(schedule.contains(date("2026-10-24T22:00:00Z")))
    }

    @MainActor
    func testScheduleFollowsBothDaylightSavingTransitions() async {
        let schedule = CommuteSchedule(enabled: true, weekdays: [1], start: "01:00", end: "02:00")
        XCTAssertTrue(schedule.contains(date("2026-10-25T00:30:00Z")))
        XCTAssertTrue(schedule.contains(date("2026-10-25T01:30:00Z")))
        XCTAssertFalse(schedule.contains(date("2026-10-25T02:00:00Z")))
        XCTAssertFalse(schedule.contains(date("2026-03-29T01:30:00Z")))
    }

    @MainActor
    func testDisabledAndInvalidSchedulesNeverShowTrains() async {
        for schedule in [
            CommuteSchedule(start: "06:30", end: "09:30"),
            CommuteSchedule(enabled: true, weekdays: [], start: "06:30", end: "09:30"),
            CommuteSchedule(enabled: true, start: "09:30", end: "09:30"),
            CommuteSchedule(enabled: true, start: "25:00", end: "09:30"),
        ] {
            XCTAssertFalse(schedule.contains(date("2026-10-19T06:30:00Z")))
        }
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        preferences.toHome = preferences.toWork
        XCTAssertEqual(preferences.activeDirections(at: date("2026-10-19T06:30:00Z")), [.toWork, .toHome])
    }

    @MainActor
    func testPreferencesPersistPerUserAndCanonicalServerAndRejectInvalidEdits() async throws {
        let suite = "commute.\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: suite)!
        defer { defaults.removePersistentDomain(forName: suite) }
        let store = CommutePreferencesStore(defaults: defaults)
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        XCTAssertTrue(store.save(preferences, server: "https://HUB.example:443/", userID: "one"))
        let reloaded = CommutePreferencesStore(defaults: defaults)
        XCTAssertEqual(reloaded.load(server: "https://hub.example", userID: "one"), preferences)
        XCTAssertEqual(reloaded.load(server: "https://hub.example", userID: "two"), CommutePreferences())
        XCTAssertEqual(reloaded.load(server: "https://other.example", userID: "one"), CommutePreferences())
        preferences.toWork.end = preferences.toWork.start
        XCTAssertFalse(store.save(preferences, server: "https://hub.example", userID: "one"))
        XCTAssertEqual(reloaded.load(server: "https://hub.example", userID: "one").toWork.end, "09:30")
    }

    @MainActor
    func testChangedTimesKeepTheOriginalAndUnknownEstimatesDoNotInventDuration() async throws {
        let train = CommuteTrain(serviceID: "late", status: "delayed", scheduledDeparture: "08:03", expectedDeparture: "08:08", scheduledArrival: "08:58", expectedArrival: "09:03")
        XCTAssertEqual(train.departure.original, "08:03")
        XCTAssertEqual(train.departure.display, "08:08")
        XCTAssertEqual(train.arrival.original, "08:58")
        XCTAssertEqual(train.durationMinutes, 55)
        let unknown = CommuteTrain(serviceID: "unknown", status: "unknown", scheduledDeparture: "08:03", scheduledArrival: "08:58")
        XCTAssertNil(unknown.departure.original)
        XCTAssertEqual(unknown.departure.display, "08:03")
        XCTAssertNil(unknown.durationMinutes)
        let cancelled = CommuteTrain(serviceID: "cancelled", status: "cancelled", scheduledDeparture: "08:03", expectedDeparture: "08:08")
        XCTAssertNil(cancelled.departure.original)
        XCTAssertEqual(cancelled.departure.display, "08:03")
        XCTAssertNil(cancelled.durationMinutes)
        let midnight = CommuteTrain(serviceID: "night", status: "on_time", expectedDeparture: "23:45", expectedArrival: "00:38")
        XCTAssertEqual(midnight.durationMinutes, 53)
    }

    private func date(_ value: String) -> Date {
        ISO8601DateFormatter().date(from: value)!
    }
}
