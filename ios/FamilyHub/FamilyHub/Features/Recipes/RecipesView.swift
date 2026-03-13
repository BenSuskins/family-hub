// ios/FamilyHub/Features/Recipes/RecipesView.swift
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
            Group {
                switch viewModel.state {
                case .idle, .loading:
                    ProgressView()
                case .loaded(let recipes):
                    ScrollView {
                        LazyVGrid(columns: columns, spacing: 16) {
                            ForEach(recipes) { recipe in
                                NavigationLink {
                                    RecipeDetailView(recipe: recipe, apiClient: apiClient)
                                } label: {
                                    RecipeCard(recipe: recipe)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding()
                    }
                case .failed(let error):
                    ContentUnavailableView(
                        "Failed to load",
                        systemImage: "exclamationmark.triangle",
                        description: Text(error.localizedDescription)
                    )
                }
            }
            .navigationTitle("Recipes")
        }
        .task { await viewModel.load() }
    }
}

private struct RecipeCard: View {
    let recipe: Recipe

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            RoundedRectangle(cornerRadius: 8)
                .fill(Color.secondary.opacity(0.2))
                .aspectRatio(4/3, contentMode: .fit)
                .overlay {
                    if !recipe.hasImage {
                        Image(systemName: "fork.knife")
                            .foregroundStyle(.secondary)
                    }
                }

            Text(recipe.title)
                .font(.subheadline.bold())
                .lineLimit(2)

            if let servings = recipe.servings {
                Text("\(servings) servings")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(8)
        .background(Color(.secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}
