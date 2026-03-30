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

    private var displayRecipe: Recipe { fullRecipe ?? recipe }

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
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                HStack(spacing: 4) {
                    Button {
                        showCookMode = true
                    } label: {
                        Label("Cook", systemImage: "flame")
                    }
                    .disabled(isLoading)

                    Menu {
                        Button {
                            showEditForm = true
                        } label: {
                            Label("Edit", systemImage: "pencil")
                        }
                        .disabled(isLoading)

                        Button(role: .destructive) {
                            showDeleteConfirm = true
                        } label: {
                            Label("Delete", systemImage: "trash")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
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

    private func recipeContent(_ r: Recipe) -> some View {
        List {
            if let imageData, let uiImage = UIImage(data: imageData) {
                Section {
                    Image(uiImage: uiImage)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxHeight: 250)
                        .clipped()
                        .listRowInsets(EdgeInsets())
                }
            }

            Section {
                HStack(spacing: 16) {
                    if let prep = r.prepTime {
                        metaStat(label: "Prep", value: prep)
                    }
                    if let cook = r.cookTime {
                        metaStat(label: "Cook", value: cook)
                    }
                    if let servings = r.servings {
                        metaStat(label: "Serves", value: "\(servings)")
                    }
                }
            }

            if let sourceURL = r.sourceURL, !sourceURL.isEmpty, let url = URL(string: sourceURL) {
                Section("Source") {
                    Link(sourceURL, destination: url)
                        .font(.subheadline)
                        .lineLimit(1)
                }
            }

            if let ingredients = r.ingredients, !ingredients.isEmpty {
                ForEach(ingredients, id: \.name) { group in
                    Section(group.name.isEmpty || group.name == "Main" ? "Ingredients" : group.name) {
                        ForEach(group.items, id: \.self) { item in
                            Text(item)
                                .font(.subheadline)
                        }
                    }
                }
            }

            if let steps = r.steps, !steps.isEmpty {
                Section("Steps") {
                    ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.subheadline.bold())
                                .foregroundStyle(Color.accentColor)
                                .frame(width: 22, alignment: .trailing)
                            Text(step)
                                .font(.subheadline)
                        }
                    }
                }
            }

            Section {
                Button {
                    showCookMode = true
                } label: {
                    Label("Start Cooking", systemImage: "flame.fill")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                }
                .disabled(isLoading)
            }
        }
        .listStyle(.insetGrouped)
    }

    private func metaStat(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }
}
