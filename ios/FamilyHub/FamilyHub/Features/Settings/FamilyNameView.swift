import SwiftUI

struct FamilyNameView: View {
    let apiClient: any APIClientProtocol

    @State private var familyName = ""
    @State private var isLoading = false
    @State private var isSaving = false
    @State private var errorMessage: String?
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        Form {
            Section {
                TextField("Family Name", text: $familyName)
                    .autocorrectionDisabled()
            } footer: {
                Text("Displayed in the app header for all family members.")
            }
        }
        .navigationTitle("Family Name")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button("Save") { Task { await save() } }
                    .disabled(familyName.trimmingCharacters(in: .whitespaces).isEmpty || isSaving)
            }
        }
        .overlay {
            if isLoading { ProgressView() }
        }
        .alert("Error", isPresented: Binding(
            get: { errorMessage != nil },
            set: { if !$0 { errorMessage = nil } }
        )) {
            Button("OK") { errorMessage = nil }
        } message: {
            Text(errorMessage ?? "")
        }
        .task { await load() }
    }

    private func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let settings = try await apiClient.fetchSettings()
            familyName = settings.familyName
        } catch {
            errorMessage = "Failed to load settings"
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            try await apiClient.updateFamilyName(familyName.trimmingCharacters(in: .whitespaces))
            dismiss()
        } catch {
            errorMessage = "Failed to save family name"
        }
    }
}
