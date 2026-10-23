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
                    if isFoodHome {
                        dayPicker
                        NavigationLink("View week") {
                            MealsView(apiClient: apiClient, viewModel: viewModel)
                        }.frame(minHeight: 44)
                    }
                }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
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
            if (viewModel.refreshError != nil || viewModel.snapshotMetadata?.isFresh == false), case .loaded = viewModel.state {
                Section {
                    HStack {
                        Label(refreshNoticeTitle, systemImage: "arrow.clockwise.circle")
                            .font(.footnote)
                            .foregroundStyle(AppTheme.secondary)
                        Spacer()
                        Button("Retry") { Task { await viewModel.load() } }
                            .font(.footnote.weight(.semibold))
                    }
                }.listRowBackground(AppTheme.surface)
            }
            if isFoodHome && !isOverview {
                Section { recipesEntry }.listRowBackground(AppTheme.surface)
            }
        }
        .listStyle(.insetGrouped)
        .appListStyle()
        .navigationTitle(isFoodHome ? "Food" : (viewModel.weekDays.contains(where: { Calendar.current.isDateInToday($0) }) ? "This week" : "Meal plan"))
        .toolbar {
            if !isFoodHome {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Today") { viewModel.goToCurrentWeek() }
                        .accessibilityLabel("Go to current meal week")
                }
            }
        }
        .refreshable {
            await viewModel.load(forceRefresh: true)
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

    private var refreshNoticeTitle: String {
        guard let snapshot = viewModel.snapshotMetadata, !snapshot.isFresh else {
            return "Couldn’t refresh · Showing saved meals"
        }
        return "Saved meals · Last updated \(snapshot.savedAt.formatted(date: .abbreviated, time: .shortened))"
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
                Text(viewModel.currentWeek.formatted(.dateTime.weekday(.wide).day()) + " – " +
                     (viewModel.weekDays.last ?? viewModel.currentWeek).formatted(.dateTime.weekday(.wide).day().month()))
                    .font(.subheadline.weight(.semibold))
                    .multilineTextAlignment(.center)
                    .accessibilityIdentifier("meal-week-range")
                if isFoodHome {
                    Button("Today") { viewModel.goToCurrentWeek() }
                        .frame(minHeight: 44)
                }
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
        let planned = plan?.name.isEmpty == false
        return HStack(alignment: .center, spacing: 8) {
            Button {
                editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan?.name ?? "", recipeID: plan?.recipeID)
            } label: {
                HStack(spacing: 12) {
                    if let recipeID = plan?.recipeID {
                        RecipeThumbView(recipeID: recipeID, apiClient: apiClient, size: 42, placeholderSymbol: "book.closed")
                            .accessibilityHidden(true)
                    }
                    VStack(alignment: .leading, spacing: 4) {
                        Text(planned ? plan!.name : mealType.capitalized)
                            .font(.body.weight(.medium)).foregroundStyle(AppTheme.ink)
                        if planned {
                            let cookingTime = recipe?.cookTime.map { " · " + $0 } ?? ""
                            Text(mealType.capitalized + cookingTime).font(.subheadline).foregroundStyle(AppTheme.secondary)
                        } else {
                            Text("Not planned").font(.subheadline).foregroundStyle(AppTheme.secondary)
                        }
                    }
                }
                .frame(maxWidth: .infinity, minHeight: 52, alignment: .leading)
                .contentShape(Rectangle())
            }
            .buttonStyle(.borderless)
            .accessibilityLabel(planned ? "Edit \(mealType): \(plan!.name)" : "Add \(mealType)")
            .accessibilityValue(plan?.recipeID == nil ? "" : "Recipe linked")
            if let plan, let recipeID = plan.recipeID {
                NavigationLink {
                    RecipeDetailView(recipe: recipe ?? Recipe(id: recipeID, title: plan.name), apiClient: apiClient, viewModel: recipesViewModel)
                } label: {
                    Image(systemName: "chevron.right").frame(width: 44, height: 44)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.borderless)
                .frame(width: 44)
                .accessibilityLabel("View recipe for \(plan.name)")
            } else {
                Button {
                    editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan?.name ?? "", recipeID: plan?.recipeID)
                } label: {
                    Image(systemName: planned ? "pencil" : "plus").frame(width: 44, height: 44)
                }.buttonStyle(.borderless)
                .accessibilityLabel(planned ? "Change \(mealType)" : "Plan \(mealType)")
            }
        }
        .navigationLinkIndicatorVisibility(.hidden)
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
    @FocusState private var nameIsFocused: Bool
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
        let query = Self.searchWords(draft.name)
        guard !query.isEmpty else { return recipes }
        var scoredRecipes: [(recipe: Recipe, score: Int)] = []
        for recipe in recipes {
            guard let score = Self.matchScore(query, against: recipe.title) else { continue }
            scoredRecipes.append((recipe: recipe, score: score))
        }
        scoredRecipes.sort { left, right in
            if left.score != right.score { return left.score < right.score }
            return left.recipe.title.localizedCaseInsensitiveCompare(right.recipe.title) == .orderedAscending
        }
        return scoredRecipes.map(\.recipe)
    }

    private static func searchWords(_ text: String) -> [String] {
        text.lowercased().split { !$0.isLetter && !$0.isNumber }.map(String.init)
    }

    private static func matchScore(_ query: [String], against title: String) -> Int? {
        let normalizedTitle = title.localizedLowercase
        if normalizedTitle.contains(query.joined(separator: " ")) { return 0 }

        let titleWords = searchWords(title)
        var score = 0
        for queryWord in query {
            let bestDistance = titleWords.map { word in
                word.contains(queryWord) ? 0 : editDistance(queryWord, word)
            }.min() ?? Int.max
            let allowedDistance = queryWord.count >= 7 ? 2 : (queryWord.count >= 4 ? 1 : 0)
            guard bestDistance <= allowedDistance else { return nil }
            score += bestDistance
        }
        return score
    }

    private static func editDistance(_ left: String, _ right: String) -> Int {
        let rightCharacters = Array(right)
        var previousRow = Array(0...rightCharacters.count)
        for (leftIndex, leftCharacter) in left.enumerated() {
            var currentRow = [leftIndex + 1]
            for (rightIndex, rightCharacter) in rightCharacters.enumerated() {
                let substitutionCost = leftCharacter == rightCharacter ? 0 : 1
                currentRow.append(min(
                    currentRow[rightIndex] + 1,
                    previousRow[rightIndex + 1] + 1,
                    previousRow[rightIndex] + substitutionCost
                ))
            }
            previousRow = currentRow
        }
        return previousRow.last ?? left.count
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
                    .focused($nameIsFocused)
                    .autocorrectionDisabled()
                    .submitLabel(.done)
                    .onSubmit { Task { await save() } }
                } header: {
                    Text("Meal name")
                }.listRowBackground(AppTheme.surface)
                Section("Matching recipes") {
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
                            nameIsFocused = false
                        } label: {
                            HStack(spacing: 12) {
                                if recipe.hasImage {
                                    RecipeThumbView(recipeID: recipe.id, apiClient: apiClient, size: 44)
                                        .accessibilityHidden(true)
                                }
                                if !recipe.hasImage { WarmIcon(symbol: "fork.knife") }
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(recipe.title).foregroundStyle(AppTheme.ink)
                                    let details = [recipe.mealType?.capitalized, recipe.cookTime, recipe.servings.map { "\($0) servings" }].compactMap { $0 }
                                    Text(details.joined(separator: " · ")).font(.subheadline).foregroundStyle(AppTheme.secondary)
                                }.fixedSize(horizontal: false, vertical: true)
                                Spacer(minLength: 0)
                                Image(systemName: draft.recipeID == recipe.id ? "checkmark" : "chevron.right")
                                    .foregroundStyle(draft.recipeID == recipe.id ? AppTheme.accent : AppTheme.secondary)
                                    .accessibilityHidden(true)
                            }
                            .frame(minHeight: 44)
                        }
                        .accessibilityLabel(recipe.title)
                        .accessibilityAddTraits(draft.recipeID == recipe.id ? .isSelected : [])
                    }
                    if canSave {
                        Button { draft.useNameOnly(); nameIsFocused = false } label: {
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
            .task {
                nameIsFocused = true
                await loadRecipes()
            }
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
