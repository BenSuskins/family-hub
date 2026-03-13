// ios/FamilyHub/Features/Meals/MealsView.swift
import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel

    private let days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
    private let mealTypes = ["breakfast", "lunch", "dinner"]

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                switch viewModel.state {
                case .idle, .loading:
                    ProgressView()
                case .loaded(let meals):
                    mealsTable(meals)
                case .failed(let error):
                    ContentUnavailableView(
                        "Failed to load",
                        systemImage: "exclamationmark.triangle",
                        description: Text(error.localizedDescription)
                    )
                }
            }
            .navigationTitle(weekTitle)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("< Prev") { viewModel.previousWeek() }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Next >") { viewModel.nextWeek() }
                }
            }
        }
        .task { await viewModel.load() }
    }

    private static let dayLabelFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "d"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    private static let dateStringFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    private static let weekTitleFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "d MMM"
        return f
    }()

    private var weekTitle: String {
        let start = viewModel.currentWeek
        let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
        return "\(Self.weekTitleFormatter.string(from: start)) – \(Self.weekTitleFormatter.string(from: end))"
    }

    private func mealsTable(_ meals: [MealPlan]) -> some View {
        List {
            ForEach(Array(days.enumerated()), id: \.offset) { index, day in
                let date = dayDate(offset: index)
                Section(day + " " + dateLabel(date)) {
                    ForEach(mealTypes, id: \.self) { mealType in
                        let meal = meals.first(where: { $0.date == dateString(date) && $0.mealType == mealType })
                        HStack {
                            Text(mealType.capitalized)
                                .foregroundStyle(.secondary)
                                .frame(width: 90, alignment: .leading)
                            Text(meal?.name ?? "—")
                        }
                    }
                }
            }
        }
    }

    private func dayDate(offset: Int) -> Date {
        Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
    }

    private func dateLabel(_ date: Date) -> String {
        Self.dayLabelFormatter.string(from: date)
    }

    private func dateString(_ date: Date) -> String {
        Self.dateStringFormatter.string(from: date)
    }
}
