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

    private func makeEvent(
        id: String,
        title: String = "Event",
        location: String = "",
        description: String = "",
        at hour: Int = 12,
        minute: Int = 0,
        allDay: Bool = false,
        color: String = "3B82F6"
    ) -> CalendarEvent {
        // Built in the local calendar, because that is what the merge orders by
        // — a fixed UTC instant would land in a different hour per runner.
        let start = Calendar.current.date(
            bySettingHour: hour, minute: minute, second: 0, of: Date()
        ) ?? Date()
        return CalendarEvent(
            id: id,
            title: title,
            description: description,
            location: location,
            startTime: start,
            endTime: start.addingTimeInterval(3600),
            allDay: allDay,
            color: color
        )
    }

    private func makeStats(
        overdue: [Chore] = [],
        dueToday: [Chore] = [],
        meals: [MealPlan] = []
    ) -> DashboardStats {
        DashboardStats(
            choresDueToday: dueToday.count,
            choresOverdue: overdue.count,
            choresDueTodayList: dueToday,
            choresOverdueList: overdue,
            mealsThisWeek: meals.count,
            todayMeals: meals
        )
    }

    private func makeMeal(_ type: String, _ name: String) -> MealPlan {
        MealPlan(date: "2026-09-17", mealType: type, name: name, notes: "", recipeID: nil)
    }

    private let ben = User(id: "u1", name: "Ben Suskins", email: "ben@example.com", avatarURL: "", role: "admin")

    // MARK: - Ordering

    // The whole point of the redesign: one list, in the order the day happens.
    func testTheDayIsOverdueThenAllDayThenTimedThenUntimed() {
        let stats = makeStats(
            overdue: [makeChore(id: "late", dueDate: "2026-09-11")],
            dueToday: [
                makeChore(id: "whenever", name: "Whenever"),
                makeChore(id: "evening", name: "Evening", dueDate: "2026-09-17", dueTime: "19:00"),
            ]
        )
        let events = [
            makeEvent(id: "morning", title: "Morning", at: 9),
            makeEvent(id: "birthday", title: "Birthday", allDay: true),
        ]

        let data = TodayWidgetData.from(stats: stats, events: events)

        XCTAssertEqual(
            data.items.map(\.id),
            ["chore-late", "event-birthday", "event-morning", "chore-evening", "chore-whenever"]
        )
    }

    func testTimedChoresAndEventsInterleaveByTimeOfDay() {
        let stats = makeStats(dueToday: [
            makeChore(id: "bins", name: "Bins", dueTime: "08:00"),
            makeChore(id: "dinner-prep", name: "Dinner prep", dueTime: "17:00"),
        ])
        let events = [
            makeEvent(id: "standup", title: "Standup", at: 9, minute: 30),
            makeEvent(id: "swim", title: "Swimming", at: 16),
        ]

        let data = TodayWidgetData.from(stats: stats, events: events)

        XCTAssertEqual(
            data.items.map(\.id),
            ["chore-bins", "event-standup", "event-swim", "chore-dinner-prep"]
        )
    }

    // Somewhere you have to be outranks something you could do whenever.
    func testAnEventBeatsAChoreAtTheSameMinute() {
        let stats = makeStats(dueToday: [makeChore(id: "c", name: "Chore", dueTime: "18:00")])
        let events = [makeEvent(id: "e", title: "Event", at: 18)]

        let data = TodayWidgetData.from(stats: stats, events: events)

        XCTAssertEqual(data.items.map(\.id), ["event-e", "chore-c"])
    }

    func testOverdueChoresAreEarliestFirst() {
        let stats = makeStats(overdue: [
            makeChore(id: "late-b", dueDate: "2026-09-11"),
            makeChore(id: "late-a", dueDate: "2026-09-09"),
            makeChore(id: "undated"),
        ])

        let data = TodayWidgetData.from(stats: stats)

        // Undated last, so it never displaces one with a real deadline.
        XCTAssertEqual(data.items.map(\.id), ["chore-late-a", "chore-late-b", "chore-undated"])
        XCTAssertEqual(data.items.map(\.isOverdue), [true, true, true])
    }

    // `sorted(by:)` is not stable, and a recurring set can put several chores
    // on the same minute, so the order has to be settled by something. Rows
    // shuffling between refreshes would look like a bug to the user.
    func testRowsSharingAMinuteAreOrderedDeterministically() {
        let stats = makeStats(dueToday: [
            makeChore(id: "3", name: "Cleaning", dueTime: "09:00"),
            makeChore(id: "1", name: "Bins", dueTime: "09:00"),
            makeChore(id: "2", name: "Ironing", dueTime: "09:00"),
        ])

        let data = TodayWidgetData.from(stats: stats)

        XCTAssertEqual(data.items.map(\.title), ["Bins", "Cleaning", "Ironing"])
    }

    // MARK: - Row kinds

    func testAChoreRowCarriesItsIDForTheTickButtonAndAnEventRowDoesNot() {
        let stats = makeStats(dueToday: [makeChore(id: "c1", dueTime: "10:00")])

        let data = TodayWidgetData.from(stats: stats, events: [makeEvent(id: "e1", at: 11)])

        XCTAssertEqual(data.items.first?.kind, .chore)
        XCTAssertEqual(data.items.first?.choreID, "c1")
        XCTAssertEqual(data.items.last?.kind, .event)
        XCTAssertNil(data.items.last?.choreID)
    }

    // Namespacing matters: the two ids collide often enough in practice, and a
    // `ForEach` over duplicates silently drops a row.
    func testAChoreAndAnEventSharingARawIDStillGetDistinctRowIDs() {
        let stats = makeStats(dueToday: [makeChore(id: "same", dueTime: "10:00")])

        let data = TodayWidgetData.from(stats: stats, events: [makeEvent(id: "same", at: 11)])

        XCTAssertEqual(Set(data.items.map(\.id)).count, 2)
    }

    func testAnEventKeepsItsCalendarColourAndPrefersLocationForItsSubtitle() {
        let event = makeEvent(id: "e1", location: "Leisure centre", description: "Bring goggles", color: "#FF8800")

        let data = TodayWidgetData.from(stats: makeStats(), events: [event])

        XCTAssertEqual(data.items.first?.colorHex, "#FF8800")
        XCTAssertEqual(data.items.first?.subtitle, "Leisure centre")
    }

    func testAnEventWithNoLocationFallsBackToItsDescription() {
        let event = makeEvent(id: "e1", location: "   ", description: "Bring goggles")

        let data = TodayWidgetData.from(stats: makeStats(), events: [event])

        XCTAssertEqual(data.items.first?.subtitle, "Bring goggles")
    }

    func testAnAllDayEventSaysSoInsteadOfShowingAMisleadingMidnight() {
        let data = TodayWidgetData.from(stats: makeStats(), events: [makeEvent(id: "e1", allDay: true)])

        XCTAssertEqual(data.items.first?.timeLabel, "All day")
    }

    // MARK: - Chore labelling

    func testAssigneeNameAndInitialsAreResolvedAtBuildTime() {
        let stats = makeStats(dueToday: [makeChore(id: "c1", assignedTo: "u1")])

        let data = TodayWidgetData.from(stats: stats, users: ["u1": ben])

        XCTAssertEqual(data.items.first?.subtitle, "Ben Suskins")
        XCTAssertEqual(data.items.first?.initials, "BS")
    }

    func testUnknownOrMissingAssigneeLeavesTheFieldsEmpty() {
        let stats = makeStats(dueToday: [
            makeChore(id: "c1", assignedTo: "ghost"),
            makeChore(id: "c2", assignedTo: nil),
        ])

        let data = TodayWidgetData.from(stats: stats, users: ["u1": ben])

        XCTAssertEqual(data.items.compactMap(\.subtitle), [])
        XCTAssertEqual(data.items.compactMap(\.initials), [])
    }

    // How long ago it was due matters more than what time of day it was, so an
    // overdue chore trades its clock time for the date it slipped.
    func testAnOverdueChoreShowsOverdueAndKeepsItsDateInTheSubtitle() {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", dueDate: "2026-09-09", dueTime: "18:30", assignedTo: "u1")]
        )

        let data = TodayWidgetData.from(stats: stats, users: ["u1": ben])

        XCTAssertEqual(data.items.first?.timeLabel, "Overdue")
        // The date's exact wording is locale-dependent, so only the shape of
        // the subtitle is asserted here.
        XCTAssertEqual(data.items.first?.subtitle?.hasPrefix("Ben Suskins · "), true)
    }

    func testAChoreWithNoDueTimeHasNoTimeLabel() {
        let data = TodayWidgetData.from(stats: makeStats(dueToday: [makeChore(id: "c1")]))

        XCTAssertNil(data.items.first?.timeLabel)
    }

    // MARK: - Meals

    func testMealsAreKeptOutOfTheTimelineAndOrderedThroughTheDay() {
        let stats = makeStats(meals: [
            makeMeal("dinner", "Bolognese"),
            makeMeal("breakfast", "Porridge"),
        ])

        let data = TodayWidgetData.from(stats: stats)

        XCTAssertTrue(data.items.isEmpty)
        XCTAssertEqual(data.meals.map(\.slot), [.breakfast, .dinner])
        XCTAssertEqual(data.dinner?.name, "Bolognese")
    }

    func testBlankAndUnrecognisedMealSlotsAreDroppedRatherThanDrawnEmpty() {
        let stats = makeStats(meals: [
            makeMeal("dinner", "   "),
            makeMeal("brunch", "Eggs"),
            makeMeal("lunch", "Soup"),
        ])

        let data = TodayWidgetData.from(stats: stats)

        XCTAssertEqual(data.meals.map(\.name), ["Soup"])
        XCTAssertNil(data.dinner)
    }

    func testMealSlotIsReadCaseInsensitively() {
        let data = TodayWidgetData.from(stats: makeStats(meals: [makeMeal("Dinner", "Bolognese")]))

        XCTAssertEqual(data.dinner?.name, "Bolognese")
    }

    // MARK: - Counts

    func testCountsComeFromTheFullListsNotTheCappedRows() {
        let overdue = (1...6).map { makeChore(id: "o\($0)", dueDate: "2026-09-0\($0)") }
        let dueToday = (1...5).map { makeChore(id: "t\($0)", dueTime: "0\($0):00") }
        let events = (1...4).map { makeEvent(id: "e\($0)", at: 10 + $0) }

        let data = TodayWidgetData.from(stats: makeStats(overdue: overdue, dueToday: dueToday), events: events)

        XCTAssertEqual(data.items.count, TodayWidgetData.maxItems)
        XCTAssertEqual(data.totalItemCount, 15)
        XCTAssertEqual(data.overdueCount, 6)
        XCTAssertEqual(data.dueTodayCount, 5)
        XCTAssertEqual(data.choreCount, 11)
        XCTAssertEqual(data.eventCount, 4)
        XCTAssertEqual(data.hiddenCount(shown: TodayWidgetData.maxItems), 15 - TodayWidgetData.maxItems)
        XCTAssertFalse(data.isClear)
    }

    func testAViewShowingFewerRowsThanTheSnapshotHoldsGetsTheRestAsHidden() {
        let dueToday = (1...5).map { makeChore(id: "t\($0)", dueTime: "0\($0):00") }

        let data = TodayWidgetData.from(stats: makeStats(dueToday: dueToday))

        XCTAssertEqual(data.hiddenCount(shown: 3), 2)
        XCTAssertEqual(data.hiddenCount(shown: 99), 0)
    }

    func testAnEmptyDayIsClear() {
        let data = TodayWidgetData.from(stats: makeStats())

        XCTAssertTrue(data.isClear)
        XCTAssertTrue(data.items.isEmpty)
        XCTAssertTrue(data.meals.isEmpty)
        XCTAssertEqual(data.hiddenCount(shown: 0), 0)
    }

    // A day with nothing to do but something to cook still has something to
    // say, so it must not render as the all-clear.
    func testADayWithOnlyAMealPlannedIsNotClear() {
        let data = TodayWidgetData.from(stats: makeStats(meals: [makeMeal("dinner", "Bolognese")]))

        XCTAssertFalse(data.isClear)
    }

    // MARK: - Optimistic removal

    func testRemovingADueTodayChoreDropsItsRowAndItsCounts() {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", dueDate: "2026-09-09")],
            dueToday: [makeChore(id: "t1", dueTime: "09:00")]
        )
        let data = TodayWidgetData.from(stats: stats, events: [makeEvent(id: "e1", at: 12)])

        let after = data.removing(choreID: "t1")

        XCTAssertEqual(after.items.map(\.id), ["chore-o1", "event-e1"])
        XCTAssertEqual(after.totalItemCount, 2)
        XCTAssertEqual(after.overdueCount, 1)
        XCTAssertEqual(after.dueTodayCount, 0)
        XCTAssertEqual(after.eventCount, 1)
    }

    func testRemovingAnOverdueChoreDecrementsTheOverdueCount() {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", dueDate: "2026-09-09")],
            dueToday: [makeChore(id: "t1", dueTime: "09:00")]
        )
        let data = TodayWidgetData.from(stats: stats)

        let after = data.removing(choreID: "o1")

        XCTAssertEqual(after.items.map(\.id), ["chore-t1"])
        XCTAssertEqual(after.overdueCount, 0)
        XCTAssertEqual(after.dueTodayCount, 1)
    }

    // A double-tap on the widget, or a tick for a chore that only ever existed
    // in the uncapped tail, must not drive a count below zero.
    func testRemovingAnUnknownChoreIsANoOp() {
        let data = TodayWidgetData.from(stats: makeStats(dueToday: [makeChore(id: "t1")]))

        XCTAssertEqual(data.removing(choreID: "nope"), data)
        XCTAssertEqual(data.removing(choreID: "t1").removing(choreID: "t1").dueTodayCount, 0)
    }

    // Events aren't the widget's to complete, so an event id must never be
    // mistaken for a tickable chore.
    func testRemovingCannotTouchAnEventRow() {
        let data = TodayWidgetData.from(stats: makeStats(), events: [makeEvent(id: "e1")])

        XCTAssertEqual(data.removing(choreID: "e1"), data)
        XCTAssertEqual(data.removing(choreID: "event-e1"), data)
    }

    // MARK: - Encoding

    func testSnapshotSurvivesAJSONRoundTrip() throws {
        let stats = makeStats(
            overdue: [makeChore(id: "o1", name: "Bins", dueDate: "2026-09-09", assignedTo: "u1")],
            dueToday: [makeChore(id: "t1", name: "Hoovering", dueTime: "18:00")],
            meals: [makeMeal("dinner", "Bolognese")]
        )
        let original = TodayWidgetData.from(
            stats: stats,
            events: [makeEvent(id: "e1", title: "Swimming", location: "Pool")],
            users: ["u1": ben]
        )

        let encoded = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(TodayWidgetData.self, from: encoded)

        XCTAssertEqual(decoded, original)
    }

    func testMarkedStaleKeepsEverythingButTheTimestamp() {
        let original = TodayWidgetData.from(
            stats: makeStats(dueToday: [makeChore(id: "t1")], meals: [makeMeal("dinner", "Bolognese")]),
            events: [makeEvent(id: "e1")]
        )

        let stale = original.markedStale()

        XCTAssertEqual(stale.items, original.items)
        XCTAssertEqual(stale.totalItemCount, original.totalItemCount)
        XCTAssertEqual(stale.overdueCount, original.overdueCount)
        XCTAssertEqual(stale.dueTodayCount, original.dueTodayCount)
        XCTAssertEqual(stale.eventCount, original.eventCount)
        XCTAssertEqual(stale.meals, original.meals)
        XCTAssertEqual(stale.capturedAt, .distantPast)
    }
}
