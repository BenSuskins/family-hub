import XCTest
@testable import FamilyHub
import FamilyHubKit

@MainActor
final class CalendarViewModelTests: XCTestCase {
    private func makeChore(id: String, name: String, dueDate: String) -> Chore {
        Chore(id: id, name: name, description: "", status: .pending,
              dueDate: dueDate + "T00:00:00Z", assignedToUserID: nil)
    }

    private func makeResponse(chores: [Chore] = [], events: [CalendarEvent] = [], meals: [MealPlan] = []) -> CalendarResponse {
        CalendarResponse(chores: chores, events: events, meals: meals)
    }

    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.calendarResult = .success(makeResponse(chores: [makeChore(id: "1", name: "Clean", dueDate: "2026-03-15")]))
        let viewModel = CalendarViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let response) = viewModel.state else {
            XCTFail("expected loaded state")
            return
        }
        XCTAssertEqual(response.chores.count, 1)
    }

    func testChoresForDayFiltersCorrectly() async {
        let fake = FakeAPIClient()
        fake.calendarResult = .success(makeResponse(chores: [
            makeChore(id: "1", name: "A", dueDate: "2026-03-15"),
            makeChore(id: "2", name: "B", dueDate: "2026-03-20"),
        ]))
        let viewModel = CalendarViewModel(apiClient: fake)
        await viewModel.load()

        let march15 = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        let chores = viewModel.chores(for: march15)

        XCTAssertEqual(chores.count, 1)
        XCTAssertEqual(chores.first?.name, "A")
    }
    func testOpeningADayAndReturningPreservesTheDateAndReusesMonthData() async {
        let fake = FakeAPIClient()
        let today = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        let viewModel = CalendarViewModel(apiClient: fake, now: { today }, preferences: nil)
        await viewModel.load()
        let requests = fake.calendarRequests.count
        viewModel.openDay(today)
        await viewModel.load()
        XCTAssertEqual(viewModel.viewMode, .day)
        viewModel.goBack()
        XCTAssertEqual(viewModel.viewMode, .month)
        XCTAssertEqual(viewModel.selectedDate, today)
        XCTAssertEqual(fake.calendarRequests.count, requests)
    }

    func testRefreshFailureRetainsItemsAndExposesAnError() async {
        let fake = FakeAPIClient()
        let today = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        fake.calendarResult = .success(makeResponse(chores: [makeChore(id: "1", name: "Clean", dueDate: "2026-03-15")]))
        let viewModel = CalendarViewModel(apiClient: fake, now: { today }, preferences: nil)
        await viewModel.load()
        fake.calendarResult = .failure(APIError.offline)
        await viewModel.load(forceRefresh: true)
        XCTAssertEqual(viewModel.chores(for: today).map(\.name), ["Clean"])
        XCTAssertEqual(viewModel.loadError, .offline)
        XCTAssertFalse(viewModel.isLoading)
    }

    func testRemovedViewPreferencesFallBackToMonth() {
        let preferences = UserDefaults(suiteName: "CalendarViewModelTests-" + UUID().uuidString)!
        for mode in ["Multi Day", "List"] {
            preferences.set(mode, forKey: "calendar.view")
            let viewModel = CalendarViewModel(apiClient: FakeAPIClient(), preferences: preferences)
            XCTAssertEqual(viewModel.viewMode, .month)
        }
    }

    func testMonthPrefetchCoversThreeMonthsInEachDirection() async {
        let fake = FakeAPIClient()
        let today = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        fake.calendarsByMonth["2026-05"] = makeResponse(chores: [
            makeChore(id: "may", name: "Clean", dueDate: "2026-05-15")
        ])
        let viewModel = CalendarViewModel(apiClient: fake, now: { today }, preferences: nil)
        await viewModel.load()
        XCTAssertEqual(fake.calendarRequests.first, "2026-03")
        XCTAssertEqual(Set(fake.calendarRequests), Set(["2025-12", "2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06"]))
        viewModel.openDay(viewModel.schedule.adding(.month, value: 2, to: today))
        XCTAssertEqual(viewModel.chores(for: viewModel.selectedDate).map(\.name), ["Clean"])
        fake.beforeFetchCalendar = { _ in
            XCTAssertFalse(viewModel.isLoading, "Cached dates must remain available during prefetch")
        }
        await viewModel.load()
        XCTAssertEqual(fake.calendarRequests.filter { $0 == "2026-05" }.count, 1)
    }

    func testYearNavigationMakesNoCalendarRequests() async {
        let fake = FakeAPIClient()
        let viewModel = CalendarViewModel(apiClient: fake, preferences: nil)
        viewModel.setView(.year)
        await viewModel.load()
        viewModel.move(1)
        await viewModel.load()
        XCTAssertTrue(fake.calendarRequests.isEmpty)
    }

    func testTodayResetsTheDateAndViewPreferencesPersist() {
        let preferences = UserDefaults(suiteName: "CalendarViewModelTests-" + UUID().uuidString)!
        let today = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        let fake = FakeAPIClient()
        let viewModel = CalendarViewModel(apiClient: fake, now: { today }, preferences: preferences)
        viewModel.setView(.week)
        viewModel.setMonthScale(1.5)
        viewModel.setHourHeight(90)
        viewModel.move(1)
        viewModel.goToToday()
        XCTAssertEqual(viewModel.selectedDate, today)
        let reopened = CalendarViewModel(apiClient: fake, now: { today }, preferences: preferences)
        XCTAssertEqual(reopened.viewMode, .week)
        XCTAssertEqual(reopened.monthScale, 1.5)
        XCTAssertEqual(reopened.hourHeight, 90)
        XCTAssertEqual(reopened.selectedDate, today)
    }

    func testLateResponseCannotReplaceTheActivePeriod() async {
        let fake = FakeAPIClient()
        let march = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
        let july = DateComponents(calendar: .current, year: 2026, month: 7, day: 15).date!
        fake.calendarsByMonth = [
            "2026-03": makeResponse(chores: [makeChore(id: "old", name: "March", dueDate: "2026-03-15")]),
            "2026-07": makeResponse(chores: [makeChore(id: "new", name: "July", dueDate: "2026-07-15")])
        ]
        var release: CheckedContinuation<Void, Never>?
        fake.beforeFetchCalendar = { date in
            if APIDate.monthString(date) == "2026-03" {
                await withCheckedContinuation { release = $0 }
            }
        }
        let viewModel = CalendarViewModel(apiClient: fake, now: { march }, preferences: nil)
        let oldLoad = Task { await viewModel.load() }
        while release == nil { await Task.yield() }
        viewModel.selectDate(july)
        await viewModel.load()
        release?.resume()
        await oldLoad.value
        XCTAssertEqual(viewModel.selectedDate, july)
        XCTAssertEqual(viewModel.chores(for: july).map(\.name), ["July"])
        guard case .loaded(let response) = viewModel.state else { return XCTFail("Expected calendar data") }
        XCTAssertEqual(response.chores.map(\.name), ["July"])
    }

    func testTimeZoneChangePreservesTheSelectedCalendarDate() async {
        let fake = FakeAPIClient()
        var london = Calendar(identifier: .gregorian)
        london.timeZone = TimeZone(identifier: "Europe/London")!
        let today = london.date(from: DateComponents(year: 2026, month: 3, day: 15))!
        let viewModel = CalendarViewModel(apiClient: fake, calendar: london, now: { today }, preferences: nil)
        await viewModel.load()
        var pacific = london
        pacific.timeZone = TimeZone(identifier: "America/Los_Angeles")!
        viewModel.updateCalendar(pacific)
        XCTAssertEqual(viewModel.schedule.dayKey(viewModel.selectedDate), "2026-03-15")
        XCTAssertFalse(viewModel.hasLoaded(viewModel.selectedDate))
        await viewModel.load()
        XCTAssertTrue(viewModel.hasLoaded(viewModel.selectedDate))
    }

    func testRefreshUsesTheNewestCopyOfAnEventAcrossCachedMonths() async {
        let fake = FakeAPIClient()
        let today = DateComponents(calendar: .current, year: 2026, month: 3, day: 31).date!
        func event(_ title: String) -> CalendarResponse {
            makeResponse(events: [CalendarEvent(id: "trip", title: title, description: "", location: "",
                                               startTime: today, endTime: today.addingTimeInterval(172800),
                                               allDay: false, color: "")])
        }
        fake.calendarsByMonth = ["2026-03": event("Old name"), "2026-04": event("Old name")]
        let viewModel = CalendarViewModel(apiClient: fake, now: { today }, preferences: nil)
        await viewModel.load()
        fake.calendarsByMonth["2026-04"] = event("New name")
        await viewModel.load(forceRefresh: true)
        XCTAssertEqual(viewModel.events(for: today).map(\.title), ["New name"])
    }

}
