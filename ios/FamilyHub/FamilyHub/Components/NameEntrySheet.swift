import SwiftUI
import FamilyHubKit

struct NameEntrySheet: View {
    let title: String
    @Binding var name: String
    let save: (String) async -> APIError?
    @State private var isSaving = false
    @State private var error: APIError?
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Form {
                Section("Name") { TextField("Name", text: $name) }
            }
            .disabled(isSaving)
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else {
                        Button("Save") {
                            Task {
                                isSaving = true
                                error = await save(name.trimmingCharacters(in: .whitespacesAndNewlines))
                                isSaving = false
                                if error == nil { name = ""; dismiss() }
                            }
                        }
                        .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .errorAlert($error)
        }
    }
}
