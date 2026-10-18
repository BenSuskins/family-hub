import SwiftUI
import FamilyHubKit

struct ShoppingListView: View {
    let apiClient: any APIClientProtocol
    @State private var model: ShoppingViewModel
    @State private var newName = ""
    @State private var activeSheet: ShoppingSheet?
    @State private var confirmClear = false
    @State private var deletingItem: ShoppingItem?
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    @Environment(\.scenePhase) private var scenePhase
    @State private var isVisible = false

    private enum ShoppingSheet: Identifiable {
        case rename(ShoppingItem)
        case restock(ShoppingItem)
        var id: String {
            switch self {
            case .rename(let item): return "rename-" + item.id
            case .restock(let item): return "restock-" + item.id
            }
        }
    }

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _model = State(wrappedValue: ShoppingViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            if !model.canEdit {
                Section {
                    if let date = model.capturedAt {
                        Label("Read-only copy", systemImage: "wifi.slash")
                        Text("Last updated \(date.formatted(date: .abbreviated, time: .shortened))")
                            .font(.caption).foregroundStyle(.secondary)
                    } else {
                        Text(model.isLoading ? "Loading…" : "Shopping list unavailable.")
                    }
                    Button("Refresh") { Task { await model.load() } }
                }
            }
            Section("Add an item") {
                HStack {
                    TextField("Item name", text: $newName)
                        .onSubmit { add() }
                    Button(action: add) { Image(systemName: "plus") }
                        .frame(minWidth: 44, minHeight: 44)
                        .accessibilityLabel("Add item")
                        .disabled(newName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                .disabled(!model.canEdit || model.isChanging)
            }
            Section("To buy") {
                if model.items.allSatisfy(\.purchased) { Text("The list is empty.").foregroundStyle(.secondary) }
                ForEach(model.items.filter { !$0.purchased }) { row($0) }
            }
            if model.items.contains(where: \.purchased) {
                Section("Purchased") {
                    ForEach(model.items.filter(\.purchased)) { row($0) }
                    Button("Clear purchased items", role: .destructive) { confirmClear = true }
                        .disabled(!model.canEdit || model.isChanging)
                }
            }
        }
        .navigationTitle("Shopping List")
        .refreshable { await model.load() }
        .errorAlert($model.error)
        .confirmationDialog("Clear purchased items?", isPresented: $confirmClear) {
            Button("Clear purchased items", role: .destructive) { Task { await model.clearPurchased() } }
        }
        .confirmationDialog("Delete this shopping item?", isPresented: Binding(
            get: { deletingItem != nil }, set: { if !$0 { deletingItem = nil } }
        ), titleVisibility: .visible) {
            if let item = deletingItem {
                Button("Delete Item", role: .destructive) { Task { await model.remove(item) } }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .rename(let item): ShoppingNameView(item: item, model: model)
            case .restock(let item): ShoppingRestockView(item: item, model: model, apiClient: apiClient)
            }
        }
        .onAppear { isVisible = true }
        .onDisappear { isVisible = false }
        .task(id: scenePhase == .active && isVisible) { [shouldRefresh = scenePhase == .active && isVisible] in
            guard shouldRefresh else { return }
            while !Task.isCancelled {
                await model.load()
                do { try await Task.sleep(for: .seconds(30)) } catch { return }
            }
        }
    }

    private func add() {
        let name = newName
        Task {
            await model.add(name: name)
            if model.error == nil { newName = "" }
        }
    }

    private func row(_ item: ShoppingItem) -> some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
            : AnyLayout(HStackLayout())
        return layout {
            Button {
                Task {
                    if let updated = await model.toggle(item), updated.purchased, updated.inventoryItemID != nil, !updated.restocked {
                        activeSheet = .restock(updated)
                    }
                }
            } label: {
                Image(systemName: item.purchased ? "checkmark.circle.fill" : "circle")
                    .frame(width: 44, height: 44)
            }
            .buttonStyle(.borderless)
            .accessibilityLabel(item.purchased ? "Restore \(item.name)" : "Purchase \(item.name)")
            Text(item.name).strikethrough(item.purchased)
            Spacer()
            if item.purchased, item.inventoryItemID != nil {
                if item.restocked {
                    Text("Restocked").font(.caption).foregroundStyle(.secondary)
                } else {
                    Button("Restock") { activeSheet = .restock(item) }.buttonStyle(.borderless)
                        .frame(minHeight: 44)
                }
            }
            Menu {
                Button("Rename") { activeSheet = .rename(item) }
                Button("Delete", role: .destructive) { deletingItem = item }
            } label: {
                Image(systemName: "ellipsis").frame(width: 44, height: 44)
            }
            .accessibilityLabel("Actions for \(item.name)")
        }
        .disabled(!model.canEdit || model.isChanging)
    }
}

private struct ShoppingNameView: View {
    let item: ShoppingItem
    @Bindable var model: ShoppingViewModel
    @State private var name = ""
    @Environment(\.dismiss) private var dismiss
    var body: some View {
        NavigationStack {
            Form { TextField("Item name", text: $name) }
                .navigationTitle("Rename item")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Save") {
                            Task {
                                await model.rename(item, name: name)
                                if model.error == nil { dismiss() }
                            }
                        }.disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || !model.canEdit || model.isChanging)
                    }
                }
                .errorAlert($model.error)
        }.onAppear { name = item.name }
    }
}

private struct ShoppingRestockView: View {
    let item: ShoppingItem
    @Bindable var model: ShoppingViewModel
    let apiClient: any APIClientProtocol
    @State private var inventory: InventoryItem?
    @State private var amount = ""
    @State private var error: APIError?
    @State private var isLoading = true
    @Environment(\.dismiss) private var dismiss

    private var validAmount: Int? {
        guard let value = Int(amount), let inventory else { return nil }
        if inventory.trackingMode == .level { return (0...100).contains(value) ? value : nil }
        return (1...1_000_000).contains(value) ? value : nil
    }

    var body: some View {
        NavigationStack {
            Form {
                Section(item.name) {
                    if let inventory {
                        Text(inventory.trackingMode == .level ? "Set the new inventory level." : "Add the units you purchased to inventory.")
                        TextField(inventory.trackingMode == .level ? "New level (%)" : "Units to add", text: $amount)
                            .keyboardType(.numberPad)
                        Text("Current stock: \(inventory.statusLabel)").foregroundStyle(.secondary)
                    } else {
                        Text(isLoading ? "Loading inventory…" : "Inventory unavailable. You can restock later.")
                    }
                }
            }
            .navigationTitle("Restock inventory")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Not now") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Restock") {
                        guard let validAmount, let inventory else { return }
                        Task {
                            if await model.restock(item, quantity: inventory.trackingMode == .count ? validAmount : nil, level: inventory.trackingMode == .level ? validAmount : nil) { dismiss() }
                        }
                    }.disabled(validAmount == nil || !model.canEdit || model.isChanging)
                }
            }
            .errorAlert($error)
            .errorAlert($model.error)
            .task {
                defer { isLoading = false }
                do {
                    inventory = try await apiClient.fetchInventory().flatMap(\.items).first { $0.id == item.inventoryItemID }
                } catch { self.error = .from(error) }
            }
        }
    }
}
