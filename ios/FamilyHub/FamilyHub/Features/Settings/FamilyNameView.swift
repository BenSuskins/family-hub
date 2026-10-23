import SwiftUI
import FamilyHubKit

struct FamilyNameView: View {
    let apiClient: any APIClientProtocol

    @State private var familyName = ""
    @State private var isLoading = true
    @State private var hasLoaded = false
    @State private var isSaving = false
    @State private var actionError: APIError?
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        Form {
            Section {
                TextField("Family Name", text: $familyName)
                    .textInputAutocapitalization(.words)
                    .autocorrectionDisabled()
            } header: { Text("Family name") } footer: {
                Text("Displayed in the app header for all family members.")
            }.listRowBackground(AppTheme.surface)
            Section {
                Button("Save family name") { Task { await save() } }
                    .buttonStyle(AppPrimaryButtonStyle())
                    .accessibilityIdentifier("Save")
                    .disabled(familyName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || isSaving || !hasLoaded)
            }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
        }
        .appListStyle()
        .navigationTitle("Family name")
        .navigationBarTitleDisplayMode(.large)
        .overlay {
            if isLoading {
                ProgressView()
            } else if !hasLoaded {
                ContentUnavailableView {
                    Label("Family Name Unavailable", systemImage: "house")
                } description: {
                    Text("The family name could not be loaded.")
                } actions: {
                    Button("Retry") { Task { await load() } }
                }
            }
        }
        .errorAlert($actionError)
        .task { await load() }
    }

    private func load() async {
        isLoading = true
        actionError = nil
        defer { isLoading = false }
        do {
            let settings = try await apiClient.fetchSettings()
            familyName = settings.familyName
            hasLoaded = true
        } catch {
            actionError = .from(error)
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            try await apiClient.updateFamilyName(familyName.trimmingCharacters(in: .whitespacesAndNewlines))
            dismiss()
        } catch {
            actionError = .from(error)
        }
    }
}
