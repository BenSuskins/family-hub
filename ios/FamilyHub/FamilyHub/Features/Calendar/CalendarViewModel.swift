// ios/FamilyHub/Features/Calendar/CalendarViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class CalendarViewModel {
    var state: ViewState<[Chore]> = .idle
    var currentMonth: Date = {
        let components = Calendar.current.dateComponents([.year, .month], from: Date())
        return Calendar.current.date(from: components)!
    }()
    var selectedDay: Date?

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        do {
            let chores = try await apiClient.fetchCalendar(month: currentMonth)
            state = .loaded(chores)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }

    func nextMonth() {
        currentMonth = Calendar.current.date(byAdding: .month, value: 1, to: currentMonth)!
        Task { await load() }
    }

    func previousMonth() {
        currentMonth = Calendar.current.date(byAdding: .month, value: -1, to: currentMonth)!
        Task { await load() }
    }

    func chores(for day: Date) -> [Chore] {
        guard case .loaded(let chores) = state else { return [] }
        let dayString = iso8601DayString(day)
        return chores.filter { chore in
            guard let dueDate = chore.dueDate else { return false }
            return dueDate.hasPrefix(dayString)
        }
    }

    private func iso8601DayString(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
    }
}
