import SwiftUI
import UIKit

struct RecipesView: View {
    @State private var viewModel: RecipesViewModel
    private let apiClient: any APIClientProtocol
    private let columns = [
        GridItem(.flexible(), alignment: .top),
        GridItem(.flexible(), alignment: .top)
    ]

    @State private var showCreateForm = false

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                if case .failed(let error) = viewModel.state {
                    ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
                } else {
                    ScrollView {
                        VStack(spacing: 0) {
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 8) {
                                    FilterChip(label: "All", isSelected: viewModel.selectedMealType == nil) {
                                        viewModel.selectedMealType = nil
                                    }
                                    ForEach(RecipesViewModel.mealTypeOptions, id: \.self) { mealType in
                                        FilterChip(label: mealType.capitalized, isSelected: viewModel.selectedMealType == mealType) {
                                            viewModel.selectedMealType = viewModel.selectedMealType == mealType ? nil : mealType
                                        }
                                    }
                                }
                                .padding(.horizontal, 14)
                                .padding(.vertical, 10)
                            }
                            LazyVGrid(columns: columns, spacing: 10) {
                                ForEach(viewModel.filteredRecipes) { recipe in
                                    NavigationLink {
                                        RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: viewModel)
                                    } label: {
                                        RecipeCardView(recipe: recipe, imageData: viewModel.recipeImages[recipe.id])
                                    }
                                    .buttonStyle(ScaleButtonStyle())
                                }
                            }
                            .padding(.horizontal, 14)
                            .padding(.bottom, 8)
                        }
                    }
                    .refreshable { await viewModel.load() }
                }
            }
            .navigationTitle("Recipes")
            .navigationBarTitleDisplayMode(.large)
            .searchable(text: $viewModel.searchQuery, prompt: "Search recipes")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showCreateForm = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showCreateForm) {
                RecipeFormView(mode: .create, viewModel: viewModel, apiClient: apiClient)
            }
        }
        .task { await viewModel.load() }
    }
}

private struct FilterChip: View {
    let label: String
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(label)
                .font(.subheadline.weight(isSelected ? .semibold : .regular))
                .padding(.horizontal, 14)
                .padding(.vertical, 6)
                .foregroundStyle(isSelected ? Color.white : Color.primary)
                .background {
                    Capsule().fill(isSelected ? AnyShapeStyle(Color.accentColor) : AnyShapeStyle(.ultraThinMaterial))
                }
        }
        .buttonStyle(.plain)
        .sensoryFeedback(.selection, trigger: isSelected)
    }
}

private struct RecipeCardView: View {
    let recipe: Recipe
    let imageData: Data?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ZStack {
                Rectangle()
                    .fill(.ultraThinMaterial)
                    .overlay {
                        Image(systemName: "fork.knife")
                            .foregroundStyle(.tertiary)
                            .font(.title2)
                    }
                if let imageData, let uiImage = UIImage(data: imageData) {
                    Image(uiImage: uiImage)
                        .resizable()
                        .scaledToFill()
                }
            }
            .aspectRatio(4/3, contentMode: .fit)
            .frame(maxWidth: .infinity)
            .clipShape(RoundedRectangle(cornerRadius: 10))
            VStack(alignment: .leading, spacing: 4) {
                Text(recipe.title)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(2)
                    .fixedSize(horizontal: false, vertical: true)
                if recipe.prepTime != nil || recipe.servings != nil {
                    HStack(spacing: 8) {
                        if let prep = recipe.prepTime {
                            Label("\(prep) prep", systemImage: "clock")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                        if let servings = recipe.servings {
                            Label("\(servings)", systemImage: "person.2")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }
            .padding(.horizontal, 6)
            .padding(.bottom, 8)
        }
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 14))
    }
}

private struct ScaleButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.96 : 1)
            .animation(.spring(duration: 0.2), value: configuration.isPressed)
    }
}
