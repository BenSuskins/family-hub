import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var editingMeal: EditingMeal?

    private static let dateKeyFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "yyyy-MM-dd"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let weekStartFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "MMM d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dayEndFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dayNameFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "EEEE"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dayNumberFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let monthDayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "MMM d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    private var isCurrentWeek: Bool {
        Calendar.current.isDate(viewModel.currentWeek, equalTo: Date(), toGranularity: .weekOfYear)
    }

    private var weekRangeString: String {
        let start = Self.weekStartFormatter.string(from: viewModel.currentWeek)
        let endDate = Calendar.current.date(byAdding: .day, value: 6, to: viewModel.currentWeek)!
        let endDay = Self.dayEndFormatter.string(from: endDate)
        return "\(start) – \(endDay)"
    }

    private func plannedCount(from meals: [MealPlan]) -> Int {
        meals.filter { !$0.name.trimmingCharacters(in: .whitespaces).isEmpty }.count
    }

    private var todayDateKey: String {
        Self.dateKeyFormatter.string(from: Date())
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                editorialHeader(meals: loadedMeals)
                weekNavStrip
                scrollContent
            }
            .meshBackground()
            .navigationBarTitleDisplayMode(.inline)
            .toolbar(.hidden, for: .navigationBar)
            .sheet(item: $editingMeal) { meal in
                MealEditSheet(meal: meal, viewModel: viewModel, apiClient: apiClient)
            }
        }
        .task { await viewModel.load() }
    }

    private var loadedMeals: [MealPlan] {
        if case .loaded(let meals) = viewModel.state { return meals }
        return []
    }

    // MARK: - Editorial Header

    private func editorialHeader(meals: [MealPlan]) -> some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 0) {
                Text(weekRangeString.uppercased())
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(.secondary)
                    .kerning(1.4)
                Text("Plan the week")
                    .font(.system(size: 36, weight: .bold))
                    .foregroundStyle(.primary)
                    .tracking(-1.6)
                    .padding(.top, 8)
                if !meals.isEmpty {
                    Text("\(plannedCount(from: meals)) of 21 meals planned")
                        .font(.system(size: 13))
                        .foregroundStyle(.secondary)
                        .padding(.top, 6)
                }
            }
        }
        .padding(.horizontal, 22)
        .padding(.top, 16)
        .padding(.bottom, 4)
    }

    // MARK: - Week navigation strip

    private var weekNavStrip: some View {
        HStack(spacing: 4) {
            Button {
                viewModel.previousWeek()
            } label: {
                Image(systemName: "chevron.left")
                    .font(.system(size: 14, weight: .semibold))
                    .frame(width: 32, height: 32)
                    .background(Color(UIColor.secondarySystemGroupedBackground), in: Circle())
            }
            .buttonStyle(.plain)

            Spacer()

            if !isCurrentWeek {
                Button("Jump to this week") {
                    viewModel.goToCurrentWeek()
                }
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(Color.accentColor)
            }

            Spacer()

            Button {
                viewModel.nextWeek()
            } label: {
                Image(systemName: "chevron.right")
                    .font(.system(size: 14, weight: .semibold))
                    .frame(width: 32, height: 32)
                    .background(Color(UIColor.secondarySystemGroupedBackground), in: Circle())
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
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
            // Day header
            HStack(alignment: .center, spacing: 8) {
                Text(Self.dayNameFormatter.string(from: date))
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(isToday ? Color.accentColor : .primary)
                Text(Self.dayNumberFormatter.string(from: date))
                    .font(.system(size: 13).monospacedDigit())
                    .foregroundStyle(.tertiary)
                if isToday {
                    Text("TODAY")
                        .font(.system(size: 9.5, weight: .bold))
                        .foregroundStyle(Color.accentColor)
                        .kerning(0.08 * 9.5)
                        .padding(.horizontal, 7)
                        .padding(.vertical, 2)
                        .background(Color.accentColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 6))
                }
            }
            .padding(.horizontal, 20)
            .padding(.top, 14)
            .padding(.bottom, 8)

            // Meal slots card
            VStack(spacing: 0) {
                ForEach(Array(["breakfast", "lunch", "dinner"].enumerated()), id: \.element) { index, mealType in
                    let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                    if index > 0 {
                        Divider().padding(.leading, 74)
                    }
                    MealsSlotRow(mealType: mealType, plan: plan) {
                        editingMeal = EditingMeal(
                            date: dateKey,
                            mealType: mealType,
                            name: plan?.name ?? "",
                            recipeID: plan?.recipeID
                        )
                    }
                    .contextMenu {
                        if plan != nil {
                            Button(role: .destructive) {
                                Task { await viewModel.deleteMeal(date: dateKey, mealType: mealType) }
                            } label: {
                                Label("Remove", systemImage: "trash")
                            }
                        }
                    }
                }
            }
            .glassCard(radius: 18)
            .padding(.horizontal, 16)
        }
    }
}

