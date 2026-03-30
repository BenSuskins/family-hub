import SwiftUI
import UIKit

struct SearchView: View {
    @State private var viewModel: RecipesViewModel
    private let apiClient: any APIClientProtocol

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
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                case .failed(let error):
                    ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
                case .loaded:
                    if viewModel.filteredRecipes.isEmpty {
                        ContentUnavailableView.search(text: viewModel.searchQuery)
                    } else {
                        List(viewModel.filteredRecipes) { recipe in
                            NavigationLink {
                                RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: viewModel)
                            } label: {
                                RecipeRowView(recipe: recipe, apiClient: apiClient)
                            }
                        }
                        .listStyle(.insetGrouped)
                    }
                }
            }
            .navigationTitle("Search")
            .searchable(text: $viewModel.searchQuery, prompt: "Search recipes")
        }
        .task { await viewModel.load() }
    }
}

private struct RecipeRowView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol

    @State private var imageData: Data?

    var body: some View {
        HStack(spacing: 8) {
            Group {
                if let imageData, let uiImage = UIImage(data: imageData) {
                    Image(uiImage: uiImage)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                } else {
                    RoundedRectangle(cornerRadius: 6)
                        .fill(Color(.tertiarySystemFill))
                        .overlay {
                            Image(systemName: "fork.knife")
                                .font(.subheadline)
                                .foregroundStyle(.tertiary)
                        }
                }
            }
            .frame(width: 44, height: 44)
            .clipShape(RoundedRectangle(cornerRadius: 6))
            .task(id: recipe.id) {
                guard recipe.hasImage else { return }
                imageData = try? await apiClient.fetchRecipeImage(id: recipe.id)
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(recipe.title)
                    .font(.body)
                HStack(spacing: 8) {
                    if let mealType = recipe.mealType {
                        Text(mealType.capitalized)
                            .font(.caption2.weight(.medium))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.accentColor.opacity(0.15))
                            .foregroundStyle(Color.accentColor)
                            .clipShape(Capsule())
                    }
                    if let prep = recipe.prepTime {
                        Label(prep, systemImage: "clock")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if let servings = recipe.servings {
                        Label("\(servings)", systemImage: "person.2")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }
}
