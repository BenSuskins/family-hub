import SwiftUI
import FamilyHubKit

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var editingMeal: EditingMeal?
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    private let apiClient: any APIClientProtocol
    private let isFoodHome: Bool

    init(apiClient: any APIClientProtocol, isFoodHome: Bool = false, viewModel: MealsViewModel? = nil) {
        self.apiClient = apiClient
        self.isFoodHome = isFoodHome
        _viewModel = State(wrappedValue: viewModel ?? MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            Section {
                weekControls
                if isFoodHome { dayPicker }
            } header: {
                HStack {
                    Text("Meal plan").font(.title3.bold())
                    Spacer()
                    if isFoodHome {
                        NavigationLink {
                            MealsView(apiClient: apiClient, viewModel: viewModel)
                        } label: {
                            Text("View week")
                        }
                        .font(.subheadline)
                        .frame(minHeight: 44)
                    }
                }
                .textCase(nil)
            }
            switch viewModel.state {
            case .idle, .loading:
                Section { ProgressView("Loading meals") }
            case .failed(let error):
                Section { ErrorStateView(error: error) { await viewModel.load() } }
            case .loaded(let meals):
                ForEach(isFoodHome ? [viewModel.selectedDay] : viewModel.weekDays, id: \.self) { date in
                    Section(date.formatted(.dateTime.weekday(.wide).day().month())) {
                        ForEach(["breakfast", "lunch", "dinner"], id: \.self) { mealType in
                            mealRow(mealType, on: date, meals: meals)
                        }
                    }
                    .textCase(nil)
                }
            }
            if isFoodHome {
                Section { recipesEntry }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(isFoodHome ? "Food" : "Meal Plan")
        .toolbar {
            if isFoodHome {
                ToolbarItem(placement: .topBarTrailing) { FeatureProfileButton(apiClient: apiClient) }
            }
        }
        .refreshable {
            await viewModel.load()
            if isFoodHome { await recipesViewModel.load(forceRefresh: true) }
        }
        .sheet(item: $editingMeal) { meal in
            MealEditSheet(meal: meal, viewModel: viewModel, apiClient: apiClient)
        }
        .task { await viewModel.load() }
        .task { if isFoodHome { await recipesViewModel.load() } }
    }

    private var recipesEntry: some View {
        NavigationLink(value: FoodDestination.recipes) {
            HStack(spacing: 8) {
                FeatureEntryLabel(title: "Recipes", subtitle: "Your family cookbook", systemImage: "book")
                Spacer(minLength: 0)
                if !dynamicTypeSize.isAccessibilitySize {
                    ForEach(recipesViewModel.featuredRecipes.filter(\.hasImage).prefix(3)) { recipe in
                        RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 36, cornerRadius: 8)
                            .accessibilityHidden(true)
                    }
                }
            }
        }
        .accessibilityLabel("Recipes")
        .accessibilityHint("Browse your family cookbook")
    }

    private var weekControls: some View {
        HStack(spacing: 8) {
            Button("Previous week", systemImage: "chevron.left") { viewModel.previousWeek() }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
            VStack(spacing: 4) {
                Text(viewModel.currentWeek.formatted(.dateTime.day().month()) + " – " +
                     (viewModel.weekDays.last ?? viewModel.currentWeek).formatted(.dateTime.day().month()))
                    .font(.subheadline.weight(.semibold))
                    .multilineTextAlignment(.center)
                    .accessibilityIdentifier("meal-week-range")
                Button("Today") { viewModel.goToCurrentWeek() }
                    .font(.subheadline)
                    .frame(minHeight: 44)
            }
            .frame(maxWidth: .infinity)
            Button("Next week", systemImage: "chevron.right") { viewModel.nextWeek() }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
        }
        .buttonStyle(.borderless)
    }

    @ViewBuilder
    private var dayPicker: some View {
        if dynamicTypeSize.isAccessibilitySize {
            DatePicker("Meal date", selection: Binding(
                get: { viewModel.selectedDay },
                set: { date in Task { await viewModel.selectDay(date) } }
            ), displayedComponents: .date)
            .accessibilityIdentifier("meal-date-picker")
        } else {
            HStack(spacing: 0) {
                ForEach(viewModel.weekDays, id: \.self) { date in
                    let isSelected = Calendar.current.isDate(date, inSameDayAs: viewModel.selectedDay)
                    Button {
                        Task { await viewModel.selectDay(date) }
                    } label: {
                        VStack(spacing: 6) {
                            Text(date.formatted(.dateTime.weekday(.narrow)))
                                .font(.caption)
                            Text(date.formatted(.dateTime.day()))
                                .font(.body.weight(.semibold))
                        }
                        .foregroundStyle(isSelected ? Color(.systemBackground) : .primary)
                        .frame(maxWidth: .infinity, minHeight: 56)
                        .background(isSelected ? Color.accentColor : .clear, in: RoundedRectangle(cornerRadius: 12))
                    }
                    .buttonStyle(.borderless)
                    .accessibilityLabel(date.formatted(date: .complete, time: .omitted))
                    .accessibilityAddTraits(isSelected ? .isSelected : [])
                }
            }
            .listRowInsets(EdgeInsets(top: 8, leading: 8, bottom: 8, trailing: 8))
        }
    }

    private func mealRow(_ mealType: String, on date: Date, meals: [MealPlan]) -> some View {
        let dateKey = APIDate.dayString(date)
        let plan = meals.first { $0.date == dateKey && $0.mealType == mealType }
        let recipe = recipesViewModel.filteredRecipes.first { $0.id == plan?.recipeID }
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 12))
            : AnyLayout(HStackLayout(spacing: 16))
        return VStack(alignment: .leading, spacing: 12) {
            Button {
                editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan?.name ?? "", recipeID: plan?.recipeID)
            } label: {
                layout {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(mealType.uppercased())
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)
                        Text(plan?.name.isEmpty == false ? plan!.name : "Add \(mealType)")
                            .font(.headline)
                            .foregroundStyle(.primary)
                        if let recipe {
                            let times = [recipe.prepTime.map { "Prep " + $0 }, recipe.cookTime.map { "Cook " + $0 }].compactMap { $0 }
                            if !times.isEmpty {
                                Text(times.joined(separator: " · "))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        } else if plan?.name.isEmpty != false {
                            Text("Choose a recipe or write a meal")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .fixedSize(horizontal: false, vertical: true)
                    if !dynamicTypeSize.isAccessibilitySize { Spacer(minLength: 0) }
                    if let recipe, recipe.hasImage {
                        RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 80, cornerRadius: 12)
                            .accessibilityHidden(true)
                    } else {
                        Image(systemName: plan?.name.isEmpty == false ? "pencil" : "plus.circle.fill")
                            .font(.title2)
                            .foregroundStyle(Color.accentColor)
                            .frame(width: 44, height: 44)
                            .accessibilityHidden(true)
                    }
                }
                .frame(maxWidth: .infinity, minHeight: 72, alignment: .leading)
                .padding(.vertical, 8)
                .contentShape(Rectangle())
            }
            .buttonStyle(.borderless)
            .accessibilityLabel(plan?.name.isEmpty == false ? "Edit \(mealType): \(plan!.name)" : "Add \(mealType)")
            if let plan, let recipeID = plan.recipeID {
                Divider()
                NavigationLink {
                    RecipeDetailView(recipe: recipe ?? Recipe(id: recipeID, title: plan.name), apiClient: apiClient, viewModel: recipesViewModel)
                } label: {
                    Label("View recipe", systemImage: "book")
                        .font(.subheadline)
                        .frame(minHeight: 44)
                }
                .accessibilityLabel("View recipe for \(plan.name)")
            }
        }
    }
}

