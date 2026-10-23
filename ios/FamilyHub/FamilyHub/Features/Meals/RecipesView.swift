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
                if viewModel.refreshError != nil || viewModel.snapshotMetadata?.isFresh == false {
                    Section {
                        HStack {
                            Label(refreshNoticeTitle, systemImage: "arrow.clockwise.circle")
                                .font(.footnote)
                                .foregroundStyle(AppTheme.secondary)
                            Spacer()
                            Button("Retry") { Task { await viewModel.load(forceRefresh: true) } }
                                .font(.footnote.weight(.semibold))
                        }
                    }.listRowBackground(AppTheme.surface)
                }
                Section {
                    Picker("Meal Type", selection: $viewModel.selectedMealType) {
                        Text("All").tag(String?.none)
                        ForEach(RecipesViewModel.mealTypeOptions, id: \.self) { mealType in
                            Text(mealType.capitalized).tag(String?.some(mealType))
                        }
                    }
                }.listRowBackground(AppTheme.surface)
                Section("Recipes") {
                    if viewModel.filteredRecipes.isEmpty {
                        ContentUnavailableView {
                            Label("No matching recipes", systemImage: "magnifyingglass")
                        } description: {
                            Text("Try another search or clear the meal type filter.")
                        } actions: {
                            Button("Clear search and filters") { viewModel.searchQuery = ""; viewModel.selectedMealType = nil }
                        }
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
                                            .foregroundStyle(AppTheme.secondary)
                                    }
                                    let times = [recipe.prepTime.map { "Prep " + $0 }, recipe.cookTime.map { "Cook " + $0 }].compactMap { $0 }
                                    if !times.isEmpty {
                                        Text(times.joined(separator: " · "))
                                            .font(.caption)
                                            .foregroundStyle(AppTheme.secondary)
                                    }
                                }
                                .fixedSize(horizontal: false, vertical: true)
                            }
                            .padding(.vertical, 4)
                        }
                    }
                }.listRowBackground(AppTheme.surface)
                Section {
                    Button("New recipe") { showCreateForm = true }
                        .frame(maxWidth: .infinity, minHeight: 52)
                        .background(AppTheme.sage, in: Capsule())
                        .listRowBackground(Color.clear)
                        .listRowInsets(EdgeInsets())
                }
            }
            .refreshable { await viewModel.load(forceRefresh: true) }
        }
        .appListStyle()
        .navigationTitle("Recipes")
        .searchable(text: $viewModel.searchQuery, placement: .navigationBarDrawer(displayMode: .always), prompt: "Search recipes")
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

    private var refreshNoticeTitle: String {
        guard let snapshot = viewModel.snapshotMetadata, !snapshot.isFresh else {
            return "Couldn’t refresh · Showing saved recipes"
        }
        return "Saved recipes · Last updated \(snapshot.savedAt.formatted(date: .abbreviated, time: .shortened))"
    }
}
