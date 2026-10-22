import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class CommuteViewModelTests: XCTestCase {
    func testDisabledSettingsMakeNoTrainRequests() async {
        let fake = FakeAPIClient()
        let model = CommuteViewModel(apiClient: fake, store: makeStore())
        model.configure(userID: "one")
        await model.refresh(force: true)
        XCTAssertTrue(fake.commuteRequests.isEmpty)
        XCTAssertTrue(model.activeDirections.isEmpty)
    }

    func testShowsTrainsOnlyInsideTheScheduleAndExpiresOldResults() async {
        let fake = FakeAPIClient()
        fake.serverIdentifier = "https://hub.example"
        let store = makeStore()
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        store.save(preferences, server: fake.serverIdentifier!, userID: "one")
        var now = ISO8601DateFormatter().date(from: "2026-10-19T06:30:00Z")!
        fake.commuteResult = .success(board(at: now))
        let model = CommuteViewModel(apiClient: fake, store: store, now: { now })
        model.configure(userID: "one")
        await model.refresh()
        guard case .loaded(let board) = model.states[.toWork] else { return XCTFail("Expected a train board") }
        XCTAssertEqual(board.trains.first?.departure.display, "07:42")
        now = now.addingTimeInterval(30)
        model.tick()
        guard case .failed = model.states[.toWork] else { return XCTFail("Expired trains must not stay live") }
        now = ISO8601DateFormatter().date(from: "2026-10-19T08:30:00Z")!
        await model.refresh(force: true)
        XCTAssertTrue(model.activeDirections.isEmpty)
        XCTAssertEqual(fake.commuteRequests, [.toWork])
    }

    func testAccountChangeDiscardsAnInFlightBoard() async {
        let fake = FakeAPIClient()
        fake.serverIdentifier = "https://hub.example"
        let store = makeStore()
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        store.save(preferences, server: fake.serverIdentifier!, userID: "one")
        let now = ISO8601DateFormatter().date(from: "2026-10-19T06:30:00Z")!
        fake.commuteResult = .success(board(at: now))
        var continuation: CheckedContinuation<Void, Never>?
        fake.beforeFetchCommute = { await withCheckedContinuation { continuation = $0 } }
        let model = CommuteViewModel(apiClient: fake, store: store, now: { now })
        model.configure(userID: "one")
        let request = Task { await model.refresh() }
        while continuation == nil { await Task.yield() }
        model.configure(userID: "two")
        continuation?.resume()
        await request.value
        XCTAssertTrue(model.activeDirections.isEmpty)
        XCTAssertTrue(model.states.isEmpty)
    }

    func testLeavingHomeCancelsTheRequestAndIgnoresItsResult() async {
        let fake = FakeAPIClient()
        let store = makeStore()
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        store.save(preferences, server: "demo", userID: "one")
        let now = ISO8601DateFormatter().date(from: "2026-10-19T06:30:00Z")!
        fake.commuteResult = .success(board(at: now))
        var continuation: CheckedContinuation<Void, Never>?
        var cancelled = false
        fake.beforeFetchCommute = {
            await withCheckedContinuation { continuation = $0 }
            cancelled = Task.isCancelled
        }
        let model = CommuteViewModel(apiClient: fake, store: store, now: { now })
        model.configure(userID: "one")
        let request = Task { await model.refresh() }
        while continuation == nil { await Task.yield() }
        model.stop()
        continuation?.resume()
        await request.value
        XCTAssertTrue(cancelled)
        XCTAssertTrue(model.states.isEmpty)
    }

    func testFailedRequestCanBeRetriedWithoutWaitingThirtySeconds() async {
        let fake = FakeAPIClient()
        let store = makeStore()
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        store.save(preferences, server: "demo", userID: "one")
        let now = ISO8601DateFormatter().date(from: "2026-10-19T06:30:00Z")!
        let model = CommuteViewModel(apiClient: fake, store: store, now: { now })
        model.configure(userID: "one")
        await model.refresh()
        guard case .failed = model.states[.toWork] else { return XCTFail("Expected unavailable trains") }
        fake.commuteResult = .success(board(at: now))
        await model.retry(.toWork)
        guard case .loaded = model.states[.toWork] else { return XCTFail("Expected fresh trains after retry") }
    }

    func testRefreshUsesTheProviderFetchTimeForItsThirtySecondLimit() async {
        let fake = FakeAPIClient()
        let store = makeStore()
        var preferences = CommutePreferences()
        preferences.toWork.enabled = true
        store.save(preferences, server: "demo", userID: "one")
        var now = ISO8601DateFormatter().date(from: "2026-10-19T06:30:00Z")!
        fake.commuteResult = .success(board(at: now.addingTimeInterval(-20)))
        let model = CommuteViewModel(apiClient: fake, store: store, now: { now })
        model.configure(userID: "one")
        await model.refresh()
        now = now.addingTimeInterval(9)
        await model.refresh()
        XCTAssertEqual(fake.commuteRequests, [.toWork])
        now = now.addingTimeInterval(1)
        fake.commuteResult = .success(board(at: now))
        await model.refresh()
        XCTAssertEqual(fake.commuteRequests, [.toWork, .toWork])
        guard case .loaded = model.states[.toWork] else { return XCTFail("Expected a refreshed board") }
    }

    private func makeStore() -> CommutePreferencesStore {
        let suite = "commute-model.\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: suite)!
        addTeardownBlock { defaults.removePersistentDomain(forName: suite) }
        return CommutePreferencesStore(defaults: defaults)
    }

    private func board(at date: Date) -> CommuteBoard {
        let timestamp = ISO8601DateFormatter().string(from: date)
        return CommuteBoard(direction: .toWork, origin: CommuteStation(code: "KEL", name: "Kelvedon"),
                            destination: CommuteStation(code: "LST", name: "London Liverpool Street"),
                            generatedAt: timestamp, fetchedAt: timestamp,
                            trains: [CommuteTrain(serviceID: "first", status: "on_time", scheduledDeparture: "07:42", expectedDeparture: "07:42", expectedArrival: "08:34")])
    }
}
