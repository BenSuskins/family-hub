import SwiftUI
import UIKit

struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol
    let viewModel: RecipesViewModel

    @State private var showCookMode = false
    @State private var fullRecipe: Recipe?
    @State private var isLoading = true
    @State private var fetchError = false
    @State private var imageData: Data?

    @State private var showEditForm = false
    @State private var showDeleteConfirm = false

    @Environment(\.dismiss) private var dismiss
    @Environment(ConfigStore.self) private var configStore

    private var displayRecipe: Recipe { fullRecipe ?? recipe }

    private var recipeURL: URL? {
        URL(string: configStore.baseURL)?
            .appendingPathComponent("recipes")
            .appendingPathComponent(recipe.id)
    }

    var body: some View {
        Group {
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if fetchError && fullRecipe == nil {
                ContentUnavailableView("Failed to load", systemImage: "exclamationmark.triangle")
            } else {
                recipeContent(displayRecipe)
            }
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(.hidden, for: .navigationBar)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                HStack(spacing: 4) {
                    if let recipeURL {
                        ShareLink(
                            item: recipeURL,
                            subject: Text(recipe.title),
                            message: Text("Check out this recipe: \(recipe.title)")
                        ) {
                            toolbarCircleButton(systemImage: "square.and.arrow.up")
                        }
                    }

                    Button { showCookMode = true } label: {
                        toolbarCircleButton(systemImage: "flame")
                    }
                    .disabled(isLoading)

                    Menu {
                        Button { showEditForm = true } label: {
                            Label("Edit", systemImage: "pencil")
                        }
                        .disabled(isLoading)
                        Button(role: .destructive) { showDeleteConfirm = true } label: {
                            Label("Delete", systemImage: "trash")
                        }
                    } label: {
                        toolbarCircleButton(systemImage: "ellipsis")
                    }
                }
            }
        }
        .fullScreenCover(isPresented: $showCookMode) {
            CookModeView(recipe: displayRecipe)
        }
        .sheet(isPresented: $showEditForm) {
            if let r = fullRecipe {
                RecipeFormView(mode: .edit(r), viewModel: viewModel, apiClient: apiClient) { updated in
                    fullRecipe = updated
                    if updated.hasImage {
                        Task { imageData = try? await apiClient.fetchRecipeImage(id: updated.id) }
                    } else {
                        imageData = nil
                    }
                }
            }
        }
        .confirmationDialog(
            "Delete \"\(recipe.title)\"?",
            isPresented: $showDeleteConfirm,
            titleVisibility: .visible
        ) {
            Button("Delete", role: .destructive) {
                Task {
                    let ok = await viewModel.deleteRecipe(id: recipe.id)
                    if ok { dismiss() }
                }
            }
        } message: {
            Text("This recipe will be permanently deleted.")
        }
        .task {
            do {
                fullRecipe = try await apiClient.fetchRecipe(id: recipe.id)
            } catch {
                fetchError = true
            }
            isLoading = false
            if displayRecipe.hasImage {
                imageData = try? await apiClient.fetchRecipeImage(id: recipe.id)
            }
        }
    }

    // MARK: - Main content

    private func recipeContent(_ r: Recipe) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                heroSection(r)
                metaPillsRow(r)
                    .padding(.horizontal, 16)
                    .padding(.top, 14)
                    .padding(.bottom, 6)

                if let ingredients = r.ingredients, !ingredients.isEmpty {
                    ingredientsSection(ingredients)
                }

                if let steps = r.steps, !steps.isEmpty {
                    stepsSection(steps)
                }

                if let sourceURL = r.sourceURL, !sourceURL.isEmpty, let url = URL(string: sourceURL) {
                    Link(destination: url) {
                        Text("Source · \(sourceURL)")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                            .padding(.horizontal, 20)
                            .padding(.top, 20)
                    }
                }

                Spacer(minLength: 80)
            }
        }
        .ignoresSafeArea(edges: .top)
        .meshBackground()
    }

    // MARK: - Hero

    private func heroSection(_ r: Recipe) -> some View {
        ZStack(alignment: .bottom) {
            Color.clear
                .frame(maxWidth: .infinity)
                .frame(height: 280)
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
                                    .font(.system(size: 40))
                                    .foregroundStyle(.tertiary)
                            }
                    }
                }
                .clipped()

            LinearGradient(
                colors: [.clear, .black.opacity(0.65)],
                startPoint: .center,
                endPoint: .bottom
            )
            .frame(height: 160)

            VStack(alignment: .leading, spacing: 4) {
                if let mealType = r.mealType {
                    Text(mealType.uppercased())
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(.white.opacity(0.9))
                        .kerning(0.6)
                }
                Text(r.title)
                    .font(.system(size: 28, weight: .bold))
                    .foregroundStyle(.white)
                    .lineLimit(3)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 20)
            .padding(.bottom, 16)
        }
        .frame(height: 280)
    }

    // MARK: - Meta pills

    private func metaPillsRow(_ r: Recipe) -> some View {
        HStack(spacing: 8) {
            if let prep = r.prepTime {
                metaPill(icon: "clock", label: "\(prep) min")
            }
            if let cook = r.cookTime {
                metaPill(icon: "flame", label: "\(cook) cook")
            }
            if let servings = r.servings {
                metaPill(icon: "person.2", label: "\(servings) servings")
            }
            Spacer()
        }
    }

    private func metaPill(icon: String, label: String) -> some View {
        HStack(spacing: 5) {
            Image(systemName: icon)
                .font(.system(size: 13))
                .foregroundStyle(.secondary)
            Text(label)
                .font(.system(size: 13, weight: .medium))
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 7)
        .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
        .shadow(color: .black.opacity(0.04), radius: 1, x: 0, y: 1)
    }

    // MARK: - Ingredients

    private func ingredientsSection(_ groups: [IngredientGroup]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Ingredients")
                .font(.system(size: 20, weight: .bold))
                .padding(.horizontal, 20)
                .padding(.top, 24)
                .padding(.bottom, 12)

            ForEach(groups, id: \.name) { group in
                if !group.name.isEmpty && group.name != "Main" {
                    Text(group.name)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(.secondary)
                        .textCase(.uppercase)
                        .kerning(0.4)
                        .padding(.horizontal, 20)
                        .padding(.top, 8)
                        .padding(.bottom, 4)
                }

                VStack(spacing: 0) {
                    ForEach(Array(group.items.enumerated()), id: \.element) { index, item in
                        if index > 0 {
                            Divider().padding(.leading, 46)
                        }
                        HStack(spacing: 12) {
                            Circle()
                                .strokeBorder(Color(UIColor.tertiaryLabel), lineWidth: 1.5)
                                .frame(width: 18, height: 18)
                            Text(item)
                                .font(.system(size: 15))
                            Spacer()
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 11)
                        .frame(minHeight: 44)
                    }
                }
                .glassCard(radius: 12)
                .padding(.horizontal, 16)
                .padding(.bottom, 8)
            }
        }
    }

    // MARK: - Steps

    private func stepsSection(_ steps: [String]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Method")
                .font(.system(size: 20, weight: .bold))
                .padding(.horizontal, 20)
                .padding(.top, 24)
                .padding(.bottom, 12)

            VStack(spacing: 10) {
                ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                    HStack(alignment: .top, spacing: 12) {
                        ZStack {
                            Circle()
                                .fill(Color.accentColor)
                                .frame(width: 24, height: 24)
                            Text("\(index + 1)")
                                .font(.system(size: 13, weight: .bold))
                                .foregroundStyle(.white)
                        }
                        .padding(.top, 1)

                        Text(step)
                            .font(.system(size: 15))
                            .lineSpacing(3)
                            .fixedSize(horizontal: false, vertical: true)

                        Spacer(minLength: 0)
                    }
                    .padding(14)
                    .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
                    .shadow(color: .black.opacity(0.04), radius: 1, x: 0, y: 1)
                }
            }
            .padding(.horizontal, 16)
        }
    }

    // MARK: - Toolbar circle button

    private func toolbarCircleButton(systemImage: String) -> some View {
        Image(systemName: systemImage)
            .font(.system(size: 15, weight: .medium))
            .foregroundStyle(.white)
            .frame(width: 32, height: 32)
            .background(.ultraThinMaterial, in: Circle())
    }
}
