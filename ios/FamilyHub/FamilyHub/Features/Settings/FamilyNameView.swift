import SwiftUI

struct FamilyNameView: View {
    let apiClient: any APIClientProtocol

    @State private var familyName = ""
    @State private var isLoading = false
    @State private var isSaving = false
    @State private var actionError: APIError?
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
        .errorAlert($actionError)
        .task { await load() }
    }

    private func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let settings = try await apiClient.fetchSettings()
            familyName = settings.familyName
        } catch {
            actionError = .from(error)
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            try await apiClient.updateFamilyName(familyName.trimmingCharacters(in: .whitespaces))
            dismiss()
        } catch {
            actionError = .from(error)
        }
    }
}
