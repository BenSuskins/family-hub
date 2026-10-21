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

    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        List {
            if isOverview {
                Section {
                    Text("Plan the week. Find a recipe.").foregroundStyle(AppTheme.secondary)
                        .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 0))
                }.listRowBackground(Color.clear)
                Section {
                    NavigationLink(value: FoodDestination.mealPlan) {
                        FeatureEntryLabel(title: "Meal plan", subtitle: "Friday–Thursday", systemImage: "calendar")
                    }
                    .accessibilityLabel("Meal plan")
                    recipesEntry
                }.listRowBackground(AppTheme.surface)
            } else {
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
            }.listRowBackground(AppTheme.surface)
            }
            switch viewModel.state {
            case .idle, .loading:
                Section { ProgressView("Loading meals") }.listRowBackground(AppTheme.surface)
            case .failed(let error):
                Section { ErrorStateView(error: error) { await viewModel.load() } }.listRowBackground(AppTheme.surface)
            case .loaded(let meals):
                ForEach(isFoodHome ? [viewModel.selectedDay] : viewModel.weekDays, id: \.self) { date in
                    Section(isOverview ? "Today’s meals" : date.formatted(.dateTime.weekday(.wide).day().month())) {
                        ForEach(["breakfast", "lunch", "dinner"], id: \.self) { mealType in
                            mealRow(mealType, on: date, meals: meals)
                        }
                    }.listRowBackground(AppTheme.surface)
                    .textCase(nil)
                }
            }
            if isFoodHome && !isOverview {
                Section { recipesEntry }.listRowBackground(AppTheme.surface)
            }
        }
        .listStyle(.insetGrouped)
        .appListStyle()
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
        .onReceive(NotificationCenter.default.publisher(for: .familyHubIntentChanged)) { _ in Task { await viewModel.load() } }
        .onChange(of: scenePhase) { _, phase in if phase == .active { Task { await viewModel.load() } } }
        .task { if isFoodHome { await recipesViewModel.load() } }
    }

    private var isOverview: Bool { isFoodHome && UIDevice.current.userInterfaceIdiom == .phone }

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
                        .foregroundStyle(isSelected ? AppTheme.surface : .primary)
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
                        if isOverview {
                            Text(plan?.name.isEmpty == false ? plan!.name : mealType.capitalized)
                                .font(.body.weight(.medium)).foregroundStyle(AppTheme.ink)
                            Text(plan?.name.isEmpty == false ? mealType.capitalized : "Not planned")
                                .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        } else {
                            Text(mealType.uppercased())
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(AppTheme.secondary)
                            Text(plan?.name.isEmpty == false ? plan!.name : "Add \(mealType)")
                                .font(.headline).foregroundStyle(AppTheme.ink)
                            if let recipe {
                                let times = [recipe.prepTime.map { "Prep " + $0 }, recipe.cookTime.map { "Cook " + $0 }].compactMap { $0 }
                                if !times.isEmpty {
                                    Text(times.joined(separator: " · ")).font(.subheadline).foregroundStyle(AppTheme.secondary)
                                }
                            } else if plan?.name.isEmpty != false {
                                Text("Choose a recipe or write a meal").font(.subheadline).foregroundStyle(AppTheme.secondary)
                            }
                        }
                    }
                    .fixedSize(horizontal: false, vertical: true)
                    if !dynamicTypeSize.isAccessibilitySize { Spacer(minLength: 0) }
                    if let recipe, recipe.hasImage, !isOverview {
                        RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 80, cornerRadius: 12)
                            .accessibilityHidden(true)
                    } else {
                        Image(systemName: plan?.name.isEmpty == false ? (isOverview ? "chevron.right" : "pencil") : "plus")
                            .font(.title2)
                            .foregroundStyle(Color.accentColor)
                            .frame(width: 44, height: 44)
                            .accessibilityHidden(true)
                    }
                }
                .frame(maxWidth: .infinity, minHeight: isOverview ? 52 : 72, alignment: .leading)
                .padding(.vertical, isOverview ? 0 : 8)
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

