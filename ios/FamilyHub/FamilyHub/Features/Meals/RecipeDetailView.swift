import SwiftUI
import UIKit
import FamilyHubKit

struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol
    @Bindable var viewModel: RecipesViewModel

    @State private var showCookMode = false
    @State private var watchLink = PhoneWatchLink.shared
    @State private var sentToWatch = false
    @State private var fullRecipe: Recipe?
    @State private var isLoading = true
    @State private var loadError: APIError?
    @State private var imageData: Data?

    @State private var showEditForm = false
    @State private var showDeleteConfirm = false

    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
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
            } else if let loadError, fullRecipe == nil {
                ErrorStateView(error: loadError) { await loadRecipe() }
            } else {
                recipeContent(displayRecipe)
            }
        }
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                HStack(spacing: 4) {
                    if let recipeURL {
                        ShareLink(
                            item: recipeURL,
                            subject: Text(recipe.title),
                            message: Text("Check out this recipe: \(recipe.title)")
                        ) {
                            Image(systemName: "square.and.arrow.up")
                        }
                    }

                    Button { showCookMode = true } label: {
                        Image(systemName: "flame")
                    }
                    .accessibilityLabel("Start Cook Mode")
                    .disabled(isLoading || fullRecipe == nil)

                    Menu {
                        if watchLink.canSendToWatch {
                            Button { sendToWatch() } label: {
                                Label("Cook on Watch", systemImage: "applewatch")
                            }
                            .disabled(isLoading || (displayRecipe.steps ?? []).isEmpty)
                        }
                        Button { showEditForm = true } label: {
                            Label("Edit", systemImage: "pencil")
                        }
                        .disabled(isLoading)
                        Button(role: .destructive) { showDeleteConfirm = true } label: {
                            Label("Delete", systemImage: "trash")
                        }
                    } label: {
                        Image(systemName: "ellipsis")
                    }
                    .accessibilityLabel("Recipe actions")
                }
            }
        }
        .fullScreenCover(isPresented: $showCookMode) {
            CookModeView(recipe: displayRecipe)
        }
        .alert(watchLink.lastTransferError == nil ? "Sent to Apple Watch" : "Couldn’t send to Watch",
               isPresented: $sentToWatch) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(watchLink.lastTransferError ?? "Open Family Hub on your watch to start cooking.")
        }
        .sheet(isPresented: $showEditForm) {
            if let r = fullRecipe {
                RecipeFormView(mode: .edit(r), viewModel: viewModel, apiClient: apiClient) { updated in
                    fullRecipe = updated
                    viewModel.beginRecipeVisit()
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
        .errorAlert($viewModel.actionError)
        .task {
            viewModel.beginRecipeVisit()
            await loadRecipe()
        }
    }

    private func loadRecipe() async {
        isLoading = true
        loadError = nil
        do { fullRecipe = try await apiClient.fetchRecipe(id: recipe.id) }
        catch { loadError = .from(error) }
        isLoading = false
        if displayRecipe.hasImage {
            imageData = try? await apiClient.fetchRecipeImage(id: recipe.id)
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
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                            .fixedSize(horizontal: false, vertical: true)
                            .padding(.horizontal, 20)
                            .padding(.top, 20)
                    }
                }

                Spacer(minLength: 80)
            }
        }
        .groupedBackground()
    }

    // MARK: - Hero

    private func heroSection(_ recipe: Recipe) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            if let imageData, let image = UIImage(data: imageData) {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFit()
                    .frame(maxHeight: 280)
                    .frame(maxWidth: .infinity)
                    .accessibilityHidden(true)
            }
            VStack(alignment: .leading, spacing: 8) {
                if let mealType = recipe.mealType {
                    Text(mealType.capitalized).font(.subheadline).foregroundStyle(.secondary)
                }
                Text(recipe.title)
                    .font(.largeTitle.bold())
                    .fixedSize(horizontal: false, vertical: true)
                    .accessibilityAddTraits(.isHeader)
            }
            .padding(.horizontal, 20)
        }
    }

    // MARK: - Meta pills

    private func metaPillsRow(_ r: Recipe) -> some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
            : AnyLayout(HStackLayout(spacing: 8))
        return layout {
            if let prep = r.prepTime {
                metaPill(icon: "clock", label: prep)
            }
            if let cook = r.cookTime {
                metaPill(icon: "flame", label: cook)
            }
            if let servings = r.servings {
                metaPill(icon: "person.2", label: "\(servings) servings")
            }
            if !dynamicTypeSize.isAccessibilitySize { Spacer() }
        }
    }

    private func metaPill(icon: String, label: String) -> some View {
        HStack(spacing: 5) {
            Image(systemName: icon)
                .font(.footnote)
                .foregroundStyle(.secondary)
            Text(label)
                .font(.footnote.weight(.medium))
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 7)
        .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))

    }

    // MARK: - Ingredients

    private func ingredientsSection(_ groups: [IngredientGroup]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Ingredients")
                .font(.title3.weight(.bold))
                .padding(.horizontal, 20)
                .padding(.top, 24)
                .padding(.bottom, 12)

            ForEach(Array(groups.enumerated()), id: \.offset) { groupIndex, group in
                if !group.name.isEmpty && group.name != "Main" {
                    Text(group.name)
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .textCase(.uppercase)
                        .kerning(0.4)
                        .padding(.horizontal, 20)
                        .padding(.top, 8)
                        .padding(.bottom, 4)
                }

                VStack(spacing: 0) {
                    ForEach(Array(group.items.enumerated()), id: \.offset) { itemIndex, item in
                        let index = IngredientIndex(group: groupIndex, item: itemIndex)
                        IngredientRow(item: item, showDivider: itemIndex > 0,
                                      shoppingState: viewModel.ingredientShoppingState(at: index)) {
                            Task { await viewModel.addIngredient(item, at: index) }
                        }
                    }
                }
                .groupedCard(radius: 12)
                .padding(.horizontal, 16)
                .padding(.bottom, 8)
            }
        }
    }

    // MARK: - Steps

    private func stepsSection(_ steps: [String]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Method")
                .font(.title3.weight(.bold))
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
                                .font(.footnote.weight(.bold))
                                .foregroundStyle(.white)
                        }
                        .padding(.top, 1)

                        Text(step)
                            .font(.subheadline)
                            .lineSpacing(3)
                            .fixedSize(horizontal: false, vertical: true)

                        Spacer(minLength: 0)
                    }
                    .padding(14)
                    .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))

                }
            }
            .padding(.horizontal, 16)
        }
    }

    // MARK: - Toolbar circle button

    /// Hand the recipe to the watch. Sends the fully-loaded recipe rather than
    /// the possibly-stub one the row passed in, so the steps and the server's
    /// step timings travel with it.
    private func sendToWatch() {
        // Always confirm, even on failure. The previous version set the flag
        // from the return value, so a refused send produced no feedback at all
        // and looked identical to a successful one.
        watchLink.send(recipe: displayRecipe)
        sentToWatch = true
    }

}

