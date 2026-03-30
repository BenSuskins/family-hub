import SwiftUI
import PhotosUI
import UIKit

struct RecipeFormView: View {
    enum FormMode {
        case create
        case edit(Recipe)
    }

    let mode: FormMode
    let viewModel: RecipesViewModel
    let apiClient: any APIClientProtocol
    var onSave: ((Recipe) -> Void)?

    @Environment(\.dismiss) private var dismiss

    @State private var title = ""
    @State private var mealType = ""
    @State private var servings = ""
    @State private var prepTime = ""
    @State private var cookTime = ""
    @State private var sourceURL = ""
    @State private var steps: [StepDraft] = [StepDraft()]
    @State private var groups: [IngredientGroupDraft] = [IngredientGroupDraft()]

    @State private var photoPickerItem: PhotosPickerItem?
    @State private var selectedImageData: Data?
    @State private var existingHasImage = false
    @State private var clearImage = false

    @State private var isSaving = false
    @State private var validationError: String?

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }

    private var editingRecipe: Recipe? {
        if case .edit(let r) = mode { return r }
        return nil
    }

    var body: some View {
        NavigationStack {
            Form {
                basicInfoSection
                timingSection
                sourceSection
                ingredientGroupsSection
                stepsSection
                imageSection
            }
            .navigationTitle(isEditing ? "Edit Recipe" : "New Recipe")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await save() }
                    }
                    .disabled(isSaving || title.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .alert("Error", isPresented: Binding(
                get: { validationError != nil },
                set: { if !$0 { validationError = nil } }
            )) {
                Button("OK", role: .cancel) {}
            } message: {
                Text(validationError ?? "")
            }
            .onAppear { populateFromRecipe() }
            .onChange(of: photoPickerItem) { _, item in
                Task {
                    guard let item else { return }
                    selectedImageData = try? await item.loadTransferable(type: Data.self)
                    clearImage = false
                }
            }
        }
    }

    // MARK: - Sections

    private var basicInfoSection: some View {
        Section("Recipe Info") {
            TextField("Title (required)", text: $title)

            Picker("Meal Type", selection: $mealType) {
                Text("None").tag("")
                ForEach(RecipesViewModel.mealTypeOptions, id: \.self) { type in
                    Text(type.capitalized).tag(type)
                }
            }

            HStack {
                Text("Servings")
                Spacer()
                TextField("e.g. 4", text: $servings)
                    .keyboardType(.numberPad)
                    .multilineTextAlignment(.trailing)
                    .frame(width: 80)
            }
        }
    }

    private var timingSection: some View {
        Section("Timing") {
            HStack {
                Text("Prep Time")
                Spacer()
                TextField("e.g. 15 min", text: $prepTime)
                    .multilineTextAlignment(.trailing)
            }
            HStack {
                Text("Cook Time")
                Spacer()
                TextField("e.g. 30 min", text: $cookTime)
                    .multilineTextAlignment(.trailing)
            }
        }
    }

    private var sourceSection: some View {
        Section("Source") {
            TextField("Source URL", text: $sourceURL)
                .keyboardType(.URL)
                .autocorrectionDisabled()
                .textInputAutocapitalization(.never)
        }
    }

    private var ingredientGroupsSection: some View {
        Section {
            ForEach($groups) { $group in
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        TextField("Group name (e.g. Main, Sauce)", text: $group.name)
                            .font(.subheadline.weight(.semibold))
                        if groups.count > 1 {
                            Button(role: .destructive) {
                                groups.removeAll { $0.id == group.id }
                            } label: {
                                Image(systemName: "minus.circle.fill")
                                    .foregroundStyle(.red)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    TextEditor(text: $group.items)
                        .frame(minHeight: 80)
                        .font(.subheadline)
                        .overlay(alignment: .topLeading) {
                            if group.items.isEmpty {
                                Text("One ingredient per line")
                                    .font(.subheadline)
                                    .foregroundStyle(.tertiary)
                                    .padding(.top, 8)
                                    .padding(.leading, 4)
                                    .allowsHitTesting(false)
                            }
                        }
                }
                .padding(.vertical, 4)
            }
            Button {
                groups.append(IngredientGroupDraft())
            } label: {
                Label("Add Ingredient Group", systemImage: "plus.circle")
            }
        } header: {
            Text("Ingredients")
        }
    }

    private var stepsSection: some View {
        Section {
            ForEach(Array(steps.enumerated()), id: \.element.id) { index, step in
                HStack(alignment: .top, spacing: 10) {
                    Text("\(index + 1)")
                        .font(.subheadline.bold())
                        .foregroundStyle(Color.accentColor)
                        .frame(width: 22, alignment: .trailing)
                        .padding(.top, 8)
                    TextField("Describe this step", text: $steps[index].text, axis: .vertical)
                        .lineLimit(2...6)
                    if steps.count > 1 {
                        Button(role: .destructive) {
                            steps.removeAll { $0.id == step.id }
                        } label: {
                            Image(systemName: "minus.circle.fill")
                                .foregroundStyle(.red)
                        }
                        .buttonStyle(.plain)
                        .padding(.top, 8)
                    }
                }
            }
            Button {
                steps.append(StepDraft())
            } label: {
                Label("Add Step", systemImage: "plus.circle")
            }
        } header: {
            Text("Steps")
        }
    }

    private var imageSection: some View {
        Section("Image") {
            if let data = selectedImageData, let uiImage = UIImage(data: data) {
                VStack(alignment: .leading, spacing: 8) {
                    Image(uiImage: uiImage)
                        .resizable()
                        .aspectRatio(4/3, contentMode: .fill)
                        .frame(maxHeight: 200)
                        .clipped()
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    Button(role: .destructive) {
                        selectedImageData = nil
                        photoPickerItem = nil
                        clearImage = true
                    } label: {
                        Label("Remove Image", systemImage: "trash")
                    }
                }
            } else if existingHasImage && !clearImage {
                HStack {
                    Label("Current image set", systemImage: "photo")
                        .foregroundStyle(.secondary)
                    Spacer()
                    Button(role: .destructive) {
                        clearImage = true
                    } label: {
                        Text("Remove")
                            .foregroundStyle(.red)
                    }
                }
            }

            PhotosPicker(selection: $photoPickerItem, matching: .images) {
                Label(
                    (selectedImageData != nil || (existingHasImage && !clearImage)) ? "Change Photo" : "Choose Photo",
                    systemImage: "photo.badge.plus"
                )
            }
        }
    }

    // MARK: - Actions

    private func populateFromRecipe() {
        guard let r = editingRecipe else { return }
        title = r.title
        mealType = r.mealType ?? ""
        servings = r.servings.map(String.init) ?? ""
        prepTime = r.prepTime ?? ""
        cookTime = r.cookTime ?? ""
        sourceURL = r.sourceURL ?? ""
        existingHasImage = r.hasImage

        if let ingredientGroups = r.ingredients, !ingredientGroups.isEmpty {
            groups = ingredientGroups.map {
                IngredientGroupDraft(name: $0.name == "Main" ? "" : $0.name, items: $0.items.joined(separator: "\n"))
            }
        }

        if let recipeSteps = r.steps, !recipeSteps.isEmpty {
            steps = recipeSteps.map { StepDraft(text: $0) }
        }
    }

    private func save() async {
        let trimmedTitle = title.trimmingCharacters(in: .whitespaces)
        guard !trimmedTitle.isEmpty else {
            validationError = "Title is required."
            return
        }

        isSaving = true
        defer { isSaving = false }

        let imageDataValue = buildImageDataString()

        let request = RecipeRequest(
            title: trimmedTitle,
            steps: steps
                .map { $0.text.trimmingCharacters(in: .whitespaces) }
                .filter { !$0.isEmpty },
            ingredients: groups.compactMap { g in
                let items = g.items
                    .components(separatedBy: "\n")
                    .map { $0.trimmingCharacters(in: .whitespaces) }
                    .filter { !$0.isEmpty }
                guard !items.isEmpty else { return nil }
                let groupName = g.name.trimmingCharacters(in: .whitespaces)
                return IngredientGroup(name: groupName.isEmpty ? "Main" : groupName, items: items)
            },
            mealType: mealType.isEmpty ? nil : mealType,
            servings: Int(servings),
            prepTime: prepTime.isEmpty ? nil : prepTime.trimmingCharacters(in: .whitespaces),
            cookTime: cookTime.isEmpty ? nil : cookTime.trimmingCharacters(in: .whitespaces),
            sourceURL: sourceURL.isEmpty ? nil : sourceURL.trimmingCharacters(in: .whitespaces),
            imageData: imageDataValue
        )

        let result: Recipe?
        if let existing = editingRecipe {
            result = await viewModel.updateRecipe(id: existing.id, request)
        } else {
            result = await viewModel.createRecipe(request)
        }

        if let saved = result {
            onSave?(saved)
            dismiss()
        } else {
            validationError = viewModel.errorMessage ?? "Failed to save recipe."
        }
    }

    private func buildImageDataString() -> String? {
        if let data = selectedImageData {
            let mimeType = data.jpegMimeType
            let base64 = data.base64EncodedString()
            return "data:\(mimeType);base64,\(base64)"
        }
        if clearImage {
            return ""
        }
        return nil
    }
}

// MARK: - Draft models

private struct IngredientGroupDraft: Identifiable {
    let id = UUID()
    var name: String = ""
    var items: String = ""
}

private struct StepDraft: Identifiable {
    let id = UUID()
    var text: String = ""
}

// MARK: - Data helper

private extension Data {
    var jpegMimeType: String {
        // Detect by magic bytes
        var byte: UInt8 = 0
        copyBytes(to: &byte, count: 1)
        switch byte {
        case 0xFF: return "image/jpeg"
        case 0x89: return "image/png"
        case 0x47: return "image/gif"
        case 0x52: return "image/webp"
        default: return "image/jpeg"
        }
    }
}
