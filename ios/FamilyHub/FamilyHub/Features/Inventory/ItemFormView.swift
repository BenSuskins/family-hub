import SwiftUI
import FamilyHubKit

struct ItemFormView: View {
    enum FormMode { case create(areaID: String), edit(InventoryItem) }
    private static let countLowAtDefault = 2
    private static let levelLowAtDefault = 20
    let mode: FormMode
    @Bindable var viewModel: InventoryViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var trackingMode: TrackingMode = .count
    @State private var quantity = 1
    @State private var level = 100
    @State private var lowAt = ItemFormView.countLowAtDefault
    @State private var unit = "pcs"
    @State private var isSaving = false
    @State private var confirmDelete = false
    @FocusState private var nameFocused: Bool
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }
    private var validStock: Bool { (0...1_000_000).contains(quantity) && (0...100).contains(level) && (0...(trackingMode == .level ? 100 : 1_000_000)).contains(lowAt) }
    private var trimmedName: String { name.trimmingCharacters(in: .whitespacesAndNewlines) }
    private var unitOptions: [String] {
        InventoryStyle.units.contains(unit) || unit.isEmpty ? InventoryStyle.units : [unit] + InventoryStyle.units
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Item name") {
                    TextField("Name", text: $name, axis: .vertical).lineLimit(1...3).focused($nameFocused)
                        .submitLabel(.done).onSubmit { nameFocused = false }
                }.listRowBackground(AppTheme.surface)
                Section {
                    VStack(alignment: .leading, spacing: 12) {
                        if dynamicTypeSize.isAccessibilitySize {
                            trackingPicker.labelsHidden()
                        } else {
                            trackingPicker.pickerStyle(.segmented)
                        }
                        Text(trackingMode == .count ? "Use a count for separate items such as pods or rolls." : "Use a percentage for a partly used bottle.")
                            .font(.footnote).foregroundStyle(AppTheme.secondary)
                    }
                } header: { Text("Track stock by") }
                .listRowBackground(Color.clear)
                .onChange(of: trackingMode) { _, newMode in
                    guard !isEditing else { return }
                    lowAt = newMode == .level ? Self.levelLowAtDefault : Self.countLowAtDefault
                }
                Section {
                    if trackingMode == .count {
                        LabeledContent("Quantity") {
                            QuantityStepper(quantity: quantity, editableQuantity: $quantity, label: name,
                                            onDecrement: { quantity = max(0, quantity - 1) },
                                            onIncrement: { quantity = min(1_000_000, quantity + 1) })
                        }
                        LabeledContent("Alert at or below") {
                            TextField("Alert threshold", value: $lowAt, format: .number).keyboardType(.numberPad).multilineTextAlignment(.trailing)
                        }
                        Picker("Unit", selection: $unit) {
                            Text("None").tag("")
                            ForEach(unitOptions, id: \.self) { Text($0).tag($0) }
                        }
                    } else {
                        VStack(alignment: .leading) {
                            LabeledContent("Remaining level (%)") {
                                TextField("Remaining level", value: $level, format: .number).keyboardType(.numberPad).multilineTextAlignment(.trailing)
                            }
                        }
                        LabeledContent {
                            TextField("Alert threshold", value: $lowAt, format: .number).keyboardType(.numberPad).multilineTextAlignment(.trailing)
                        } label: {
                            Text("Low at (%)")
                        }
                    }
                } footer: {
                    Text("Items at or below this threshold appear in Running low.")
                }.listRowBackground(AppTheme.surface)
                if isEditing {
                    Section { Button("Delete Item", role: .destructive) { confirmDelete = true }.foregroundStyle(AppTheme.destructive) }.listRowBackground(AppTheme.surface)
                }
            }
            .disabled(isSaving)
            .appListStyle()
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle(isEditing ? "Edit Item" : "New Item")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else { Button("Save") { Task { await save() } }.buttonStyle(AppCommitButtonStyle()).disabled(trimmedName.isEmpty || !validStock) }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .confirmationDialog("Delete \(name)?", isPresented: $confirmDelete, titleVisibility: .visible) {
                Button("Delete Item", role: .destructive) { Task { await delete() } }
            }
            .onAppear(perform: populate)
            .errorAlert($viewModel.actionError)
        }
    }

    private var trackingPicker: some View {
        Picker("Track stock by", selection: $trackingMode) {
            Text("Count").tag(TrackingMode.count)
            Text("Level").tag(TrackingMode.level)
        }
    }

    private func populate() {
        if case .edit(let item) = mode {
            name = item.name
            trackingMode = item.trackingMode
            quantity = item.quantity
            level = item.level
            lowAt = item.lowAt
            unit = item.unit
        } else {
            nameFocused = true
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        let request = ItemRequest(name: trimmedName, trackingMode: trackingMode,
                                  quantity: quantity, level: level, unit: unit, lowAt: lowAt)
        let result: InventoryItem?
        switch mode {
        case .create(let areaID):
            result = await viewModel.createItem(areaID: areaID, request)
        case .edit(let item):
            result = await viewModel.updateItem(areaID: item.areaID, id: item.id, request)
        }
        if result != nil { dismiss() }
    }

    private func delete() async {
        guard case .edit(let item) = mode else { return }
        isSaving = true
        defer { isSaving = false }
        if await viewModel.deleteItem(areaID: item.areaID, id: item.id) {
            dismiss()
        }
    }
}
