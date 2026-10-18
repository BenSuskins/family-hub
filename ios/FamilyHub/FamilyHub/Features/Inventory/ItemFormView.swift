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
                    .onChange(of: trackingMode) { _, newMode in
                        guard !isEditing else { return }
                        lowAt = newMode == .level ? Self.levelLowAtDefault : Self.countLowAtDefault
                    }
                }
                Section("Stock") {
                    if trackingMode == .count {
                        Stepper("Count: \(quantity)", value: $quantity, in: 0...1_000_000)
                        Stepper("Low at: \(lowAt)", value: $lowAt, in: 0...1_000_000)
                        Picker("Unit", selection: $unit) {
                            Text("None").tag("")
                            ForEach(unitOptions, id: \.self) { Text($0).tag($0) }
                        }
                    } else {
                        VStack(alignment: .leading) {
                            Text("Remaining: \(level)%")
                            Slider(value: Binding(get: { Double(level) }, set: { level = Int($0) }), in: 0...100, step: 5) {
                                Text("Remaining level")
                            }
                            .accessibilityValue("\(level) percent")
                        }
                        Stepper("Low below: \(lowAt)%", value: $lowAt, in: 0...100)
                    }
                }
                if isEditing {
                    Section { Button("Delete Item", role: .destructive) { confirmDelete = true } }
                }
            }
            .disabled(isSaving)
            .navigationTitle(isEditing ? "Edit Item" : "New Item")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() }.disabled(isSaving) }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving { ProgressView() }
                    else { Button("Save") { Task { await save() } }.disabled(trimmedName.isEmpty) }
                }
            }
            .interactiveDismissDisabled(isSaving)
            .confirmationDialog("Delete this item?", isPresented: $confirmDelete, titleVisibility: .visible) {
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
