import SwiftUI

/// Add / edit an area: name, icon grid, colour swatches, with a live preview tile.
struct AreaFormView: View {
    enum FormMode {
        case create
        case edit(InventoryArea)
    }

    let mode: FormMode
    @Bindable var viewModel: InventoryViewModel
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var icon = "box"
    @State private var tint = "blue"
    @State private var isSaving = false

    @FocusState private var nameFocused: Bool

    private let iconColumns = Array(repeating: GridItem(.flexible()), count: 4)

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }

    private var trimmedName: String { name.trimmingCharacters(in: .whitespaces) }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 0) {
                    AreaIconTile(icon: icon, tint: tint, size: 72)
                        .padding(.top, 8)
                        .padding(.bottom, 18)

                    nameField
                    iconPicker
                    colorPicker

                    Spacer(minLength: 40)
                }
            }
            .meshBackground()
            .navigationTitle(isEditing ? "Edit Area" : "New Area")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { Task { await save() } }
                        .fontWeight(.semibold)
                        .disabled(trimmedName.isEmpty || isSaving)
                }
            }
            .onAppear(perform: populate)
            .errorAlert($viewModel.actionError)
        }
    }

    private var nameField: some View {
        TextField("e.g. Laundry cupboard", text: $name)
            .font(.system(size: 17))
            .focused($nameFocused)
            .submitLabel(.done)
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
            .glassCard(radius: 14)
            .padding(.horizontal, 16)
    }

    private var iconPicker: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Icon")
            LazyVGrid(columns: iconColumns, spacing: 12) {
                ForEach(InventoryStyle.icons, id: \.self) { option in
                    Button { icon = option } label: {
                        RoundedRectangle(cornerRadius: 13, style: .continuous)
                            .fill(icon == option ? InventoryStyle.color(tint) : Color(.tertiarySystemFill))
                            .frame(height: 48)
                            .overlay {
                                Image(systemName: InventoryStyle.symbol(option))
                                    .font(.system(size: 22, weight: .semibold))
                                    .foregroundStyle(icon == option ? Color.white : Color.secondary)
                            }
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(14)
            .glassCard(radius: 14)
            .padding(.horizontal, 16)
        }
    }

    private var colorPicker: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Color")
            HStack(spacing: 14) {
                ForEach(InventoryStyle.tints, id: \.self) { option in
                    Button { tint = option } label: {
                        Circle()
                            .fill(InventoryStyle.color(option))
                            .frame(width: 34, height: 34)
                            .overlay {
                                if tint == option {
                                    Circle().stroke(InventoryStyle.color(option), lineWidth: 2)
                                        .padding(-4)
                                }
                            }
                    }
                    .buttonStyle(.plain)
                }
                Spacer(minLength: 0)
            }
            .padding(14)
            .glassCard(radius: 14)
            .padding(.horizontal, 16)
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
