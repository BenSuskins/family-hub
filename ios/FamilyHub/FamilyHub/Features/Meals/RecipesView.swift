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
    @State private var isSearching = false

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                if case .failed(let error) = viewModel.state {
                    ErrorStateView(error: error) { await viewModel.load() }
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
                    .refreshable { await viewModel.load(forceRefresh: true) }
                }
            }
            .meshBackground()
            .navigationTitle("Recipes")
            .navigationBarTitleDisplayMode(.large)
            .searchable(text: $viewModel.searchQuery, isPresented: $isSearching, prompt: "Search recipes")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    HStack(spacing: 4) {
                        Button {
                            isSearching = true
                        } label: {
                            Image(systemName: "magnifyingglass")
                        }
                        Button {
                            showCreateForm = true
                        } label: {
                            Image(systemName: "plus")
                        }
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

    // MARK: - Featured horizontal scroll (up to 8 recipes)

    private var featuredRow: some View {
        let featured = viewModel.featuredRecipes
        return ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 12) {
                ForEach(featured) { recipe in
                    NavigationLink {
                        RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: viewModel)
                    } label: {
                        RecipeCard(recipe: recipe, imageData: viewModel.recipeImages[recipe.id])
                            .frame(width: 250)
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
                        RecipeCard(
                            recipe: recipe,
                            imageData: viewModel.recipeImages[recipe.id],
                            imageHeight: 120,
                            titleFontSize: 14,
                            subtitleFontSize: 11
                        )
                    }
                    .buttonStyle(ScaleButtonStyle())
                }
            }
            .padding(.horizontal, 14)
        }
    }
}

// MARK: - Recipe card (used in both featured scroll and grid)

private struct RecipeCard: View {
    let recipe: Recipe
    let imageData: Data?
    var imageHeight: CGFloat = 160
    var titleFontSize: CGFloat = 16
    var subtitleFontSize: CGFloat = 12

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            heroImage
                .frame(maxWidth: .infinity)
                .frame(height: imageHeight)
                .clipped()

            VStack(alignment: .leading, spacing: 3) {
                Text(recipe.title)
                    .font(.system(size: titleFontSize, weight: .semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(2, reservesSpace: true)
                    .multilineTextAlignment(.leading)

                Text(subtitleString)
                    .font(.system(size: subtitleFontSize))
                    .foregroundStyle(.secondary)
                    .lineLimit(1, reservesSpace: true)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(UIColor.secondarySystemGroupedBackground))
        }
        .clipShape(RoundedRectangle(cornerRadius: 14))
        .shadow(color: .black.opacity(0.08), radius: 4, x: 0, y: 2)
    }

    private var subtitleString: String {
        let timeParts = [recipe.prepTime, recipe.cookTime].compactMap { $0 }
        let timeString = timeParts.joined(separator: " · ")
        if !timeString.isEmpty, let mealType = recipe.mealType {
            return "\(timeString) · \(mealType.capitalized)"
        } else if !timeString.isEmpty {
            return timeString
        } else if let mealType = recipe.mealType {
            return mealType.capitalized
        }
        return ""
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
