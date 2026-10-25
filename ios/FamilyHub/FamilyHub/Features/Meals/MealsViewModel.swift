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
    private(set) var mealWeekStart: MealWeekday = .friday
    var currentWeek: Date = MealWeekday.friday.weekStart(containing: Date())
    private(set) var selectedDay: Date = Date()

    var weekDays: [Date] {
        (0..<7).compactMap { Calendar.current.date(byAdding: .day, value: $0, to: currentWeek) }
    }

    private var loadIdentifier = UUID()
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load(forceRefresh: Bool = false) async {
        let identifier = UUID()
        loadIdentifier = identifier
        do {
            let settings = try await apiClient.fetchSettings()
            guard !Task.isCancelled, loadIdentifier == identifier else { return }
            if settings.mealWeekStart != mealWeekStart {
                mealWeekStart = settings.mealWeekStart
                currentWeek = mealWeekStart.weekStart(containing: selectedDay)
                state = .idle
                snapshotMetadata = nil
            }
        } catch {
            guard !Task.isCancelled, loadIdentifier == identifier else { return }
        }
        let week = currentWeek
        if case .loaded = state {} else { state = .loading }
        refreshError = nil
        do {
            let meals = try await apiClient.fetchMeals(week: week, forceRefresh: forceRefresh)
            guard !Task.isCancelled, loadIdentifier == identifier, week == currentWeek else { return }
            let metadata = await apiClient.snapshotMetadata(for: "meals-\(APIDate.dayString(week))")
            guard !Task.isCancelled, loadIdentifier == identifier, week == currentWeek else { return }
            snapshotMetadata = metadata
            state = .loaded(meals)
        }
        catch {
            guard !Task.isCancelled, loadIdentifier == identifier, week == currentWeek else { return }
            let apiError = APIError.from(error)
            refreshError = apiError
            if case .loading = state { state = .failed(apiError) }
        }
    }

    func selectDay(_ date: Date) async {
        selectedDay = date
        let week = mealWeekStart.weekStart(containing: date)
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
        currentWeek = mealWeekStart.weekStart(containing: Date())
        selectedDay = Date()
        Task { await load() }
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
