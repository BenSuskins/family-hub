import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel

    private let mealTypes = ["breakfast", "lunch", "dinner"]

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "EEE d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dateKeyFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "yyyy-MM-dd"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let weekTitleFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d MMM"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                Group {
                    if case .failed(let error) = viewModel.state {
                        VStack {
                            Text(error.localizedDescription)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.statusRed)
                                .padding()
                            Spacer()
                        }
                    } else if case .loaded(let meals) = viewModel.state {
                        mealsContent(meals)
                    } else {
                        ProgressView().tint(Theme.textSecondary)
                    }
                }
            }
            .navigationTitle(weekTitle)
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        viewModel.previousWeek()
                    } label: {
                        Image(systemName: "chevron.left")
                            .foregroundStyle(Theme.accent)
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        viewModel.nextWeek()
                    } label: {
                        Image(systemName: "chevron.right")
                            .foregroundStyle(Theme.accent)
                    }
                }
            }
        }
        .task { await viewModel.load() }
    }

    private var weekTitle: String {
        let start = viewModel.currentWeek
        let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
        return "\(Self.weekTitleFormatter.string(from: start)) – \(Self.weekTitleFormatter.string(from: end))"
    }

    private func mealsContent(_ meals: [MealPlan]) -> some View {
        List {
            ForEach(0..<7, id: \.self) { offset in
                let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                let dateKey = Self.dateKeyFormatter.string(from: date)
                Section {
                    ForEach(mealTypes, id: \.self) { mealType in
                        let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                        HStack(spacing: 12) {
                            Text(mealType.capitalized)
                                .font(.system(size: 12))
                                .foregroundStyle(Theme.textMuted)
                                .frame(width: 70, alignment: .leading)
                            Text(plan?.name ?? "—")
                                .font(.system(size: 14))
                                .foregroundStyle(plan != nil ? Theme.textPrimary : Theme.textMuted)
                        }
                        .padding(.vertical, 4)
                        .listRowBackground(Theme.surface)
                        .listRowSeparatorTint(Theme.borderDivider)
                    }
                } header: {
                    Text(Self.dayFormatter.string(from: date))
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(Theme.textSecondary)
                        .textCase(nil)
                }
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
        .refreshable { await viewModel.load() }
    }
}
