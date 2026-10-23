import Foundation
import Observation
import FamilyHubKit

enum CalendarViewMode: String, CaseIterable {
    case year = "Year"
    case month = "Month"
    case day = "Day"
    case week = "Week"
}

@Observable
@MainActor
final class CalendarViewModel {
    private(set) var state: ViewState<CalendarResponse> = .idle
    private(set) var isLoading = false
    private(set) var loadError: APIError?
    private(set) var selectedDate: Date
    private(set) var viewMode: CalendarViewMode
    private(set) var monthScale: Double
    private(set) var hourHeight: Double
    private(set) var navigationRevision = 0
    private(set) var users: [String: User] = [:]
    private(set) var snapshotMetadata: [String: SnapshotMetadata] = [:]
    private(set) var schedule: CalendarSchedule

    private let apiClient: any APIClientProtocol
    private let now: () -> Date
    private let preferences: UserDefaults?
    private var combinedResponse = CalendarResponse(chores: [], events: [], meals: [])
    private var responses: [String: CalendarResponse] = [:]
    private var responseVersions: [String: Int] = [:]
    private var responseVersion = 0
    private var requests: [String: Task<CalendarResponse, Error>] = [:]
    private var loadIdentifier = UUID()
    private var hasUsers = false

    init(apiClient: any APIClientProtocol, calendar: Calendar = .current,
         now: @escaping () -> Date = Date.init, preferences: UserDefaults? = .standard) {
        self.apiClient = apiClient
        self.schedule = CalendarSchedule(calendar: calendar)
        self.now = now
        self.preferences = preferences
        selectedDate = calendar.startOfDay(for: now())
        viewMode = preferences?.string(forKey: "calendar.view").flatMap(CalendarViewMode.init(rawValue:)) ?? .month
        monthScale = min(2, max(0.6, preferences?.object(forKey: "calendar.monthScale") as? Double ?? 1))
        hourHeight = min(120, max(44, preferences?.object(forKey: "calendar.hourHeight") as? Double ?? 64))
    }

    var currentMonth: Date { schedule.monthStart(containing: selectedDate) }
    var currentMonthSnapshotMetadata: SnapshotMetadata? { snapshotMetadata[schedule.monthKey(currentMonth)] }
    var currentWeekStart: Date { schedule.weekStart(containing: selectedDate) }
    var today: Date { now() }
    var requestKey: String { "\(viewMode.rawValue)-\(schedule.dayKey(selectedDate))-\(schedule.calendar.timeZone.identifier)-\(schedule.calendar.firstWeekday)" }

    var timelineDays: [Date] {
        let start = viewMode == .week ? currentWeekStart : selectedDate
        let count = viewMode == .week ? 7 : 1
        return (0..<count).map { schedule.adding(.day, value: $0, to: start) }
    }

    func updateCalendar(_ calendar: Calendar) {
        guard calendar != schedule.calendar else { return }
        let date = schedule.calendar.dateComponents([.year, .month, .day], from: selectedDate)
        cancelLoads()
        schedule = CalendarSchedule(calendar: calendar)
        selectedDate = calendar.date(from: date)!
        responses = [:]
        snapshotMetadata = [:]
        responseVersions = [:]
        combinedResponse = CalendarResponse(chores: [], events: [], meals: [])
        state = .idle
    }

    func setView(_ mode: CalendarViewMode) {
        viewMode = mode
        preferences?.set(mode.rawValue, forKey: "calendar.view")
    }

    func setMonthScale(_ scale: Double) {
        monthScale = min(2, max(0.6, scale))
        preferences?.set(monthScale, forKey: "calendar.monthScale")
    }

    func setHourHeight(_ height: Double) {
        hourHeight = min(120, max(44, height))
        preferences?.set(hourHeight, forKey: "calendar.hourHeight")
    }

    func selectDate(_ date: Date) {
        selectedDate = schedule.calendar.startOfDay(for: date)
    }

    func openDay(_ date: Date) {
        selectDate(date)
        setView(.day)
    }

    func openMonth(_ date: Date) {
        selectDate(date)
        setView(.month)
    }

    func goBack() {
        setView(viewMode == .month ? .year : .month)
    }

    func move(_ direction: Int) {
        let component: Calendar.Component = viewMode == .year ? .year : viewMode == .month ? .month : .day
        let step = viewMode == .week ? 7 : 1
        selectDate(schedule.adding(component, value: direction * step, to: selectedDate))
    }

    func goToToday() {
        selectDate(now())
        navigationRevision += 1
    }

    func hasLoaded(_ month: Date) -> Bool { responses[schedule.monthKey(month)] != nil }

