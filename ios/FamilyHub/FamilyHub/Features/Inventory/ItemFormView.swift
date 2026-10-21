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
                Section("Item") {
                    TextField("Name", text: $name).focused($nameFocused)
                    Picker("Tracking", selection: $trackingMode) {
                        Text("Count").tag(TrackingMode.count)
                        Text("Level").tag(TrackingMode.level)
                    }
                    Text(trackingMode == .count ? "Count tracks individual units, such as bottles or packs." : "Level tracks the remaining percentage, such as a bottle of cleaner.")
                        .font(.footnote).foregroundStyle(AppTheme.secondary)
                    .onChange(of: trackingMode) { _, newMode in
                        guard !isEditing else { return }
                        lowAt = newMode == .level ? Self.levelLowAtDefault : Self.countLowAtDefault
                    }
                }.listRowBackground(AppTheme.surface)
                Section("Stock") {
                    if trackingMode == .count {
                        LabeledContent("Count") {
                            TextField("Count", value: $quantity, format: .number).keyboardType(.numberPad).multilineTextAlignment(.trailing)
                        }
                        Stepper("Count: \(quantity)", value: $quantity, in: 0...1_000_000)
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
                            Slider(value: Binding(get: { Double(level) }, set: { level = Int($0) }), in: 0...100, step: 5) {
                                Text("Remaining level")
                            }
                            .accessibilityValue("\(level) percent")
                        }
                        LabeledContent("Alert at or below (%)") {
                            TextField("Alert threshold", value: $lowAt, format: .number).keyboardType(.numberPad).multilineTextAlignment(.trailing)
                        }
                    }
                }.listRowBackground(AppTheme.surface)
                if isEditing {
                    Section { Button("Delete Item", role: .destructive) { confirmDelete = true } }.listRowBackground(AppTheme.surface)
                }
            }
            .disabled(isSaving)
            .appListStyle()
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
