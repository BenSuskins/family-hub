import SwiftUI

// Duplicated here since the extension cannot import the main app module
private struct ShareRecipeRequest: Encodable {
    var title: String
    var steps: [String]
    var ingredients: [ShareIngredientGroup]
    var sourceURL: String?
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
                isLoadingOG = false
            }
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        errorMessage = nil

        let request = ShareRecipeRequest(
            title: title.trimmingCharacters(in: .whitespaces),
            steps: [],
            ingredients: [],
            sourceURL: sharedURL.absoluteString
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
