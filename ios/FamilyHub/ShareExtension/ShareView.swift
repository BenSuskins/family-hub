import SwiftUI
import UIKit

// Duplicated here since the extension cannot import the main app module
private enum ShareMealType: String, CaseIterable {
    case breakfast
    case lunch
    case dinner
    case side
    case dessert

    var displayName: String {
        rawValue.capitalized
    }
}

private struct ShareRecipeRequest: Encodable {
    var title: String
    var steps: [String]
    var ingredients: [ShareIngredientGroup]
    var mealType: String?
    var servings: Int?
    var prepTime: String?
    var cookTime: String?
    var sourceURL: String?
    var imageData: String?
}

private struct ShareIngredientGroup: Encodable {
    var name: String
    var items: [String]
}

struct ShareView: View {
    let sharedURL: URL
    let baseURL: String
    let apiToken: String
    var onDismiss: () -> Void

    @State private var title = ""
    @State private var selectedMealType: ShareMealType?
    @State private var ogImageDataURI: String?
    @State private var ingredients: [String] = []
    @State private var steps: [String] = []
    @State private var prepTime: String?
    @State private var cookTime: String?
    @State private var servings: Int?
    @State private var isLoadingOG = true
    @State private var isSaving = false
    @State private var errorMessage: String?
    @State private var savedSuccessfully = false

    var body: some View {
        NavigationStack {
            Form {
                if isLoadingOG {
                    Section {
                        HStack {
                            ProgressView()
                            Text("Fetching page info…")
                                .foregroundStyle(.secondary)
                                .padding(.leading, 8)
                        }
                    }
                }

                Section("Recipe Title") {
                    TextField("Title", text: $title)
                }

                Section("Meal Type") {
                    Picker("Type", selection: $selectedMealType) {
                        Text("None").tag(ShareMealType?.none)
                        ForEach(ShareMealType.allCases, id: \.self) { type in
                            Text(type.displayName).tag(ShareMealType?.some(type))
                        }
                    }
                    .pickerStyle(.menu)
                }

                if !ingredients.isEmpty {
                    Section("Ingredients (\(ingredients.count))") {
                        ForEach(ingredients, id: \.self) { ingredient in
                            Text(ingredient)
                                .font(.subheadline)
                        }
                    }
                }

                if !steps.isEmpty {
                    Section("Steps (\(steps.count))") {
                        ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                            HStack(alignment: .top, spacing: 8) {
                                Text("\(index + 1).")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 24, alignment: .trailing)
                                Text(step)
                                    .font(.subheadline)
                            }
                        }
                    }
                }

                if prepTime != nil || cookTime != nil || servings != nil {
                    Section("Details") {
                        if let prepTime {
                            LabeledContent("Prep Time", value: prepTime)
                                .font(.subheadline)
                        }
                        if let cookTime {
                            LabeledContent("Cook Time", value: cookTime)
                                .font(.subheadline)
                        }
                        if let servings {
                            LabeledContent("Servings", value: "\(servings)")
                                .font(.subheadline)
                        }
                    }
                }

                if let ogImageDataURI,
                   let dataRange = ogImageDataURI.range(of: "base64,"),
                   let imageData = Data(base64Encoded: String(ogImageDataURI[dataRange.upperBound...])),
                   let uiImage = UIImage(data: imageData) {
                    Section("Image") {
                        Image(uiImage: uiImage)
                            .resizable()
                            .scaledToFit()
                            .frame(maxHeight: 200)
                            .cornerRadius(8)
                    }
                }

                Section("Source") {
                    Text(sharedURL.absoluteString)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }
            }
            .navigationTitle("Add Recipe")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { onDismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await save() }
                    }
                    .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty || isSaving || isLoadingOG)
                    .overlay {
                        if isSaving { ProgressView().scaleEffect(0.7) }
                    }
                }
            }
            .task {
                let meta = await OpenGraphFetcher.fetch(url: sharedURL)
                if let ogTitle = meta.title, !ogTitle.isEmpty {
                    title = ogTitle
                }
                ingredients = meta.ingredients ?? []
                steps = meta.steps ?? []
                prepTime = meta.prepTime
                cookTime = meta.cookTime
                servings = meta.servings
                if let imageURL = meta.imageURL {
                    ogImageDataURI = await OpenGraphFetcher.fetchImageAsDataURI(from: imageURL)
                }
                isLoadingOG = false
            }
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        errorMessage = nil

        let ingredientGroups = ingredients.isEmpty ? [] : [ShareIngredientGroup(name: "", items: ingredients)]
        let request = ShareRecipeRequest(
            title: title.trimmingCharacters(in: .whitespaces),
            steps: steps,
            ingredients: ingredientGroups,
            mealType: selectedMealType?.rawValue,
            servings: servings,
            prepTime: prepTime,
            cookTime: cookTime,
            sourceURL: sharedURL.absoluteString,
            imageData: ogImageDataURI
        )

        guard let url = URL(string: baseURL.hasSuffix("/") ? baseURL + "api/recipes" : baseURL + "/api/recipes"),
              let body = try? JSONEncoder().encode(request) else {
            errorMessage = "Invalid configuration."
            return
        }

        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = "POST"
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.setValue("Bearer \(apiToken)", forHTTPHeaderField: "Authorization")
        urlRequest.httpBody = body
        urlRequest.timeoutInterval = 15

        do {
            let (_, response) = try await URLSession.shared.data(for: urlRequest)
            guard let http = response as? HTTPURLResponse else {
                errorMessage = "Unexpected response."
                return
            }
            switch http.statusCode {
            case 200...299:
                onDismiss()
            case 401:
                errorMessage = "Not signed in. Please open Family Hub and sign in first."
            default:
                errorMessage = "Server error (\(http.statusCode)). Please try again."
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
