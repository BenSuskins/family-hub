import SwiftUI
import UIKit

/// Rounded thumbnail for a recipe slot in meal rows.
///
/// - Recipe with loaded image: shows the image.
/// - Recipe loading: filled quaternary background.
/// - No recipe, placeholderText provided: first letter on a muted background.
/// - No recipe, no text (empty slot): dashed border with + icon.
struct RecipeThumbView: View {
    let recipeID: String?
    let apiClient: any APIClientProtocol
    var size: CGFloat = 48
    var cornerRadius: CGFloat = 10
    var placeholderText: String? = nil

    @State private var imageData: Data?

    var body: some View {
        Group {
            if let data = imageData, let uiImage = UIImage(data: data) {
                Image(uiImage: uiImage)
                    .resizable()
                    .scaledToFill()
            } else if recipeID != nil {
                Color(.quaternarySystemFill)
            } else if let initial = placeholderText?.first.map(String.init), !initial.isEmpty {
                initialsPlaceholder(initial)
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

    private func initialsPlaceholder(_ letter: String) -> some View {
        Color(.tertiarySystemFill)
            .overlay {
                Text(letter.uppercased())
                    .font(.system(size: size * 0.4, weight: .semibold, design: .rounded))
                    .foregroundStyle(Color(.secondaryLabel))
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
