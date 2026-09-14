import XCTest
@testable import FamilyHubKit

final class TodayWidgetDataTests: XCTestCase {

    private func makeChore(
        id: String,
        name: String = "Chore",
        status: ChoreStatus = .pending,
        dueDate: String? = nil,
        dueTime: String? = nil,
        assignedTo: String? = nil
    ) -> Chore {
        Chore(
            id: id,
            name: name,
            description: "",
            status: status,
            dueDate: dueDate,
            assignedToUserID: assignedTo,
            dueTime: dueTime
        )
    }

    private func makeStats(overdue: [Chore] = [], dueToday: [Chore] = []) -> DashboardStats {
        DashboardStats(
            choresDueToday: dueToday.count,
            choresOverdue: overdue.count,
            choresDueTodayList: dueToday,
            choresOverdueList: overdue,
            mealsThisWeek: 0,
            todayMeals: []
        )
    }

    private let ben = User(id: "u1", name: "Ben Suskins", email: "ben@example.com", avatarURL: "", role: "admin")

    // MARK: - Ordering

    func testOverdueChoresLeadAndEachGroupIsEarliestFirst() {
        let stats = makeStats(
            overdue: [
                makeChore(id: "late-b", dueDate: "2026-09-11"),
                makeChore(id: "late-a", dueDate: "2026-09-09"),
            ],
            dueToday: [
                makeChore(id: "evening", dueDate: "2026-09-14T18:00:00Z"),
                makeChore(id: "morning", dueDate: "2026-09-14T07:15:00Z"),
            ]
        )

        let data = TodayWidgetData.from(stats: stats, users: [:])

        XCTAssertEqual(data.items.map(\.id), ["late-a", "late-b", "morning", "evening"])
        XCTAssertEqual(data.items.map(\.isOverdue), [true, true, false, false])
    }

    // `sorted(by:)` is not stable, and a recurring set can put several chores
    // on the same midnight, so the order has to be settled by something. Rows
    // shuffling between refreshes would look like a bug to the user.
    func testChoresSharingADueDateAreOrderedDeterministically() {
        let stats = makeStats(dueToday: [
            makeChore(id: "c", dueDate: "2026-09-14"),
            makeChore(id: "a", dueDate: "2026-09-14"),
            makeChore(id: "b", dueDate: "2026-09-14"),
        ])

        let data = TodayWidgetData.from(stats: stats, users: [:])

        XCTAssertEqual(data.items.map(\.id), ["a", "b", "c"])
    }

    func testUndatedChoresSortLastWithinTheirGroup() {
        let stats = makeStats(dueToday: [
            makeChore(id: "undated", dueDate: nil),
            makeChore(id: "dated", dueDate: "2026-09-14"),
        ])

        let data = TodayWidgetData.from(stats: stats, users: [:])

        XCTAssertEqual(data.items.map(\.id), ["dated", "undated"])
    }

    // MARK: - Counts

    func testCountsComeFromTheFullListsNotTheCappedItems() {
        let overdue = (1...6).map { makeChore(id: "o\($0)", dueDate: "2026-09-0\($0)") }
        let dueToday = (1...5).map { makeChore(id: "t\($0)", dueDate: "2026-09-14") }

        let data = TodayWidgetData.from(stats: makeStats(overdue: overdue, dueToday: dueToday), users: [:])

        XCTAssertEqual(data.items.count, TodayWidgetData.maxItems)
        XCTAssertEqual(data.overdueCount, 6)
        XCTAssertEqual(data.dueTodayCount, 5)
        XCTAssertEqual(data.totalCount, 11)
        XCTAssertEqual(data.hiddenCount, 11 - TodayWidgetData.maxItems)
        XCTAssertFalse(data.isClear)
    }

    func testNothingOutstandingIsClear() {
        let data = TodayWidgetData.from(stats: makeStats(), users: [:])

        XCTAssertTrue(data.isClear)
        XCTAssertEqual(data.hiddenCount, 0)
        XCTAssertTrue(data.items.isEmpty)
    }

    // MARK: - Assignee flattening

    func testAssigneeNameAndInitialsAreResolvedAtBuildTime() {
        let stats = makeStats(dueToday: [makeChore(id: "c1", dueDate: "2026-09-14", assignedTo: "u1")])

        let data = TodayWidgetData.from(stats: stats, users: ["u1": ben])

        XCTAssertEqual(data.items.first?.assigneeName, "Ben Suskins")
        XCTAssertEqual(data.items.first?.assigneeInitials, "BS")
    }

