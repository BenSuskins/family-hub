import SwiftUI
import FamilyHubKit

struct NameEntrySheet: View {
    let title: String
    @Binding var name: String
    var fieldTitle = "Name"
    var actionTitle = "Save"
    var explanation: String?
    var fieldHelp: String?
    var primaryActionInContent = false
    var deletionName: String?
    var deleteActionTitle = "Delete"
    var delete: (() async -> APIError?)?
    let save: (String) async -> APIError?
    @State private var isSaving = false
    @State private var error: APIError?
    @State private var confirmingDelete = false
    @Environment(\.dismiss) private var dismiss
    @FocusState private var nameIsFocused: Bool

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField(fieldTitle, text: $name).accessibilityLabel(fieldTitle)
                        .focused($nameIsFocused).submitLabel(.done).onSubmit { nameIsFocused = false }
                    if !primaryActionInContent, let explanation { Text(explanation).font(.footnote).foregroundStyle(AppTheme.secondary) }
                } header: { Text(fieldTitle) } footer: {
                    if let fieldHelp { Text(fieldHelp) }
                }.listRowBackground(AppTheme.surface)
                if primaryActionInContent {
                    if let explanation {
                        Section { Text(explanation).font(.subheadline) }.listRowBackground(AppTheme.sage)
                    }
                    Section {
                        commitButton.buttonStyle(AppPrimaryButtonStyle())
                    }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
                }
                if delete != nil {
                    Section {
                        Button(deleteActionTitle, role: .destructive) { confirmingDelete = true }
                            .foregroundStyle(AppTheme.destructive)
                    }.listRowBackground(AppTheme.surface)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .disabled(isSaving)
            .appListStyle()
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                if !primaryActionInContent {
                    ToolbarItem(placement: .confirmationAction) {
                        commitButton.buttonStyle(AppCommitButtonStyle())
                    }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .errorAlert($error)
            .confirmationDialog("Delete \(deletionName ?? name)?", isPresented: $confirmingDelete, titleVisibility: .visible) {
                Button(deleteActionTitle, role: .destructive) {
                    Task {
                        guard let delete else { return }
                        isSaving = true
                        error = await delete()
                        isSaving = false
                        if error == nil { dismiss() }
                    }
                }
            }
        }
    }
    private var commitButton: some View {
        Button {
            Task {
                isSaving = true
                error = await save(name.trimmingCharacters(in: .whitespacesAndNewlines))
                isSaving = false
                if error == nil { name = ""; dismiss() }
            }
        } label: {
            if isSaving { ProgressView() }
            else { Text(actionTitle) }
        }
        .accessibilityLabel(actionTitle)
        .disabled(isSaving || name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
    }

}
