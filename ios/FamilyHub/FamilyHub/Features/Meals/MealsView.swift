import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
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
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
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
                    if plan?.recipeID != nil {
                        RecipeMealRow(
                            mealType: mealType,
                            plan: plan!,
                            apiClient: apiClient,
                            recipesViewModel: recipesViewModel,
                            onEdit: {
                                editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan!.name, recipeID: plan!.recipeID)
                            }
                        )
                        .contextMenu {
                            Button(role: .destructive) {
                                Task { await viewModel.deleteMeal(date: dateKey, mealType: mealType) }
                            } label: {
                                Label("Remove", systemImage: "trash")
                            }
                        }
                    } else {
                        Button {
                            editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan?.name ?? "", recipeID: nil)
                        } label: {
                            MealDayRow(slot: mealType.capitalized, mealPlan: plan, apiClient: apiClient)
                        }
                        .buttonStyle(.plain)
                    }
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

    private var canSave: Bool {
        inputMode == .recipe ? selectedRecipeID != nil : !name.trimmingCharacters(in: .whitespaces).isEmpty
    }

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
                        .disabled(!canSave)
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
                        name = recipe.title
                        selectedRecipeID = recipe.id
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

            TextField("e.g. Wings and veg", text: $name, axis: .vertical)
                .font(.system(size: 18))
                .lineLimit(1...3)
                .padding(14)
                .glassCard(radius: 14)
        }
    }
}