struct EditingMeal: Identifiable {
    let date: String
    let mealType: String
    var name: String
    var recipeID: String?
    var id: String { "\(date)-\(mealType)" }
}

struct MealEditSheet: View {
    let meal: EditingMeal
    @Bindable var viewModel: MealsViewModel
    let apiClient: any APIClientProtocol
    @State private var name: String
    @State private var selectedRecipeID: String?
    @State private var recipes: [Recipe] = []
    @State private var isSaving = false
    @State private var isLoadingRecipes = true
    @State private var recipeError: APIError?
    @State private var confirmRemoval = false
    @Environment(\.dismiss) private var dismiss

    init(meal: EditingMeal, viewModel: MealsViewModel, apiClient: any APIClientProtocol) {
        self.meal = meal
        self.viewModel = viewModel
        self.apiClient = apiClient
        _name = State(initialValue: meal.name)
        _selectedRecipeID = State(initialValue: meal.recipeID)
    }

    private var filteredRecipes: [Recipe] {
        let query = name.trimmingCharacters(in: .whitespacesAndNewlines)
        return recipes.filter { query.isEmpty || $0.title.localizedCaseInsensitiveContains(query) }
    }

    private var canSave: Bool {
        !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Meal name", text: Binding(
                        get: { name },
                        set: { value in
                            name = value
                            selectedRecipeID = nil
                        }
                    ), prompt: Text("What is for \(meal.mealType)?"))
                    .accessibilityLabel("Meal name")
                    .accessibilityIdentifier("Meal name")
                    .autocorrectionDisabled()
                    .submitLabel(.done)
                    .onSubmit { Task { await save() } }
                    if selectedRecipeID != nil {
                        Label("Recipe linked", systemImage: "checkmark.circle")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        Button("Use name only") { selectedRecipeID = nil }
                    }
                } header: {
                    Text("Meal name")
                } footer: {
                    Text("Choose a recipe below, or press Return to save your own meal.")
                }
                Section("Recipes") {
                    if isLoadingRecipes {
                        ProgressView("Loading recipes")
                    } else if let recipeError {
                        Text(recipeError.localizedDescription).foregroundStyle(.secondary)
                        Text("You can still save a meal name.").foregroundStyle(.secondary)
                        Button("Retry recipes") { Task { await loadRecipes() } }
                    } else if filteredRecipes.isEmpty {
                        Text("No matching recipes. Press Return to save this meal name.")
                            .foregroundStyle(.secondary)
                    }
                    ForEach(filteredRecipes) { recipe in
                        Button {
                            name = recipe.title
                            selectedRecipeID = recipe.id
                        } label: {
                            HStack(spacing: 12) {
                                if recipe.hasImage {
                                    RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 44)
                                        .accessibilityHidden(true)
                                }
                                Text(recipe.title).foregroundStyle(.primary)
                                    .fixedSize(horizontal: false, vertical: true)
                                Spacer(minLength: 0)
                                if selectedRecipeID == recipe.id {
                                    Image(systemName: "checkmark").accessibilityHidden(true)
                                }
                            }
                            .frame(minHeight: 44)
                        }
                        .accessibilityLabel(recipe.title)
                        .accessibilityAddTraits(selectedRecipeID == recipe.id ? .isSelected : [])
                    }
                }
                if !meal.name.isEmpty || meal.recipeID != nil {
                    Section {
                        Button("Remove Meal", role: .destructive) { confirmRemoval = true }
                    }
                }
            }
            .disabled(isSaving)
            .navigationTitle(meal.name.isEmpty ? "Add \(meal.mealType.capitalized)" : "Edit \(meal.mealType.capitalized)")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else {
                        Button("Save") { Task { await save() } }.disabled(!canSave)
                    }
                }
            }
            .errorAlert($viewModel.actionError)
            .confirmationDialog("Remove this meal?", isPresented: $confirmRemoval) {
                Button("Remove Meal", role: .destructive) {
                    Task {
                        isSaving = true
                        let removed = await viewModel.deleteMeal(date: meal.date, mealType: meal.mealType)
                        isSaving = false
                        if removed { dismiss() }
                    }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .task { await loadRecipes() }
        }
    }

    private func save() async {
        guard canSave, !isSaving else { return }
        isSaving = true
        let saved = await viewModel.saveMeal(date: meal.date, mealType: meal.mealType,
                                             name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                                             recipeID: selectedRecipeID)
        isSaving = false
        if saved { dismiss() }
    }

    private func loadRecipes() async {
        isLoadingRecipes = true
        recipeError = nil
        do { recipes = try await apiClient.fetchRecipes() }
        catch { recipeError = .from(error) }
        isLoadingRecipes = false
    }
}