// MARK: - MealsSlotRow

private struct MealsSlotRow: View {
    let mealType: String
    let plan: MealPlan?
    let onTap: () -> Void

    private var isEmpty: Bool {
        plan == nil || plan!.name.trimmingCharacters(in: .whitespaces).isEmpty
    }

    private var swatchColor: Color {
        switch mealType {
        case "breakfast": return Color(.systemOrange)
        case "lunch":     return Color(.systemGreen)
        default:          return Color(.systemIndigo)
        }
    }

    private var swatchIcon: String {
        switch mealType {
        case "breakfast": return "sun.horizon.fill"
        case "lunch":     return "sun.max.fill"
        default:          return "moon.stars.fill"
        }
    }

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 12) {
                // Colored swatch
                ZStack {
                    RoundedRectangle(cornerRadius: 9)
                        .fill(swatchColor.opacity(0.15))
                        .frame(width: 30, height: 30)
                    Image(systemName: swatchIcon)
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(swatchColor)
                }
                .padding(.leading, 14)

                // Meal type label
                Text(mealType.uppercased())
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(.tertiary)
                    .kerning(0.6)
                    .lineLimit(1)
                    .minimumScaleFactor(0.7)
                    .frame(width: 64, alignment: .leading)

                // Meal name or placeholder
                Text(isEmpty ? "Tap to add" : (plan?.name ?? ""))
                    .font(.system(size: 15, weight: .medium))
                    .foregroundStyle(isEmpty ? .tertiary : .primary)
                    .lineLimit(2)
                    .frame(maxWidth: .infinity, alignment: .leading)

                // Trailing indicator
                Image(systemName: isEmpty ? "plus" : "chevron.right")
                    .font(.caption)
                    .foregroundStyle(isEmpty ? Color.accentColor : Color(UIColor.tertiaryLabel))
                    .padding(.trailing, 14)
            }
            .frame(minHeight: 50)
        }
        .buttonStyle(.plain)
    }
}

// MARK: - EditingMeal + MealEditSheet (unchanged)

struct EditingMeal: Identifiable {
    let date: String
    let mealType: String
    var name: String
    var recipeID: String?

    var id: String { "\(date)-\(mealType)" }
}

struct MealEditSheet: View {
    enum InputMode { case recipe, text }

    let meal: EditingMeal
    let viewModel: MealsViewModel
    let apiClient: any APIClientProtocol
    @State private var name: String
    @State private var selectedRecipeID: String?
    @State private var recipes: [Recipe] = []
    @State private var isSaving = false
    @State private var recipeSearchQuery = ""
    @State private var inputMode: InputMode = .recipe
    @Environment(\.dismiss) private var dismiss

    init(meal: EditingMeal, viewModel: MealsViewModel, apiClient: any APIClientProtocol) {
        self.meal = meal
        self.viewModel = viewModel
        self.apiClient = apiClient
        _name = State(initialValue: meal.name)
        _selectedRecipeID = State(initialValue: meal.recipeID)
        _inputMode = State(initialValue: meal.recipeID != nil ? .recipe : (meal.name.isEmpty ? .recipe : .text))
    }

    private var filteredRecipes: [Recipe] {
        guard !recipeSearchQuery.isEmpty else { return recipes }
        return recipes.filter { $0.title.localizedCaseInsensitiveContains(recipeSearchQuery) }
    }

