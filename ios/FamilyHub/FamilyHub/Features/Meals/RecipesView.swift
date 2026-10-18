import SwiftUI
import FamilyHubKit

struct RecipesView: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    @State private var viewModel: RecipesViewModel
    @State private var showCreateForm = false
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { _ in
            List {
                Section {
                    Picker("Meal Type", selection: $viewModel.selectedMealType) {
                        Text("All").tag(String?.none)
                        ForEach(RecipesViewModel.mealTypeOptions, id: \.self) { mealType in
                            Text(mealType.capitalized).tag(String?.some(mealType))
                        }
                    }
                }
                Section("Recipes") {
                    if viewModel.filteredRecipes.isEmpty {
                        ContentUnavailableView.search(text: viewModel.searchQuery)
                    }
                    ForEach(viewModel.filteredRecipes) { recipe in
                        NavigationLink {
                            RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: viewModel)
                        } label: {
                            let layout = dynamicTypeSize.isAccessibilitySize
                                ? AnyLayout(VStackLayout(alignment: .leading, spacing: 12))
                                : AnyLayout(HStackLayout(alignment: .top, spacing: 12))
                            layout {
                                RecipeThumbView(recipeID: recipe.hasImage ? recipe.id : nil, apiClient: apiClient, size: 60)
                                    .accessibilityHidden(true)
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(recipe.title)
                                        .font(.headline)
                                    if let mealType = recipe.mealType {
                                        Text(mealType.capitalized)
                                            .font(.subheadline)
                                            .foregroundStyle(.secondary)
                                    }
                                    let times = [recipe.prepTime, recipe.cookTime].compactMap { $0 }
                                    if !times.isEmpty {
                                        Text(times.joined(separator: " · "))
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                                .fixedSize(horizontal: false, vertical: true)
                            }
                            .padding(.vertical, 4)
                        }
                    }
                }
            }
            .refreshable { await viewModel.load(forceRefresh: true) }
        }
        .navigationTitle("Recipes")
        .searchable(text: $viewModel.searchQuery, prompt: "Search recipes")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("New Recipe", systemImage: "plus") { showCreateForm = true }
            }
        }
        .sheet(isPresented: $showCreateForm) {
            RecipeFormView(mode: .create, viewModel: viewModel, apiClient: apiClient)
        }
        .errorAlert($viewModel.actionError)
        .task { await viewModel.load() }
    }
}
