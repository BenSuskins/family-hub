import SwiftUI
import UIKit

private enum ParseStatus {
    case success, partial, failed

    var icon: String {
        switch self {
        case .success: "checkmark.circle.fill"
        case .partial: "exclamationmark.circle.fill"
        case .failed:  "xmark.circle.fill"
        }
    }

    var message: String {
        switch self {
        case .success: "Recipe details extracted"
        case .partial: "Only the title and image are available."
        case .failed:  "The import fails. Enter a title or retry the import."
        }
    }
}

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

private struct ExtractedRecipeResponse: Decodable {
    var title: String?
    var ingredients: [String]?
    var steps: [String]?
    var prepTime: String?
    var cookTime: String?
    var servings: Int?
    var imageURL: String?
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
    @State private var parseStatus: ParseStatus?

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

                if let status = parseStatus {
                    Section("Import") {
                        Label(status.message, systemImage: status.icon)
                        if status != .success {
                            Button("Retry Import") {
                                Task {
                                    isLoadingOG = true
                                    await extractRecipeFromServer()
                                    isLoadingOG = false
                                }
                            }
                            .disabled(isLoadingOG)
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
                        .fixedSize(horizontal: false, vertical: true)
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }
            }
            .disabled(isSaving)
            .interactiveDismissDisabled(isSaving)
            .navigationTitle("Add Recipe")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { onDismiss() }.disabled(isSaving)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await save() }
                    }
                    .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty || isSaving || isLoadingOG)
                }
                ToolbarItem(placement: .status) {
                    if isSaving { ProgressView("Saving recipe") }
                }
            }
            .task {
                await extractRecipeFromServer()
                isLoadingOG = false
            }
        }
    }

    private func extractRecipeFromServer() async {
        let normalizedBase = baseURL.hasSuffix("/") ? baseURL : baseURL + "/"
        guard let extractURL = URL(string: normalizedBase + "api/recipes/extract") else { return }

        var request = URLRequest(url: extractURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("Bearer \(apiToken)", forHTTPHeaderField: "Authorization")
        request.httpBody = try? JSONEncoder().encode(["url": sharedURL.absoluteString])
        request.timeoutInterval = 15

        guard let (data, response) = try? await URLSession.shared.data(for: request),
              let http = response as? HTTPURLResponse,
              (200...299).contains(http.statusCode),
              let result = try? JSONDecoder().decode(ExtractedRecipeResponse.self, from: data)
        else {
            setImportStatus(.failed)
            return
        }

        if title.isEmpty, let extractedTitle = result.title, !extractedTitle.isEmpty {
            title = extractedTitle
        }
        ingredients = result.ingredients ?? []
        steps = result.steps ?? []
        prepTime = result.prepTime
        cookTime = result.cookTime
        servings = result.servings
        if let imageURL = result.imageURL, !imageURL.isEmpty {
            ogImageDataURI = await OpenGraphFetcher.fetchImageAsDataURI(from: imageURL, referer: sharedURL)
        }

        // Social posts rarely give up more than a caption and a thumbnail, so
        // a title and an image on their own still count as a useful import.
        if !ingredients.isEmpty || !steps.isEmpty {
            setImportStatus(.success)
        } else if !title.isEmpty || ogImageDataURI != nil {
            setImportStatus(.partial)
        } else {
            setImportStatus(.failed)
        }
    }

    private func setImportStatus(_ status: ParseStatus) {
        parseStatus = status
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
            let (data, response) = try await URLSession.shared.data(for: urlRequest)
            guard let http = response as? HTTPURLResponse else {
                errorMessage = "We received an unexpected response. Please try again."
                return
            }
            if let message = ShareAPIError.message(forStatus: http.statusCode, body: data) {
                errorMessage = message
            } else {
                onDismiss()
            }
        } catch {
            errorMessage = ShareAPIError.message(for: error)
        }
    }
}
