// ios/FamilyHub/Features/Meals/MealsViewModel.swift
import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class MealsViewModel {
    var actionError: APIError?
    var state: ViewState<[MealPlan]> = .idle
    var currentWeek: Date = MealsViewModel.fridayStartOfWeek(for: Date())
    private(set) var selectedDay: Date = Date()

    var weekDays: [Date] {
        (0..<7).compactMap { Calendar.current.date(byAdding: .day, value: $0, to: currentWeek) }
    }

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        let week = currentWeek
        state = .loading
        do {
            let meals = try await apiClient.fetchMeals(week: week)
            guard week == currentWeek else { return }
            state = .loaded(meals)
        } catch {
            guard week == currentWeek else { return }
            state = .failed(.from(error))
        }
    }

    func selectDay(_ date: Date) async {
        selectedDay = date
        let week = Self.fridayStartOfWeek(for: date)
        guard !Calendar.current.isDate(week, inSameDayAs: currentWeek) else { return }
        currentWeek = week
        await load()
    }

    func nextWeek() {
        currentWeek = Calendar.current.date(byAdding: .day, value: 7, to: currentWeek)!
        selectedDay = Calendar.current.date(byAdding: .day, value: 7, to: selectedDay)!
        Task { await load() }
    }

    func previousWeek() {
        currentWeek = Calendar.current.date(byAdding: .day, value: -7, to: currentWeek)!
        selectedDay = Calendar.current.date(byAdding: .day, value: -7, to: selectedDay)!
        Task { await load() }
    }

    func goToCurrentWeek() {
        currentWeek = MealsViewModel.fridayStartOfWeek(for: Date())
        selectedDay = Date()
        Task { await load() }
    }

    private static func fridayStartOfWeek(for date: Date) -> Date {
        let calendar = Calendar(identifier: .gregorian)
        let weekday = calendar.component(.weekday, from: date)
        // weekday: 1=Sun, 2=Mon, 3=Tue, 4=Wed, 5=Thu, 6=Fri, 7=Sat
        let daysSinceFriday = (weekday - 6 + 7) % 7
        let startOfDay = calendar.startOfDay(for: date)
        return calendar.date(byAdding: .day, value: -daysSinceFriday, to: startOfDay)!
    }

    func saveMeal(date: String, mealType: String, name: String, recipeID: String? = nil) async -> Bool {
        actionError = nil
        do {
            _ = try await apiClient.saveMeal(date: date, mealType: mealType, name: name, recipeID: recipeID)
            await load()
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }

    func deleteMeal(date: String, mealType: String) async -> Bool {
        actionError = nil
        do {
            try await apiClient.deleteMeal(date: date, mealType: mealType)
            await load()
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }

}
