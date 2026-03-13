// ios/FamilyHub/Features/Recipes/RecipeDetailView.swift
import SwiftUI

struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol

    @State private var cookMode = false
    @State private var fullRecipe: Recipe?
    @State private var isLoading = true

    var body: some View {
        let displayed = fullRecipe ?? recipe
        List {
            // Meta section
            Section {
                if let servings = displayed.servings {
                    LabeledContent("Servings", value: "\(servings)")
                }
                if let prep = displayed.prepTime {
                    LabeledContent("Prep time", value: prep)
                }
                if let cook = displayed.cookTime {
                    LabeledContent("Cook time", value: cook)
                }
            }

            // Ingredients
            if let ingredients = displayed.ingredients, !ingredients.isEmpty {
                ForEach(ingredients, id: \.name) { group in
                    Section(group.name.isEmpty ? "Ingredients" : group.name) {
                        ForEach(group.items, id: \.self) { item in
                            Text(item)
                        }
                    }
                }
            }

            // Steps
            if let steps = displayed.steps, !steps.isEmpty {
                Section("Steps") {
                    ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.headline)
                                .foregroundStyle(.secondary)
                                .frame(width: 24)
                            Text(step)
                        }
                    }
                }
            }
        }
        .navigationTitle(displayed.title)
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(cookMode ? "Exit Cook Mode" : "Cook Mode") {
                    cookMode.toggle()
                    UIApplication.shared.isIdleTimerDisabled = cookMode
                }
            }
        }
        .task {
            // Fetch full recipe details (list endpoint may omit steps/ingredients)
            if let full = try? await apiClient.fetchRecipe(id: recipe.id) {
                fullRecipe = full
            }
            isLoading = false
        }
        .overlay {
            if isLoading { ProgressView() }
        }
    }
}
