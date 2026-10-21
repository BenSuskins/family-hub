import SwiftUI
import FamilyHubKit

struct AreaDetailView: View {
    let areaID: String
    @Bindable var viewModel: InventoryViewModel
    @State private var scope: Scope = .all
    @State private var activeSheet: ActiveSheet?
    @State private var deletingItem: InventoryItem?

    enum Scope: Hashable { case all, low }
    enum ActiveSheet: Identifiable {
        case addItem, editItem(InventoryItem)
        var id: String {
            switch self {
            case .addItem: return "add"
            case .editItem(let item): return "edit-\(item.id)"
            }
        }
    }

    var body: some View {
        Group {
            if let area = viewModel.area(id: areaID) {
                List {
                    Section {
                        Picker("Show", selection: $scope) {
                            Text("All items").tag(Scope.all)
                            Text("Running low (\(area.lowCount))").tag(Scope.low)
                        }
                    }.listRowBackground(AppTheme.surface)
                    Section("Items") {
                        let items = scope == .low ? area.lowItems : area.items
                        if items.isEmpty { Text("No items").foregroundStyle(AppTheme.secondary) }
                        ForEach(items) { item in
                            itemRow(item)
                                .swipeActions(allowsFullSwipe: false) {
                                    Button("Delete", role: .destructive) { deletingItem = item }
                                }
                        }
                        Button("Add Item", systemImage: "plus") { activeSheet = .addItem }
                    }.listRowBackground(AppTheme.surface)
                }
                .appListStyle()
                .navigationTitle(area.name)
            } else {
                ContentUnavailableView("Area not found", systemImage: "shippingbox")
            }
        }
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("New Item", systemImage: "plus") { activeSheet = .addItem }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .addItem: ItemFormView(mode: .create(areaID: areaID), viewModel: viewModel)
            case .editItem(let item): ItemFormView(mode: .edit(item), viewModel: viewModel)
            }
        }
        .confirmationDialog("Delete \(deletingItem?.name ?? "item")?", isPresented: Binding(
            get: { deletingItem != nil }, set: { if !$0 { deletingItem = nil } }
        ), titleVisibility: .visible) {
            if let item = deletingItem {
                Button("Delete Item", role: .destructive) { Task { await viewModel.deleteItem(areaID: areaID, id: item.id) } }
            }
        }
        .errorAlert($viewModel.actionError)
        .alert("Shopping List", isPresented: Binding(
            get: { viewModel.shoppingMessage != nil }, set: { if !$0 { viewModel.shoppingMessage = nil } }
        )) { Button("OK") { viewModel.shoppingMessage = nil } }
        message: { Text(viewModel.shoppingMessage ?? "") }
    }

    private func itemRow(_ item: InventoryItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Button { activeSheet = .editItem(item) } label: {
                VStack(alignment: .leading, spacing: 4) {
                    HStack {
                        Text(item.name).font(.headline).foregroundStyle(AppTheme.ink)
                        Spacer()
                        Image(systemName: "pencil").foregroundStyle(AppTheme.accent).accessibilityHidden(true)
                    }
                    Text(item.statusLabel + " · Alert at or below \(item.lowAt)\(item.trackingMode == .level ? "%" : "")")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                }
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
            }
            .buttonStyle(.borderless)
            if item.isLow { Label("Running low", systemImage: "exclamationmark.circle").font(.caption) }
            HStack {
                Button {
                    Task { await viewModel.addToShopping(item) }
                } label: {
                    Image(systemName: "cart.badge.plus").frame(width: 44, height: 44)
                }
                .buttonStyle(.borderless)
                .accessibilityLabel("Add \(item.name) to shopping list")
                Spacer()
                switch item.trackingMode {
                case .count:
                    QuantityStepper(quantity: item.quantity, isLow: item.isLow, label: item.name,
                                    onDecrement: { Task { await viewModel.adjustQuantity(item: item, by: -1) } },
                                    onIncrement: { Task { await viewModel.adjustQuantity(item: item, by: 1) } })
                case .level: LevelBar(level: item.level, isLow: item.isLow)
                }
            }
        }
        .padding(.vertical, 4)
    }
}