struct MealDraft {
    var name: String
    private(set) var recipeID: String?

    mutating func select(_ recipe: Recipe) {
        name = recipe.title
        recipeID = recipe.id
    }

    mutating func useNameOnly() { recipeID = nil }
}

struct MealEditSheet: View {
    let meal: EditingMeal
    @Bindable var viewModel: MealsViewModel
    let apiClient: any APIClientProtocol
    @State private var draft: MealDraft
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
        _draft = State(initialValue: MealDraft(name: meal.name, recipeID: meal.recipeID))
    }

    private var filteredRecipes: [Recipe] {
        let query = draft.name.trimmingCharacters(in: .whitespacesAndNewlines)
        return recipes.filter { query.isEmpty || $0.title.localizedCaseInsensitiveContains(query) }
    }

    private var canSave: Bool {
        !draft.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Meal name", text: $draft.name, prompt: Text("What is for \(meal.mealType)?"))
                    .accessibilityLabel("Meal name")
                    .accessibilityIdentifier("Meal name")
                    .autocorrectionDisabled()
                    .submitLabel(.done)
                    .onSubmit { Task { await save() } }
                } header: {
                    Text("Meal name")
                } footer: {
                    Text("Choose a recipe or save the meal name.")
                }.listRowBackground(AppTheme.surface)
                Section("Matching recipes") {
                    if canSave {
                        Button { draft.useNameOnly() } label: {
                            HStack {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("Use “\(draft.name)” only")
                                    Text("No recipe").font(.subheadline).foregroundStyle(AppTheme.secondary)
                                }
                                Spacer()
                                if draft.recipeID == nil { Image(systemName: "checkmark") }
                            }.frame(minHeight: 44)
                        }
                        .disabled(!canSave)
                        .accessibilityLabel("Use name only")
                        .accessibilityAddTraits(draft.recipeID == nil ? .isSelected : [])
                    }
                    if isLoadingRecipes {
                        ProgressView("Loading recipes")
                    } else if let recipeError {
                        Text(recipeError.localizedDescription).foregroundStyle(AppTheme.secondary)
                        Text("You can still save a meal name.").foregroundStyle(AppTheme.secondary)
                        Button("Retry recipes") { Task { await loadRecipes() } }
                    } else if filteredRecipes.isEmpty {
                        Text("No matching recipes. You can save the meal name.")
                            .foregroundStyle(AppTheme.secondary)
                    }
                    ForEach(filteredRecipes) { recipe in
                        Button {
                            draft.select(recipe)
                        } label: {
                            HStack(spacing: 12) {
                                if recipe.hasImage {
                                    RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 44)
                                        .accessibilityHidden(true)
                                }
                                Text(recipe.title).foregroundStyle(AppTheme.ink)
                                    .fixedSize(horizontal: false, vertical: true)
                                Spacer(minLength: 0)
                                if draft.recipeID == recipe.id {
                                    Image(systemName: "checkmark").accessibilityHidden(true)
                                }
                            }
                            .frame(minHeight: 44)
                        }
                        .accessibilityLabel(recipe.title)
                        .accessibilityAddTraits(draft.recipeID == recipe.id ? .isSelected : [])
                    }
                }.listRowBackground(AppTheme.surface)
                if !meal.name.isEmpty || meal.recipeID != nil {
                    Section {
                        Button("Remove Meal", role: .destructive) { confirmRemoval = true }
                    }.listRowBackground(AppTheme.surface)
                }
            }
            .disabled(isSaving)
            .appListStyle()
            .navigationTitle(meal.name.isEmpty ? "Add \(meal.mealType.capitalized)" : "Edit \(meal.mealType.capitalized)")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else {
                        Button("Save") { Task { await save() } }.buttonStyle(AppCommitButtonStyle()).disabled(!canSave)
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
                                             name: draft.name.trimmingCharacters(in: .whitespacesAndNewlines),
                                             recipeID: draft.recipeID)
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
