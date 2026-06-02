import Foundation
import Observation

enum CalendarViewMode: String, CaseIterable {
    case month = "Month"
    case week = "Week"
    case day = "Day"
}

@Observable
@MainActor
final class CalendarViewModel {
    var state: ViewState<CalendarResponse> = .idle
    var viewMode: CalendarViewMode = .month
    var currentDate: Date = Date()
    var selectedDay: Date? = Date()
    var users: [String: User] = [:]

    private let apiClient: any APIClientProtocol
    private var responseCache: [String: CalendarResponse] = [:]
    private var cachedUsers: [String: User]?

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    var currentMonth: Date {
        let components = Calendar.current.dateComponents([.year, .month], from: currentDate)
        return Calendar.current.date(from: components)!
    }

    var currentWeekStart: Date {
        let calendar = Calendar(identifier: .iso8601)
        return calendar.dateInterval(of: .weekOfYear, for: currentDate)!.start
    }

    func load(forceRefresh: Bool = false) async {
        if forceRefresh { cachedUsers = nil }

        let key = cacheKey()

        if let cached = responseCache[key] {
            if let cachedUsers { users = cachedUsers }
            state = .loaded(cached)
        } else {
            state = .loading
        }

        do {
            async let calendarTask = apiClient.fetchCalendar(
                view: viewMode.rawValue.lowercased(), date: currentDate)

            if cachedUsers == nil {
                let userList = try await apiClient.fetchUsers()
                users = userList.keyedByID
                cachedUsers = users
            }

            let response = try await calendarTask
            responseCache[key] = response
            state = .loaded(response)
        } catch {
            if responseCache[key] == nil {
                state = .failed(.from(error))
            }
        }
    }

    func nextMonth() {
        currentDate = Calendar.current.date(byAdding: .month, value: 1, to: currentDate)!
        Task { await load() }
    }

    func previousMonth() {
        currentDate = Calendar.current.date(byAdding: .month, value: -1, to: currentDate)!
        Task { await load() }
    }

    func nextWeek() {
        currentDate = Calendar.current.date(byAdding: .day, value: 7, to: currentDate)!
        Task { await load() }
    }

    func previousWeek() {
        currentDate = Calendar.current.date(byAdding: .day, value: -7, to: currentDate)!
        Task { await load() }
    }

    func nextDay() {
        currentDate = Calendar.current.date(byAdding: .day, value: 1, to: currentDate)!
        selectedDay = currentDate
        Task { await load() }
    }

    func previousDay() {
        currentDate = Calendar.current.date(byAdding: .day, value: -1, to: currentDate)!
        selectedDay = currentDate
        Task { await load() }
    }

    func goToToday() {
        currentDate = Date()
        selectedDay = Date()
        Task { await load() }
    }

    func chores(for day: Date) -> [Chore] {
        guard case .loaded(let response) = state else { return [] }
        let dayString = APIDate.dayString(day)
        return response.chores.filter { chore in
            guard let dueDate = chore.dueDate else { return false }
            return dueDate.hasPrefix(dayString)
        }
    }

    func events(for day: Date) -> [CalendarEvent] {
        guard case .loaded(let response) = state else { return [] }
        return response.events.filter { event in
            Calendar.current.isDate(event.startTime, inSameDayAs: day)
        }
    }

    func meals(for day: Date) -> [MealPlan] {
        guard case .loaded(let response) = state else { return [] }
        let dayString = APIDate.dayString(day)
        return response.meals.filter { $0.date == dayString }
    }

    func hasItems(for day: Date) -> Bool {
        !chores(for: day).isEmpty || !events(for: day).isEmpty || !meals(for: day).isEmpty
    }

    private func cacheKey() -> String {
        switch viewMode {
        case .month:
            return "month-\(APIDate.monthString(currentDate))"
        case .week:
            return "week-\(APIDate.dayString(currentWeekStart))"
        case .day:
            return "day-\(APIDate.dayString(currentDate))"
        }
    }
}
