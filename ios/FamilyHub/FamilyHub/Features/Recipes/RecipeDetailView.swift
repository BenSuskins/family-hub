// ios/FamilyHub/Features/Recipes/RecipeDetailView.swift
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
        ZStack {
            Theme.background.ignoresSafeArea()
            Group {
                if isLoading {
                    ProgressView().tint(Theme.textSecondary)
                } else if fetchError && fullRecipe == nil {
                    Text("Failed to load recipe details.")
                        .foregroundStyle(Theme.statusRed)
                        .padding()
                } else {
                    recipeContent(fullRecipe ?? recipe)
                }
            }
        }
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(Theme.background, for: .navigationBar)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    cookModeActive.toggle()
                } label: {
                    Label(cookModeActive ? "Exit Cook Mode" : "Cook Mode",
                          systemImage: cookModeActive ? "flame.fill" : "flame")
                        .font(.system(size: 13))
                        .foregroundStyle(cookModeActive ? Theme.statusAmber : Theme.accent)
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
            // Metadata
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
                .listRowBackground(Theme.surface)
            }

            // Ingredients
            if let ingredients = r.ingredients, !ingredients.isEmpty {
                ForEach(ingredients, id: \.name) { group in
                    Section(group.name.isEmpty ? "Ingredients" : group.name) {
                        ForEach(group.items, id: \.self) { item in
                            Text(item)
                                .font(.system(size: 14))
                                .foregroundStyle(Theme.textPrimary)
                                .listRowBackground(Theme.surface)
                                .listRowSeparatorTint(Theme.borderDivider)
                        }
                    }
                }
            }

            // Steps
            if let steps = r.steps, !steps.isEmpty {
                Section("Steps") {
                    ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.system(size: 13, weight: .bold))
                                .foregroundStyle(Theme.accent)
                                .frame(width: 22, alignment: .trailing)
                            Text(step)
                                .font(.system(size: 14))
                                .foregroundStyle(Theme.textPrimary)
                        }
                        .listRowBackground(Theme.surface)
                        .listRowSeparatorTint(Theme.borderDivider)
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
    }

    private func metaStat(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(Theme.textPrimary)
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(Theme.textMuted)
        }
        .frame(maxWidth: .infinity)
    }
}
