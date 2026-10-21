import SwiftUI
import FamilyHubKit

struct AreaFormView: View {
    enum FormMode { case create, edit(InventoryArea) }
    let mode: FormMode
    @Bindable var viewModel: InventoryViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var icon = "box"
    @State private var tint = "blue"
    @State private var isSaving = false
    @FocusState private var nameFocused: Bool

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }
    private var trimmedName: String { name.trimmingCharacters(in: .whitespacesAndNewlines) }

    var body: some View {
        NavigationStack {
            Form {
                Section("Area") {
                    TextField("Name", text: $name).focused($nameFocused)
                    Picker("Icon", selection: $icon) {
                        ForEach(InventoryStyle.icons, id: \.self) { option in
                            Label(option.capitalized, systemImage: InventoryStyle.symbol(option)).tag(option)
                        }
                    }
                    Picker("Colour", selection: $tint) {
                        ForEach(InventoryStyle.tints, id: \.self) { option in
                            Text(option.capitalized).tag(option)
                        }
                    }
                }.listRowBackground(AppTheme.surface)
                Section("Preview") {
                    Label {
                        Text(trimmedName.isEmpty ? "New Area" : trimmedName)
                    } icon: {
                        Image(systemName: InventoryStyle.symbol(icon)).foregroundStyle(InventoryStyle.color(tint))
                    }
                }.listRowBackground(AppTheme.surface)
            }
            .disabled(isSaving)
            .appListStyle()
            .navigationTitle(isEditing ? "Edit Area" : "New Area")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else { Button("Save") { Task { await save() } }.buttonStyle(AppCommitButtonStyle()).disabled(trimmedName.isEmpty) }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .onAppear(perform: populate)
            .errorAlert($viewModel.actionError)
        }
    }

    private func populate() {
        if case .edit(let area) = mode {
            name = area.name
            icon = area.icon
            tint = area.tint
        } else {
            nameFocused = true
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        let request = AreaRequest(name: trimmedName, icon: icon, tint: tint)
        let result: InventoryArea?
        switch mode {
        case .create:
            result = await viewModel.createArea(request)
        case .edit(let area):
            result = await viewModel.updateArea(id: area.id, request)
        }
        if result != nil { dismiss() }
    }
}
