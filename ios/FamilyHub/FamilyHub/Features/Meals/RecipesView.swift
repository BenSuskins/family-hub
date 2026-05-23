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
                    ContentUnavailableView(
                        "Error",
                        systemImage: "exclamationmark.triangle",
                        description: Text(error.localizedDescription)
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 0) {
                            filterChips

                            if !viewModel.filteredRecipes.isEmpty {
                                featuredRow
                                allRecipesGrid
                            }
                        }
                        .padding(.bottom, 24)
                    }
                    .refreshable { await viewModel.load() }
                }
            }
            .meshBackground()
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

    // MARK: - Filter chips

    private var filterChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                FilterChip(label: "All", isSelected: viewModel.selectedMealType == nil) {
                    viewModel.selectedMealType = nil
                }
                ForEach(RecipesViewModel.mealTypeOptions, id: \.self) { mealType in
                    FilterChip(
                        label: mealType.capitalized,
                        isSelected: viewModel.selectedMealType == mealType
                    ) {
                        viewModel.selectedMealType = viewModel.selectedMealType == mealType ? nil : mealType
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
        }
    }

    // MARK: - Featured horizontal scroll (first 2 recipes)

    private var featuredRow: some View {
        let featured = Array(viewModel.filteredRecipes.prefix(2))
        return ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 12) {
                ForEach(featured) { recipe in
                    NavigationLink {
                        RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: viewModel)
                    } label: {
                        FeaturedRecipeCard(recipe: recipe, imageData: viewModel.recipeImages[recipe.id])
                    }
                    .buttonStyle(ScaleButtonStyle())
                }
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 8)
        }
    }

    // MARK: - All recipes grid

    private var allRecipesGrid: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("All Recipes")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(.secondary)
                .textCase(.uppercase)
                .kerning(0.4)
                .padding(.horizontal, 20)
                .padding(.top, 14)
                .padding(.bottom, 8)

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
        }
    }
}

// MARK: - Featured card (large, horizontal scroll)

private struct FeaturedRecipeCard: View {
    let recipe: Recipe
    let imageData: Data?

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            ZStack(alignment: .bottom) {
                heroImage
                    .frame(width: 250, height: 140)
                    .clipped()

                LinearGradient(
                    colors: [.clear, .black.opacity(0.55)],
                    startPoint: .center,
                    endPoint: .bottom
                )
                .frame(height: 80)

                VStack(alignment: .leading, spacing: 2) {
                    Text(recipe.title)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(.white)
                        .lineLimit(2)
                    HStack(spacing: 6) {
                        if let prep = recipe.prepTime {
                            Text("\(prep) min")
                                .font(.system(size: 12))
                                .foregroundStyle(.white.opacity(0.9))
                        }
                        if let mealType = recipe.mealType {
                            Text("·")
                                .foregroundStyle(.white.opacity(0.7))
                            Text(mealType.capitalized)
                                .font(.system(size: 12))
                                .foregroundStyle(.white.opacity(0.9))
                        }
                    }
                }
                .padding(.horizontal, 12)
                .padding(.bottom, 12)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .frame(width: 250)
        .clipShape(RoundedRectangle(cornerRadius: 14))
        .shadow(color: .black.opacity(0.08), radius: 4, x: 0, y: 2)
    }

    private var heroImage: some View {
        Group {
            if let imageData, let uiImage = UIImage(data: imageData) {
                Image(uiImage: uiImage)
                    .resizable()
                    .scaledToFill()
            } else {
                Rectangle()
                    .fill(.ultraThinMaterial)
                    .overlay {
                        Image(systemName: "fork.knife")
                            .foregroundStyle(.tertiary)
                            .font(.title)
                    }
            }
        }
    }
}

// MARK: - Grid card

private struct RecipeCardView: View {
    let recipe: Recipe
    let imageData: Data?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Color.clear
                .aspectRatio(4/3, contentMode: .fit)
                .overlay {
                    if let imageData, let uiImage = UIImage(data: imageData) {
                        Image(uiImage: uiImage)
                            .resizable()
                            .scaledToFill()
                    } else {
                        Rectangle()
                            .fill(.ultraThinMaterial)
                            .overlay {
                                Image(systemName: "fork.knife")
                                    .foregroundStyle(.tertiary)
                                    .font(.title2)
                            }
                    }
                }
                .clipped()
                .clipShape(RoundedRectangle(cornerRadius: 10))

            VStack(alignment: .leading, spacing: 4) {
                Text(recipe.title)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(2)
                    .frame(minHeight: 36, alignment: .topLeading)
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
                .frame(minHeight: 14)
            }
            .padding(.horizontal, 6)
            .padding(.bottom, 8)
        }
        .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
        .shadow(color: .black.opacity(0.04), radius: 2, x: 0, y: 1)
    }
}

// MARK: - FilterChip

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

// MARK: - ScaleButtonStyle

private struct ScaleButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.96 : 1)
            .animation(.spring(duration: 0.2), value: configuration.isPressed)
    }
}
