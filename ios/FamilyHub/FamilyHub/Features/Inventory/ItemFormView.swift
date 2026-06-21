import SwiftUI

/// Add / edit an item. Count items get Count + Low-at steppers and a unit chip
/// picker; level items get a fill-percentage slider and a low-below-% threshold.
struct ItemFormView: View {
    enum FormMode {
        case create(areaID: String)
        case edit(InventoryItem)
    }

    /// Sensible low-at defaults per mode: a couple of units, or a fifth of a tank.
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

    @FocusState private var nameFocused: Bool

    private let unitColumns = [GridItem(.adaptive(minimum: 64), spacing: 8)]

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }

    private var trimmedName: String { name.trimmingCharacters(in: .whitespaces) }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    nameField
                    modePicker
                    if trackingMode == .count {
                        stockCard
                        unitPicker
                    } else {
                        levelCard
                    }
                    if isEditing { deleteButton }
                    Spacer(minLength: 40)
                }
            }
            .meshBackground()
            .navigationTitle(isEditing ? "Edit Item" : "New Item")
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
        TextField("e.g. Washing tablets", text: $name)
            .font(.system(size: 17))
            .focused($nameFocused)
            .submitLabel(.done)
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
            .glassCard(radius: 14)
            .padding(.horizontal, 16)
            .padding(.top, 12)
    }

    private var modePicker: some View {
        Picker("Tracking", selection: $trackingMode) {
            Text("Count").tag(TrackingMode.count)
            Text("Level").tag(TrackingMode.level)
        }
        .pickerStyle(.segmented)
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .onChange(of: trackingMode) { _, newMode in
            // Switching unit (count vs %) changes what "low at" means, so reset it
            // to a sensible default — but only while creating, never on an edit.
            guard !isEditing else { return }
            lowAt = newMode == .level ? Self.levelLowAtDefault : Self.countLowAtDefault
        }
    }

    private var stockCard: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Stock")
            VStack(spacing: 0) {
                numberRow(label: "Count", value: $quantity)
                Divider().padding(.leading, 16)
                numberRow(label: "Low at", value: $lowAt)
            }
            .glassCard(radius: 14)
            .padding(.horizontal, 16)
        }
    }

    private var levelCard: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Fill level")
            VStack(spacing: 0) {
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text("Remaining")
                            .font(.system(size: 16))
                        Spacer()
                        Text("\(level)%")
                            .font(.system(size: 16, weight: .semibold))
                            .monospacedDigit()
                    }
                    Slider(value: levelBinding, in: 0...100, step: 5)
                        .tint(Color.appGreen)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 11)
                Divider().padding(.leading, 16)
                numberRow(label: "Low below (%)", value: $lowAt)
            }
            .glassCard(radius: 14)
            .padding(.horizontal, 16)
        }
    }

    private var levelBinding: Binding<Double> {
        Binding(get: { Double(level) }, set: { level = Int($0) })
    }

    private func numberRow(label: String, value: Binding<Int>) -> some View {
        HStack {
            Text(label)
                .font(.system(size: 16))
            Spacer()
            QuantityStepper(
                quantity: value.wrappedValue,
                onDecrement: { value.wrappedValue = max(0, value.wrappedValue - 1) },
                onIncrement: { value.wrappedValue += 1 }
            )
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 11)
    }

    private var unitPicker: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Unit")
            LazyVGrid(columns: unitColumns, alignment: .leading, spacing: 8) {
                ForEach(unitOptions, id: \.self) { option in
                    Button { unit = option } label: {
                        Text(option)
                            .font(.system(size: 14, weight: .medium))
                            .foregroundStyle(unit == option ? Color.white : Color.primary)
                            .padding(.horizontal, 13)
                            .padding(.vertical, 7)
                            .frame(maxWidth: .infinity)
                            .background(
                                Capsule().fill(unit == option ? AnyShapeStyle(Color.accentColor) : AnyShapeStyle(Color(.tertiarySystemFill)))
                            )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 16)
        }
    }

    /// The standard unit set, plus the item's existing unit if it's a custom one.
    private var unitOptions: [String] {
        var options = InventoryStyle.units
        if !unit.isEmpty && !options.contains(unit) {
            options.insert(unit, at: 0)
        }
        return options
    }

    private var deleteButton: some View {
        Button(role: .destructive) {
            Task { await delete() }
        } label: {
            Text("Delete Item")
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .glassCard(radius: 14)
        }
        .padding(.horizontal, 16)
        .padding(.top, 24)
        .disabled(isSaving)
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