    func testUnknownOrMissingAssigneeLeavesTheFieldsEmpty() {
        let stats = makeStats(dueToday: [
            makeChore(id: "c1", dueDate: "2026-09-14", assignedTo: "ghost"),
            makeChore(id: "c2", dueDate: "2026-09-14", assignedTo: nil),
        ])

        let data = TodayWidgetData.from(stats: stats, users: ["u1": ben])

        XCTAssertEqual(data.items.compactMap(\.assigneeName), [])
        XCTAssertEqual(data.items.compactMap(\.assigneeInitials), [])
    }

    // MARK: - Detail line

    func testDueTodayShowsItsDueTimeAndOverdueShowsItsDate() {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", dueDate: "2026-09-09", dueTime: "18:30")],
            dueToday: [makeChore(id: "t1", dueDate: "2026-09-14", dueTime: "07:15")]
        )

        let data = TodayWidgetData.from(stats: stats, users: [:])

        // Overdue: how long ago matters more than the time of day, so the
        // detail is the due date. Its exact wording is locale-dependent, so
        // only its presence is asserted here.
        XCTAssertNotNil(data.items.first?.detail)
        XCTAssertNotEqual(data.items.first?.detail, "18:30")
        XCTAssertEqual(data.items.last?.detail, "07:15")
    }

    func testAChoreWithNoDueTimeHasNoDetail() {
        let stats = makeStats(dueToday: [makeChore(id: "t1", dueDate: "2026-09-14", dueTime: nil)])

        let data = TodayWidgetData.from(stats: stats, users: [:])

        XCTAssertNil(data.items.first?.detail)
    }

    // MARK: - Optimistic removal

    func testRemovingADueTodayChoreDropsItAndItsCount() {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", dueDate: "2026-09-09")],
            dueToday: [makeChore(id: "t1", dueDate: "2026-09-14")]
        )
        let data = TodayWidgetData.from(stats: stats, users: [:])

        let after = data.removing(choreID: "t1")

        XCTAssertEqual(after.items.map(\.id), ["o1"])
        XCTAssertEqual(after.overdueCount, 1)
        XCTAssertEqual(after.dueTodayCount, 0)
    }

    func testRemovingAnOverdueChoreDecrementsTheOverdueCount() {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", dueDate: "2026-09-09")],
            dueToday: [makeChore(id: "t1", dueDate: "2026-09-14")]
        )
        let data = TodayWidgetData.from(stats: stats, users: [:])

        let after = data.removing(choreID: "o1")

        XCTAssertEqual(after.items.map(\.id), ["t1"])
        XCTAssertEqual(after.overdueCount, 0)
        XCTAssertEqual(after.dueTodayCount, 1)
    }

    // A double-tap on the widget, or a tick for a chore that only ever existed
    // in the uncapped tail, must not drive a count below zero.
    func testRemovingAnUnknownChoreIsANoOp() {
        let data = TodayWidgetData.from(
            stats: makeStats(dueToday: [makeChore(id: "t1", dueDate: "2026-09-14")]),
            users: [:]
        )

        XCTAssertEqual(data.removing(choreID: "nope"), data)
        XCTAssertEqual(data.removing(choreID: "t1").removing(choreID: "t1").dueTodayCount, 0)
    }

    // MARK: - Encoding

    func testSnapshotSurvivesAJSONRoundTrip() throws {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", name: "Bins", dueDate: "2026-09-09", assignedTo: "u1")],
            dueToday: [makeChore(id: "t1", name: "Hoovering", dueDate: "2026-09-14", dueTime: "18:00")]
        )
        let original = TodayWidgetData.from(stats: stats, users: ["u1": ben])

        let encoded = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(TodayWidgetData.self, from: encoded)

        XCTAssertEqual(decoded, original)
    }
}

extension TodayWidgetDataTests {

    func testMarkedStaleKeepsEverythingButTheTimestamp() {
        let original = TodayWidgetData.from(
            stats: makeStats(dueToday: [makeChore(id: "t1", dueDate: "2026-09-14")]),
            users: [:]
        )

        let stale = original.markedStale()

        XCTAssertEqual(stale.items, original.items)
        XCTAssertEqual(stale.overdueCount, original.overdueCount)
        XCTAssertEqual(stale.dueTodayCount, original.dueTodayCount)
        XCTAssertEqual(stale.capturedAt, .distantPast)
    }
}
