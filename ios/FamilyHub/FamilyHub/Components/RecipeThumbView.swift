import SwiftUI
import UIKit

/// Rounded thumbnail for a recipe slot in meal rows.
///
/// Shows the recipe image when a recipeID is provided and the image loads.
/// Shows a dashed placeholder with a + icon when recipeID is nil (empty slot).
/// Shows a filled placeholder while loading.
struct RecipeThumbView: View {
    let recipeID: String?
    let apiClient: any APIClientProtocol
    var size: CGFloat = 48
    var cornerRadius: CGFloat = 10

    @State private var imageData: Data?
    @State private var isLoading = false

    var body: some View {
        Group {
            if let data = imageData, let uiImage = UIImage(data: data) {
                Image(uiImage: uiImage)
                    .resizable()
                    .scaledToFill()
            } else if recipeID != nil {
                Color(.quaternarySystemFill)
            } else {
                dashedPlaceholder
            }
        }
        .frame(width: size, height: size)
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
        .task(id: recipeID) {
            imageData = nil
            guard let id = recipeID else { return }
            imageData = try? await apiClient.fetchRecipeImage(id: id)
        }
    }

    private var dashedPlaceholder: some View {
        RoundedRectangle(cornerRadius: cornerRadius)
            .strokeBorder(style: StrokeStyle(lineWidth: 1.5, dash: [4]))
            .foregroundStyle(Color(.tertiaryLabel))
            .overlay {
                Image(systemName: "plus")
                    .font(.system(size: size * 0.35, weight: .regular))
                    .foregroundStyle(Color(.tertiaryLabel))
            }
            .opacity(0.6)
    }
}
