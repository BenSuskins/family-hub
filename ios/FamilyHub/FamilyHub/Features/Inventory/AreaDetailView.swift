import SwiftUI
import FamilyHubKit

struct AreaDetailView: View {
    let areaID: String
    @Bindable var viewModel: InventoryViewModel
    @State private var scope: Scope = .all
    @State private var activeSheet: ActiveSheet?
    @State private var deletingItem: InventoryItem?
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

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
                        if dynamicTypeSize.isAccessibilitySize {
                            filterPicker(area)
                        } else {
                            filterPicker(area).pickerStyle(.segmented)
                        }
                    }
                    .listRowBackground(Color.clear)
                    .listRowInsets(EdgeInsets())
                    let items = scope == .low ? area.lowItems : area.items
                    if items.isEmpty {
                        Section {
                            ContentUnavailableView {
                                Label(area.items.isEmpty ? "No items yet" : "No items running low", systemImage: "cabinet")
                            } description: {
                                Text(area.items.isEmpty ? "Add an item to track its stock here." : "All items in this area are above their alert levels.")
                            }
                        }.listRowBackground(Color.clear)
                        Section {
                            if area.items.isEmpty {
                                Button("Add item") { activeSheet = .addItem }
                                    .buttonStyle(AppPrimaryButtonStyle())
                            } else {
                                Button("Show all items") { scope = .all }
                                    .frame(maxWidth: .infinity, minHeight: 52)
                                    .background(AppTheme.sage, in: Capsule())
                            }
                        }
                        .listRowBackground(Color.clear)
                        .listRowInsets(EdgeInsets())
                    } else {
                        ForEach(items) { item in
                            Section {
                                itemRow(item)
                                    .swipeActions(allowsFullSwipe: false) {
                                        Button("Delete", role: .destructive) { deletingItem = item }
                                    }
                            }.listRowBackground(AppTheme.surface)
                        }
                        Section {
                            Button("Add item") { activeSheet = .addItem }
                                .frame(maxWidth: .infinity, minHeight: 52)
                                .background(AppTheme.sage, in: Capsule())
                        }
                        .listRowBackground(Color.clear)
                        .listRowInsets(EdgeInsets())
                    }
                }
                .appListStyle()
                .listSectionSpacing(12)
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

    private func filterPicker(_ area: InventoryArea) -> some View {
        Picker("Show", selection: $scope) {
            Text("All items").tag(Scope.all)
            Text("Running low · \(area.lowCount)").tag(Scope.low)
        }
        .accessibilityLabel("Inventory filter")
    }

    private func itemRow(_ item: InventoryItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Button { activeSheet = .editItem(item) } label: {
                VStack(alignment: .leading, spacing: 4) {
                    HStack {
                        Text(item.name).font(.headline).foregroundStyle(AppTheme.ink)
                        Spacer()
                        Image(systemName: "chevron.right").foregroundStyle(AppTheme.secondary).accessibilityHidden(true)
                    }
                    Text(item.statusLabel + " · Alert at or below \(item.lowAt)\(item.trackingMode == .level ? "%" : "")")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                }
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
            }
            .buttonStyle(.borderless)
            if item.isLow { Label("Running low", systemImage: "exclamationmark.circle").font(.caption).foregroundStyle(AppTheme.clay) }
            let layout = dynamicTypeSize.isAccessibilitySize
                ? AnyLayout(VStackLayout(alignment: .leading, spacing: 12))
                : AnyLayout(HStackLayout())
            layout {
                Button {
                    Task { await viewModel.addToShopping(item) }
                } label: {
                    Label("Add to Shopping", systemImage: "cart.badge.plus")
                        .font(.subheadline)
                        .frame(minHeight: 44)
                }
                .buttonStyle(.borderless)
                .accessibilityLabel("Add \(item.name) to shopping list")
                if !dynamicTypeSize.isAccessibilitySize { Spacer() }
                switch item.trackingMode {
                case .count:
                    QuantityStepper(quantity: item.quantity, isLow: item.isLow, label: item.name,
                                    onDecrement: { Task { await viewModel.adjustQuantity(item: item, by: -1) } },
                                    onIncrement: { Task { await viewModel.adjustQuantity(item: item, by: 1) } })
                case .level:
                    LevelStepper(
                        level: item.level,
                        isLow: item.isLow,
                        label: item.name,
                        onDecrement: { Task { await viewModel.setLevel(item: item, to: item.level - 5) } },
                        onIncrement: { Task { await viewModel.setLevel(item: item, to: item.level + 5) } }
                    )
                }
            }
        }
        .padding(.vertical, 4)
    }
}
