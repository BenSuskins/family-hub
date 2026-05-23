import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var editingMeal: EditingMeal?

    private static let dateKeyFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "yyyy-MM-dd"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let weekOfFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "MMM d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dayNameFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "EEEE"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dayNumberFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
    }

    private var isCurrentWeek: Bool {
        Calendar.current.isDate(viewModel.currentWeek, equalTo: Date(), toGranularity: .weekOfYear)
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                weekNavStrip
                scrollContent
            }
            .meshBackground()
            .navigationTitle("Meal plan")
            .navigationBarTitleDisplayMode(.large)
            .sheet(item: $editingMeal) { meal in
                MealEditSheet(meal: meal, viewModel: viewModel, apiClient: apiClient)
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Week navigation strip

    private var weekNavStrip: some View {
        HStack(spacing: 4) {
            Button {
                viewModel.previousWeek()
                Task { await viewModel.load() }
            } label: {
                Image(systemName: "chevron.left")
                    .font(.system(size: 14, weight: .semibold))
                    .frame(width: 32, height: 32)
                    .background(Color(UIColor.secondarySystemGroupedBackground), in: Circle())
            }
            .buttonStyle(.plain)

            VStack(spacing: 1) {
                Text("Week of \(Self.weekOfFormatter.string(from: viewModel.currentWeek))")
                    .font(.system(size: 15, weight: .semibold))
                if isCurrentWeek {
                    Text("This week")
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                } else {
                    Button("Jump to this week") {
                        viewModel.goToCurrentWeek()
                        Task { await viewModel.load() }
                    }
                    .font(.system(size: 12))
                }
            }
            .frame(maxWidth: .infinity)

            Button {
                viewModel.nextWeek()
                Task { await viewModel.load() }
            } label: {
                Image(systemName: "chevron.right")
                    .font(.system(size: 14, weight: .semibold))
                    .frame(width: 32, height: 32)
                    .background(Color(UIColor.secondarySystemGroupedBackground), in: Circle())
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 16)
        .padding(.bottom, 12)
    }

    // MARK: - Scroll content

    @ViewBuilder
    private var scrollContent: some View {
        switch viewModel.state {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let error):
            ContentUnavailableView(
                "Error",
                systemImage: "exclamationmark.triangle",
                description: Text(error.localizedDescription)
            )
        case .loaded(let meals):
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    ForEach(0..<7, id: \.self) { offset in
                        let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                        let dateKey = Self.dateKeyFormatter.string(from: date)
                        let isToday = Calendar.current.isDateInToday(date)

                        daySection(
                            date: date,
                            dateKey: dateKey,
                            isToday: isToday,
                            meals: meals
                        )
                    }
                    Spacer(minLength: 24)
                }
                .padding(.bottom, 8)
            }
            .refreshable { await viewModel.load() }
        }
    }

    // MARK: - Day section

    private func daySection(date: Date, dateKey: String, isToday: Bool, meals: [MealPlan]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(Self.dayNameFormatter.string(from: date))
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(isToday ? Color.accentColor : Color.primary)
                Text(Self.dayNumberFormatter.string(from: date))
                    .font(.system(size: 16))
                    .foregroundStyle(.tertiary)
                    .monospacedDigit()
                if isToday {
                    Text("Today")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(.secondary)
                        .kerning(0.5)
                        .textCase(.uppercase)
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 14)
            .padding(.bottom, 8)

            VStack(spacing: 0) {
                ForEach(Array(["breakfast", "lunch", "dinner"].enumerated()), id: \.element) { index, mealType in
                    let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                    if index > 0 {
                        Divider().padding(.leading, 74)
                    }
                    Button {
                        editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan?.name ?? "")
                    } label: {
                        MealDayRow(
                            slot: mealType.capitalized,
                            mealPlan: plan,
                            apiClient: apiClient
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
            .glassCard(radius: 12)
            .padding(.horizontal, 16)
        }
    }
}

// MARK: - MealDayRow

private struct MealDayRow: View {
    let slot: String
    let mealPlan: MealPlan?
    let apiClient: any APIClientProtocol

    var body: some View {
        HStack(spacing: 12) {
            RecipeThumbView(
                recipeID: mealPlan?.recipeID,
                apiClient: apiClient,
                size: 48,
                cornerRadius: 10
            )

            VStack(alignment: .leading, spacing: 2) {
                Text(slot.uppercased())
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(.secondary)
                    .kerning(0.4)
                Text(mealPlan?.name ?? "—")
                    .font(.system(size: 16))
                    .foregroundStyle(mealPlan != nil ? .primary : .tertiary)
                    .lineLimit(1)
            }

            Spacer(minLength: 0)

            if mealPlan == nil {
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(minHeight: 72)
    }
}

// MARK: - EditingMeal + MealEditSheet (unchanged)

struct EditingMeal: Identifiable {
    let date: String
    let mealType: String
    var name: String

    var id: String { "\(date)-\(mealType)" }
}

private struct MealEditSheet: View {
    let meal: EditingMeal
    let viewModel: MealsViewModel
    let apiClient: any APIClientProtocol
    @State private var name: String
    @State private var selectedRecipeID: String?
    @State private var recipes: [Recipe] = []
    @State private var isSaving = false
    @State private var recipeSearchQuery = ""
    @Environment(\.dismiss) private var dismiss

    init(meal: EditingMeal, viewModel: MealsViewModel, apiClient: any APIClientProtocol) {
        self.meal = meal
        self.viewModel = viewModel
        self.apiClient = apiClient
        _name = State(initialValue: meal.name)
    }

    private var filteredRecipes: [Recipe] {
        let slotFiltered = recipes.filter { $0.mealType == nil || $0.mealType == meal.mealType }
        guard !recipeSearchQuery.isEmpty else { return slotFiltered }
        return slotFiltered.filter { $0.title.localizedCaseInsensitiveContains(recipeSearchQuery) }
    }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    TextField("Meal name", text: $name)
                } header: {
                    Text(meal.mealType.capitalized)
                }

                Section("Pick a Recipe") {
                    TextField("Search recipes", text: $recipeSearchQuery)
                    ForEach(filteredRecipes) { recipe in
                        Button {
                            name = recipe.title
                            selectedRecipeID = recipe.id
                        } label: {
                            HStack {
                                Text(recipe.title)
                                Spacer()
                                if selectedRecipeID == recipe.id {
                                    Image(systemName: "checkmark")
                                        .foregroundStyle(Color.accentColor)
                                }
                            }
                        }
                        .tint(.primary)
                    }
                }
            }
            .listStyle(.insetGrouped)
            .navigationTitle(meal.name.isEmpty ? "Add Meal" : "Edit Meal")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving {
                        ProgressView()
                    } else {
                        Button("Save") {
                            Task {
                                isSaving = true
                                let saved = await viewModel.saveMeal(
                                    date: meal.date,
                                    mealType: meal.mealType,
                                    name: name,
                                    recipeID: selectedRecipeID
                                )
                                isSaving = false
                                if saved { dismiss() }
                            }
                        }
                        .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                }
            }
        }
        .presentationDetents([.large])
        .task {
            do {
                recipes = try await apiClient.fetchRecipes()
            } catch {
                // Recipes are optional — meal can still be saved with free text
            }
        }
    }
}
