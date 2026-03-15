import SwiftUI

struct RecipesView: View {
    @State private var viewModel: RecipesViewModel
    private let apiClient: any APIClientProtocol
    private let columns = [GridItem(.flexible()), GridItem(.flexible())]

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                Group {
                    if case .failed(let error) = viewModel.state {
                        VStack {
                            Text(error.localizedDescription)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.statusRed)
                                .padding()
                            Spacer()
                        }
                    } else {
                        ScrollView {
                            LazyVGrid(columns: columns, spacing: 10) {
                                ForEach(viewModel.filteredRecipes) { recipe in
                                    NavigationLink {
                                        RecipeDetailView(recipe: recipe, apiClient: apiClient)
                                    } label: {
                                        RecipeCard(recipe: recipe)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                            .padding(.horizontal, 14)
                            .padding(.top, 8)
                        }
                        .refreshable { await viewModel.load() }
                    }
                }
            }
            .navigationTitle("Recipes")
            .navigationBarTitleDisplayMode(.large)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .searchable(text: $viewModel.searchQuery, prompt: "Search recipes")
        }
        .task { await viewModel.load() }
    }
}

private struct RecipeCard: View {
    let recipe: Recipe

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            RoundedRectangle(cornerRadius: 8)
                .fill(Theme.surfaceElevated)
                .aspectRatio(4/3, contentMode: .fit)
                .overlay {
                    Image(systemName: "fork.knife")
                        .foregroundStyle(Theme.textMuted)
                        .font(.system(size: 22))
                }
            VStack(alignment: .leading, spacing: 4) {
                Text(recipe.title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(Theme.textPrimary)
                    .lineLimit(2)
                HStack(spacing: 8) {
                    if let prep = recipe.prepTime {
                        Label("\(prep) prep", systemImage: "clock")
                            .font(.system(size: 10))
                            .foregroundStyle(Theme.textMuted)
                    }
                    if let servings = recipe.servings {
                        Label("\(servings)", systemImage: "person.2")
                            .font(.system(size: 10))
                            .foregroundStyle(Theme.textMuted)
                    }
                }
            }
            .padding(.horizontal, 6)
            .padding(.bottom, 8)
        }
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
