// ios/FamilyHub/Features/Meals/MealsViewModel.swift
import Foundation
import Observation

@Observable
@MainActor
final class MealsViewModel {
    var state: ViewState<[MealPlan]> = .idle
    var currentWeek: Date = {
        let calendar = Calendar(identifier: .iso8601)
        return calendar.dateInterval(of: .weekOfYear, for: Date())!.start
    }()

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
        let calendar = Calendar(identifier: .iso8601)
        currentWeek = calendar.dateInterval(of: .weekOfYear, for: Date())!.start
        Task { await load() }
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
