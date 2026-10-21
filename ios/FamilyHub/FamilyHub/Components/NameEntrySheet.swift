import SwiftUI
import FamilyHubKit

struct NameEntrySheet: View {
    let title: String
    @Binding var name: String
    var fieldTitle = "Name"
    var actionTitle = "Save"
    var explanation: String?
    let save: (String) async -> APIError?
    @State private var isSaving = false
    @State private var error: APIError?
    @Environment(\.dismiss) private var dismiss
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    var body: some View {
        NavigationStack {
            Form {
                Section(fieldTitle) {
                    TextField(fieldTitle, text: $name).accessibilityLabel(fieldTitle)
                    if let explanation { Text(explanation).font(.footnote).foregroundStyle(AppTheme.secondary) }
                }.listRowBackground(AppTheme.surface)
            }
            .disabled(isSaving)
            .appListStyle()
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else {
                        Button(dynamicTypeSize.isAccessibilitySize && actionTitle == "Create token" ? "Create" : actionTitle) {
                            Task {
                                isSaving = true
                                error = await save(name.trimmingCharacters(in: .whitespacesAndNewlines))
                                isSaving = false
                                if error == nil { name = ""; dismiss() }
                            }
                        }.buttonStyle(AppCommitButtonStyle())
                        .accessibilityLabel(actionTitle)
                        .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .errorAlert($error)
        }
    }
}