    private var canSave: Bool { true }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    modeToggle
                    if inputMode == .recipe {
                        recipePickerSection
                    } else {
                        textInputSection
                    }
                }
                .padding(.horizontal, 16)
                .padding(.top, 8)
                .padding(.bottom, 32)
            }
            .meshBackground()
            .navigationTitle(meal.name.isEmpty ? "Add \(meal.mealType.capitalized)" : "Edit \(meal.mealType.capitalized)")
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
                                let isNowEmpty = inputMode == .recipe
                                    ? selectedRecipeID == nil
                                    : name.trimmingCharacters(in: .whitespaces).isEmpty
                                let wasEmpty = meal.name.isEmpty && meal.recipeID == nil
                                if isNowEmpty && wasEmpty {
                                    dismiss()
                                } else if isNowEmpty {
                                    isSaving = true
                                    let deleted = await viewModel.deleteMeal(date: meal.date, mealType: meal.mealType)
                                    isSaving = false
                                    if deleted { dismiss() }
                                } else {
                                    isSaving = true
                                    let saved = await viewModel.saveMeal(
                                        date: meal.date,
                                        mealType: meal.mealType,
                                        name: name,
                                        recipeID: inputMode == .recipe ? selectedRecipeID : nil
                                    )
                                    isSaving = false
                                    if saved { dismiss() }
                                }
                            }
                        }
                        .fontWeight(.semibold)
                    }
                }
            }
        }
        .presentationDetents([.large])
        .task {
            do {
                recipes = try await apiClient.fetchRecipes()
            } catch {}
        }
    }

    // MARK: - Mode toggle

    private var modeToggle: some View {
        HStack(spacing: 10) {
            ForEach([InputMode.recipe, InputMode.text], id: \.self) { mode in
                let isSelected = inputMode == mode
                Button {
                    withAnimation(.spring(duration: 0.2)) { inputMode = mode }
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: mode == .recipe ? "book" : "pencil")
                            .font(.system(size: 15, weight: .medium))
                            .foregroundStyle(isSelected ? Color.accentColor : .secondary)
                        Text(mode == .recipe ? "From recipe" : "Quick text")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(isSelected ? Color.accentColor : .primary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                    .background(
                        RoundedRectangle(cornerRadius: 14)
                            .fill(isSelected
                                  ? Color.accentColor.opacity(0.12)
                                  : Color(UIColor.secondarySystemGroupedBackground))
                            .overlay(
                                RoundedRectangle(cornerRadius: 14)
                                    .strokeBorder(isSelected ? Color.accentColor.opacity(0.5) : Color.clear, lineWidth: 1)
                            )
                    )
                }
                .buttonStyle(.plain)
                .sensoryFeedback(.selection, trigger: isSelected)
            }
        }
    }

    // MARK: - Recipe picker

    private var recipePickerSection: some View {
        VStack(spacing: 0) {
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
                    .font(.system(size: 14))
                TextField("Search recipes", text: $recipeSearchQuery)
                    .autocorrectionDisabled()
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
            .padding(.bottom, 10)

            VStack(spacing: 0) {
                ForEach(Array(filteredRecipes.enumerated()), id: \.element.id) { index, recipe in
                    if index > 0 {
                        Divider().padding(.leading, 74)
                    }
                    Button {
                        if selectedRecipeID == recipe.id {
                            selectedRecipeID = nil
                            name = ""
                        } else {
                            name = recipe.title
                            selectedRecipeID = recipe.id
                        }
                    } label: {
                        HStack(spacing: 12) {
                            RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 48, cornerRadius: 10)

                            VStack(alignment: .leading, spacing: 2) {
                                Text(recipe.title)
                                    .font(.system(size: 16, weight: .medium))
                                    .foregroundStyle(.primary)
                                    .lineLimit(1)
                                HStack(spacing: 4) {
                                    if let prep = recipe.prepTime {
                                        Text("\(prep) min")
                                            .font(.system(size: 13))
                                            .foregroundStyle(.secondary)
                                    }
                                    if let mealType = recipe.mealType {
                                        Text("·")
                                            .foregroundStyle(.secondary)
                                        Text(mealType.capitalized)
                                            .font(.system(size: 13))
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }

                            Spacer(minLength: 0)

                            CheckCircleView(
                                isSelected: selectedRecipeID == recipe.id,
                                size: 22,
                                color: .accentColor,
                                unselectedBorderColor: Color(UIColor.tertiaryLabel)
                            )
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 11)
                        .frame(minHeight: 68)
                    }
                    .buttonStyle(.plain)
                }

                if filteredRecipes.isEmpty {
                    Text(recipes.isEmpty ? "Loading…" : "No recipes found")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 24)
                }
            }
            .glassCard(radius: 14)
        }
    }

    // MARK: - Text input

    private var textInputSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("What's for \(meal.mealType.lowercased())?")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(.secondary)
                .textCase(.uppercase)
                .kerning(0.4)
                .padding(.horizontal, 4)

            HStack(alignment: .top, spacing: 0) {
                TextField("e.g. Wings and veg", text: $name, axis: .vertical)
                    .font(.system(size: 18))
                    .lineLimit(1...3)
                if !name.isEmpty {
                    Button {
                        name = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(Color(UIColor.tertiaryLabel))
                            .font(.system(size: 18))
                    }
                    .buttonStyle(.plain)
                    .padding(.top, 2)
                }
            }
            .padding(14)
            .glassCard(radius: 14)
        }
    }
}
