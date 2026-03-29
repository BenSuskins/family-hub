import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var editingMeal: EditingMeal?

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

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                switch viewModel.state {
                case .idle, .loading:
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                case .failed(let error):
                    ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
                case .loaded(let meals):
                    List {
                        ForEach(0..<7, id: \.self) { offset in
                            let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                            let dateKey = Self.dateKeyFormatter.string(from: date)
                            Section(Self.dayFormatter.string(from: date)) {
                                ForEach(mealTypes, id: \.self) { mealType in
                                    let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                                    Button {
                                        editingMeal = EditingMeal(date: dateKey, mealType: mealType, name: plan?.name ?? "")
                                    } label: {
                                        HStack {
                                            Text(mealType.capitalized)
                                                .foregroundStyle(.secondary)
                                            Spacer()
                                            Text(plan?.name ?? "—")
                                                .foregroundStyle(plan != nil ? .primary : .tertiary)
                                            Image(systemName: "chevron.right")
                                                .font(.caption)
                                                .foregroundStyle(.tertiary)
                                        }
                                    }
                                    .tint(.primary)
                                    .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                                        if plan != nil {
                                            Button(role: .destructive) {
                                                Task { await viewModel.deleteMeal(date: dateKey, mealType: mealType) }
                                            } label: {
                                                Label("Delete", systemImage: "trash")
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                    .listStyle(.insetGrouped)
                    .refreshable { await viewModel.load() }
                    .sheet(item: $editingMeal) { meal in
                        MealEditSheet(meal: meal, viewModel: viewModel, apiClient: apiClient)
                    }
                }
            }
            .navigationTitle("Meals")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button { viewModel.previousWeek() } label: {
                        Image(systemName: "chevron.left")
                    }
                }
                ToolbarItem(placement: .principal) {
                    Text(weekTitle)
                        .font(.headline)
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    HStack(spacing: 16) {
                        Button("Today") { viewModel.goToCurrentWeek() }
                            .font(.subheadline)
                        Button { viewModel.nextWeek() } label: {
                            Image(systemName: "chevron.right")
                        }
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
}

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
                // Recipes are optional enhancement — meal can still be saved with free text
            }
        }
    }
}
