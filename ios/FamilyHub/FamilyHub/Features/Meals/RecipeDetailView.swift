import SwiftUI
import UIKit

struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol

    @State private var cookModeActive = false
    @State private var fullRecipe: Recipe?
    @State private var isLoading = true
    @State private var fetchError = false

    var body: some View {
        Group {
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if fetchError && fullRecipe == nil {
                ContentUnavailableView("Failed to load", systemImage: "exclamationmark.triangle")
            } else {
                recipeContent(fullRecipe ?? recipe)
            }
        }
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    cookModeActive.toggle()
                } label: {
                    Label(cookModeActive ? "Exit Cook Mode" : "Cook Mode",
                          systemImage: cookModeActive ? "flame.fill" : "flame")
                        .foregroundStyle(cookModeActive ? .orange : .accentColor)
                }
            }
        }
        .onDisappear {
            UIApplication.shared.isIdleTimerDisabled = false
        }
        .onChange(of: cookModeActive) { _, active in
            UIApplication.shared.isIdleTimerDisabled = active
        }
        .task {
            do {
                fullRecipe = try await apiClient.fetchRecipe(id: recipe.id)
            } catch {
                fetchError = true
            }
            isLoading = false
        }
    }

    private func recipeContent(_ r: Recipe) -> some View {
        List {
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

            if let ingredients = r.ingredients, !ingredients.isEmpty {
                ForEach(ingredients, id: \.name) { group in
                    Section(group.name.isEmpty ? "Ingredients" : group.name) {
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
