import SwiftUI
import FamilyHubKit

/// A meal slot row for recipe-linked plans: thumbnail taps open the edit sheet,
/// name taps navigate to the recipe detail.
struct RecipeMealRow: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let mealType: String
    let plan: MealPlan
    let apiClient: any APIClientProtocol
    let recipesViewModel: RecipesViewModel
    let onEdit: () -> Void
    var thumbSize: CGFloat = 48
    var thumbCornerRadius: CGFloat = 10
    var nameFontSize: CGFloat = 16
    var nameFontWeight: Font.Weight = .regular
    var minRowHeight: CGFloat = 72

    private var stub: Recipe {
        Recipe(
            id: plan.recipeID!,
            title: plan.name,
            steps: nil,
            ingredients: nil,
            mealType: plan.mealType,
            servings: nil,
            prepTime: nil,
            cookTime: nil,
            sourceURL: nil,
            categoryID: nil,
            hasImage: false
        )
    }

    var body: some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
            : AnyLayout(HStackLayout(spacing: 0))
        layout {
            Button(action: onEdit) {
                RecipeThumbView(recipeID: plan.recipeID, apiClient: apiClient, size: thumbSize, cornerRadius: thumbCornerRadius)
                    .padding(.leading, 14)
                    .padding(.vertical, 12)
                    .padding(.trailing, 12)
            }
            .buttonStyle(.borderless)
            .accessibilityLabel("Edit \(mealType) meal")

            NavigationLink {
                RecipeDetailView(recipe: stub, apiClient: apiClient, viewModel: recipesViewModel)
            } label: {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(mealType.uppercased())
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(AppTheme.secondary)
                            .kerning(0.4)
                        Text(plan.name)
                            .font(.body.weight(nameFontWeight))
                            .foregroundStyle(AppTheme.ink)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "chevron.right")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.tertiary)
                        .padding(.trailing, 14)
                }
                .frame(maxWidth: .infinity, minHeight: minRowHeight)
            }
            .buttonStyle(.plain)
        }
    }
}
