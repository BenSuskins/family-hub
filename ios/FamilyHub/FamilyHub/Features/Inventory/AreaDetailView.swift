import SwiftUI

struct AreaDetailView: View {
    let areaID: String
    @Bindable var viewModel: InventoryViewModel

    @State private var scope: Scope = .all
    @State private var activeSheet: ActiveSheet?

    enum Scope: Hashable { case all, low }

    enum ActiveSheet: Identifiable {
        case addItem
        case editItem(InventoryItem)

        var id: String {
            switch self {
            case .addItem:            return "add"
            case .editItem(let item): return "edit-\(item.id)"
            }
        }
    }

    private var area: InventoryArea? { viewModel.area(id: areaID) }

    var body: some View {
        Group {
            if let area {
                content(for: area)
            } else {
                // Area was deleted while open — fall back to the list.
                ContentUnavailableView("Area not found", systemImage: "shippingbox")
            }
        }
    }

    private func content(for area: InventoryArea) -> some View {
        let visibleItems = scope == .low ? area.lowItems : area.items

        return ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                header(area)
                scopePicker(area)

                VStack(spacing: 0) {
                    ForEach(Array(visibleItems.enumerated()), id: \.element.id) { index, item in
                        if index > 0 { Divider().padding(.leading, 16) }
                        itemRow(item)
                    }
                    if !area.items.isEmpty { Divider().padding(.leading, 16) }
                    addItemRow
                }
                .glassCard(radius: 16)
                .padding(.horizontal, 16)

                if visibleItems.isEmpty && scope == .low {
                    Text("Nothing low here 🎉")
                        .font(.system(size: 15))
                        .foregroundStyle(.tertiary)
                        .frame(maxWidth: .infinity)
                        .padding(.top, 40)
                }

                Spacer(minLength: 24)
            }
        }
        .meshBackground()
        .navigationTitle(area.name)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { activeSheet = .addItem } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .addItem:
                ItemFormView(mode: .create(areaID: areaID), viewModel: viewModel)
            case .editItem(let item):
                ItemFormView(mode: .edit(item), viewModel: viewModel)
            }
        }
        .errorAlert($viewModel.actionError)
    }

    // MARK: - Header

    private func header(_ area: InventoryArea) -> some View {
        HStack(spacing: 14) {
            AreaIconTile(area: area, size: 52)
            VStack(alignment: .leading, spacing: 2) {
                Text(area.name)
                    .font(.system(size: 24, weight: .bold))
                HStack(spacing: 4) {
                    Text("\(area.items.count) items")
                    if area.lowCount > 0 {
                        Text("·")
                        Text("\(area.lowCount) running low")
                            .foregroundStyle(InventoryStyle.low)
                            .fontWeight(.semibold)
                    }
                }
                .font(.system(size: 14))
                .foregroundStyle(.secondary)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 20)
        .padding(.top, 6)
        .padding(.bottom, 14)
    }

    private func scopePicker(_ area: InventoryArea) -> some View {
        Picker("Filter", selection: $scope) {
            Text("All").tag(Scope.all)
            Text(area.lowCount > 0 ? "Low · \(area.lowCount)" : "Low").tag(Scope.low)
        }
        .pickerStyle(.segmented)
        .padding(.horizontal, 16)
        .padding(.bottom, 14)
    }

    // MARK: - Item row

    private func itemRow(_ item: InventoryItem) -> some View {
        HStack(spacing: 12) {
            Button { activeSheet = .editItem(item) } label: {
                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 7) {
                        Text(item.name)
                            .font(.system(size: 16, weight: .medium))
                            .foregroundStyle(.primary)
                        if item.isLow { LowBadge() }
                    }
                    Text(subtitle(for: item))
                        .font(.system(size: 13))
                        .monospacedDigit()
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            QuantityStepper(
                quantity: item.quantity,
                isLow: item.isLow,
                onDecrement: { Task { await viewModel.adjustQuantity(item: item, by: -1) } },
                onIncrement: { Task { await viewModel.adjustQuantity(item: item, by: 1) } }
            )
        }
        .padding(.leading, 16)
        .padding(.trailing, 14)
        .padding(.vertical, 12)
        .frame(minHeight: 64)
        .swipeActions(edge: .trailing) {
            Button(role: .destructive) {
                Task { await viewModel.deleteItem(areaID: areaID, id: item.id) }
            } label: {
                Label("Delete", systemImage: "trash")
            }
        }
    }

    private func subtitle(for item: InventoryItem) -> String {
        let unit = item.unit.isEmpty ? "" : " \(item.unit)"
        return "\(item.quantity)\(unit) · par \(item.par)"
    }

    private var addItemRow: some View {
        Button { activeSheet = .addItem } label: {
            HStack(spacing: 10) {
                Image(systemName: "plus")
                    .font(.system(size: 16, weight: .semibold))
                Text("Add item")
                    .font(.system(size: 16))
                Spacer(minLength: 0)
            }
            .foregroundStyle(Color.accentColor)
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .frame(minHeight: 52)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