// MARK: - IngredientRow

private struct IngredientRow: View {
    let item: String
    let showDivider: Bool
    let shoppingState: RecipesViewModel.IngredientShoppingState
    let addToShopping: () -> Void
    @State private var checked = false

    var body: some View {
        VStack(spacing: 0) {
            if showDivider {
                Divider().padding(.leading, 46)
            }
            HStack(spacing: 0) {
                Button {
                    withAnimation(.spring(duration: 0.2)) { checked.toggle() }
                } label: {
                    HStack(spacing: 12) {
                        ZStack {
                            Circle()
                                .strokeBorder(checked ? Color.accentColor : Color(UIColor.tertiaryLabel), lineWidth: 1.5)
                                .frame(width: 20, height: 20)
                            if checked {
                                Circle()
                                    .fill(Color.accentColor)
                                    .frame(width: 20, height: 20)
                                Image(systemName: "checkmark")
                                    .font(.caption2.weight(.bold))
                                    .foregroundStyle(.white)
                            }
                        }
                        Text(item)
                            .font(.subheadline)
                            .foregroundStyle(checked ? .tertiary : .primary)
                            .strikethrough(checked, color: .secondary)
                            .fixedSize(horizontal: false, vertical: true)
                        Spacer(minLength: 0)
                    }
                    .padding(.leading, 14)
                    .padding(.vertical, 11)
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel(item)
                .accessibilityValue(checked ? "Checked" : "Not checked")
                .sensoryFeedback(.selection, trigger: checked)

                Button(action: addToShopping) {
                    Group {
                        if shoppingState == .adding {
                            ProgressView()
                        } else {
                            Image(systemName: shoppingState == .added ? "checkmark" : "cart.badge.plus")
                        }
                    }
                    .frame(width: 44, height: 44)
                }
                .buttonStyle(.borderless)
                .padding(.trailing, 6)
                .disabled(shoppingState != .ready)
                .accessibilityLabel(shoppingState == .added ? "Added \(item) to shopping list" : "Add \(item) to shopping list")
                .accessibilityValue(shoppingState == .adding ? "Adding" : "")
            }
        }
    }
}
