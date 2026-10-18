import SwiftUI
import FamilyHubKit

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var editingMeal: EditingMeal?
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { meals in
            List {
                Section {
                    Text(viewModel.currentWeek.formatted(.dateTime.day().month()) + " – " +
                         Calendar.current.date(byAdding: .day, value: 6, to: viewModel.currentWeek)!.formatted(.dateTime.day().month()))
                        .font(.headline)
                    Text("\(meals.filter { !$0.name.isEmpty }.count) of 21 meals planned")
                        .foregroundStyle(.secondary)
                }
                ForEach(0..<7, id: \.self) { offset in
                    let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                    let dateKey = APIDate.dayString(date)
                    Section(date.formatted(.dateTime.weekday(.wide).day().month())) {
                        ForEach(["breakfast", "lunch", "dinner"], id: \.self) { mealType in
                            let plan = meals.first { $0.date == dateKey && $0.mealType == mealType }
                            if let plan, plan.recipeID != nil {
                                RecipeMealRow(mealType: mealType, plan: plan, apiClient: apiClient,
                                              recipesViewModel: recipesViewModel) {
                                    edit(date: dateKey, mealType: mealType, plan: plan)
                                }
                            } else {
                                Button {
                                    edit(date: dateKey, mealType: mealType, plan: plan)
                                } label: {
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text(mealType.capitalized)
                                            .font(.subheadline)
                                            .foregroundStyle(.secondary)
                                        Text(plan?.name.isEmpty == false ? plan!.name : "Add meal")
                                            .foregroundStyle(.primary)
                                    }
                                    .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                                }
                                .buttonStyle(.borderless)
                            }
                        }
                    }
                }
            }
            .refreshable { await viewModel.load() }
        }
        .navigationTitle("Meal Plan")
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                Button("Previous week", systemImage: "chevron.left") { viewModel.previousWeek() }
                Button("Today") { viewModel.goToCurrentWeek() }
                Button("Next week", systemImage: "chevron.right") { viewModel.nextWeek() }
            }
        }
        .sheet(item: $editingMeal) { meal in
            MealEditSheet(meal: meal, viewModel: viewModel, apiClient: apiClient)
        }
        .task { await viewModel.load() }
    }

    private func edit(date: String, mealType: String, plan: MealPlan?) {
        editingMeal = EditingMeal(date: date, mealType: mealType, name: plan?.name ?? "", recipeID: plan?.recipeID)
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
    enum InputMode: String, CaseIterable { case recipe = "From recipe", text = "Quick text" }

    let meal: EditingMeal
    @Bindable var viewModel: MealsViewModel
    let apiClient: any APIClientProtocol
    @State private var name: String
    @State private var selectedRecipeID: String?
    @State private var recipes: [Recipe] = []
    @State private var isSaving = false
    @State private var isLoadingRecipes = true
    @State private var recipeError: APIError?
    @State private var recipeSearchQuery = ""
    @State private var inputMode: InputMode
    @State private var confirmRemoval = false
    @Environment(\.dismiss) private var dismiss

    init(meal: EditingMeal, viewModel: MealsViewModel, apiClient: any APIClientProtocol) {
        self.meal = meal
        self.viewModel = viewModel
        self.apiClient = apiClient
        _name = State(initialValue: meal.name)
        _selectedRecipeID = State(initialValue: meal.recipeID)
        _inputMode = State(initialValue: meal.recipeID != nil || meal.name.isEmpty ? .recipe : .text)
    }

    private var filteredRecipes: [Recipe] {
        recipes.filter { recipeSearchQuery.isEmpty || $0.title.localizedCaseInsensitiveContains(recipeSearchQuery) }
    }

    private var canSave: Bool {
        inputMode == .recipe ? selectedRecipeID != nil : !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Picker("Meal source", selection: $inputMode) {
                        ForEach(InputMode.allCases, id: \.self) { Text($0.rawValue).tag($0) }
                    }
                }
                if inputMode == .recipe {
                    Section("Recipe") {
                        TextField("Search recipes", text: $recipeSearchQuery)
                            .autocorrectionDisabled()
                        if isLoadingRecipes {
                            ProgressView("Loading recipes")
                        } else if let recipeError {
                            Text(recipeError.localizedDescription).foregroundStyle(.red)
                            Button("Retry") { Task { await loadRecipes() } }
                        } else if filteredRecipes.isEmpty {
                            Text("No recipes found").foregroundStyle(.secondary)
                            if !recipeSearchQuery.trimmingCharacters(in: .whitespaces).isEmpty {
                                Button("Use as a custom meal") {
                                    name = recipeSearchQuery
                                    selectedRecipeID = nil
                                    inputMode = .text
                                }
                            }
                        }
                        ForEach(filteredRecipes) { recipe in
                            Button {
                                selectedRecipeID = recipe.id
                                name = recipe.title
                            } label: {
                                HStack {
                                    Text(recipe.title).foregroundStyle(.primary)
                                    Spacer()
                                    if selectedRecipeID == recipe.id {
                                        Image(systemName: "checkmark").accessibilityHidden(true)
                                    }
                                }
                                .frame(minHeight: 44)
                            }
                            .accessibilityAddTraits(selectedRecipeID == recipe.id ? .isSelected : [])
                        }
                    }
                } else {
                    Section("Meal name") {
                        TextField("What is for \(meal.mealType)?", text: $name, axis: .vertical)
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
        isSaving = true
        let saved = await viewModel.saveMeal(date: meal.date, mealType: meal.mealType,
                                             name: name, recipeID: inputMode == .recipe ? selectedRecipeID : nil)
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
