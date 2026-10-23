// ios/FamilyHub/Features/Meals/MealsViewModel.swift
import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class MealsViewModel {
    var actionError: APIError?
    var state: ViewState<[MealPlan]> = .idle
    private(set) var refreshError: APIError?
    private(set) var snapshotMetadata: SnapshotMetadata?
    var currentWeek: Date = MealsViewModel.fridayStartOfWeek(for: Date())
    private(set) var selectedDay: Date = Date()

    var weekDays: [Date] {
        (0..<7).compactMap { Calendar.current.date(byAdding: .day, value: $0, to: currentWeek) }
    }

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load(forceRefresh: Bool = false) async {
        let week = currentWeek
        if case .loaded = state {} else { state = .loading }
        refreshError = nil
        do {
            let meals = try await apiClient.fetchMeals(week: week, forceRefresh: forceRefresh)
            guard !Task.isCancelled, week == currentWeek else { return }
            snapshotMetadata = await apiClient.snapshotMetadata(for: "meals-\(APIDate.dayString(week))")
            state = .loaded(meals)
        }
        catch {
            guard !Task.isCancelled, week == currentWeek else { return }
            let apiError = APIError.from(error)
            refreshError = apiError
            if case .loading = state { state = .failed(apiError) }
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
        selectedDay = currentWeek
        Task { await load() }
    }

    func previousWeek() {
        currentWeek = Calendar.current.date(byAdding: .day, value: -7, to: currentWeek)!
        selectedDay = weekDays.last ?? currentWeek
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