    func load(forceRefresh: Bool = false) async {
        let identifier = UUID()
        loadIdentifier = identifier
        loadError = nil
        guard viewMode != .year else {
            requests.values.forEach { $0.cancel() }
            requests = [:]
            isLoading = false
            return
        }
        let visibleMonths = timelineDays.map { schedule.monthStart(containing: $0) }
        let required = Array(Set(visibleMonths + [currentMonth])).sorted()
        let adjacent = [
            schedule.adding(.month, value: -1, to: required.first!),
            schedule.adding(.month, value: 1, to: required.last!),
        ]
        let months = required + adjacent.filter { !required.contains($0) }
        let keys = Set(months.map { schedule.monthKey($0) })
        for key in Array(requests.keys) where !keys.contains(key) || forceRefresh {
            requests.removeValue(forKey: key)?.cancel()
        }
        isLoading = required.contains { forceRefresh || !hasLoaded($0) }
        if responses.isEmpty { state = .loading }
        defer { if loadIdentifier == identifier { isLoading = false } }

        for month in months {
            guard !Task.isCancelled, loadIdentifier == identifier else { return }
            if !required.contains(month) { isLoading = false }
            let key = schedule.monthKey(month)
            if !forceRefresh, responses[key] != nil {
                let metadata: SnapshotMetadata?
                if let storedMetadata = snapshotMetadata[key] {
                    metadata = storedMetadata
                } else {
                    metadata = await apiClient.snapshotMetadata(for: "calendar-\(key)")
                }
                snapshotMetadata[key] = metadata
                if metadata == nil || metadata?.isFresh == true { continue }
            }
            let request = requests[key] ?? Task { try await apiClient.fetchCalendar(view: "month", date: month, forceRefresh: forceRefresh) }
            requests[key] = request
            do {
                let response = try await request.value
                guard !Task.isCancelled, loadIdentifier == identifier else { return }
                requests[key] = nil
                responses[key] = response
                snapshotMetadata[key] = await apiClient.snapshotMetadata(for: "calendar-\(key)")
                responseVersion += 1
                responseVersions[key] = responseVersion
                publish()
            } catch {
                guard !Task.isCancelled, loadIdentifier == identifier else { return }
                requests[key] = nil
                if required.contains(month) { loadError = .from(error) }
            }
        }
        guard loadIdentifier == identifier, !Task.isCancelled else { return }
        if !hasUsers || forceRefresh {
            do {
                let userList = try await apiClient.fetchUsers()
                guard loadIdentifier == identifier, !Task.isCancelled else { return }
                users = userList.keyedByID
                hasUsers = true
            } catch { /* Names do not prevent access to the calendar. */ }
        }
        if responses.isEmpty, let loadError { state = .failed(loadError) }
    }

    func cancelLoads() {
        loadIdentifier = UUID()
        requests.values.forEach { $0.cancel() }
        requests = [:]
        isLoading = false
    }

    func events(for day: Date) -> [CalendarEvent] {
        merged.events.filter { schedule.contains($0, on: day) }.sorted {
            if $0.allDay != $1.allDay { return $0.allDay }
            return $0.startTime == $1.startTime ? $0.id < $1.id : $0.startTime < $1.startTime
        }
    }

    func chores(for day: Date) -> [Chore] {
        let key = schedule.dayKey(day)
        return merged.chores.filter { $0.dueDate?.hasPrefix(key) == true }.sorted { $0.name < $1.name }
    }

    func meals(for day: Date) -> [MealPlan] {
        let order = ["breakfast", "lunch", "dinner"]
        return merged.meals.filter { $0.date == schedule.dayKey(day) }.sorted {
            (order.firstIndex(of: $0.mealType) ?? 3) < (order.firstIndex(of: $1.mealType) ?? 3)
        }
    }

    func hasItems(for day: Date) -> Bool {
        !events(for: day).isEmpty || !chores(for: day).isEmpty || !meals(for: day).isEmpty
    }

    private var merged: CalendarResponse { combinedResponse }

    private func combined() -> CalendarResponse {
        let values = responses.keys.sorted { (responseVersions[$0] ?? 0) > (responseVersions[$1] ?? 0) }
            .compactMap { responses[$0] }
        return CalendarResponse(
            chores: unique(values.flatMap(\.chores), id: { $0.id }),
            events: unique(values.flatMap(\.events), id: { $0.id }),
            meals: unique(values.flatMap(\.meals), id: { $0.id })
        )
    }

    private func unique<Item>(_ items: [Item], id: (Item) -> String) -> [Item] {
        var identifiers: Set<String> = []
        return items.filter { identifiers.insert(id($0)).inserted }
    }

    private func publish() {
        combinedResponse = combined()
        state = .loaded(combinedResponse)
    }
}
