// ios/FamilyHub/Features/Meals/MealsViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class MealsViewModel {
    var state: ViewState<[MealPlan]> = .idle
    var currentWeek: Date = MealsViewModel.fridayStartOfWeek(for: Date())

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        do {
            let meals = try await apiClient.fetchMeals(week: currentWeek)
            state = .loaded(meals)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }

    func nextWeek() {
        currentWeek = Calendar.current.date(byAdding: .day, value: 7, to: currentWeek)!
        Task { await load() }
    }

    func previousWeek() {
        currentWeek = Calendar.current.date(byAdding: .day, value: -7, to: currentWeek)!
        Task { await load() }
    }

    func goToCurrentWeek() {
        currentWeek = MealsViewModel.fridayStartOfWeek(for: Date())
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
        do {
            _ = try await apiClient.saveMeal(date: date, mealType: mealType, name: name, recipeID: recipeID)
            await load()
            return true
        } catch {
            return false
        }
    }

    func deleteMeal(date: String, mealType: String) async -> Bool {
        do {
            try await apiClient.deleteMeal(date: date, mealType: mealType)
            await load()
            return true
        } catch {
            return false
        }
    }

}
