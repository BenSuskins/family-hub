import SwiftUI
import FamilyHubKit

struct ShoppingListView: View {
    let apiClient: any APIClientProtocol
    @State private var model: ShoppingViewModel
    @State private var inventoryModel: InventoryViewModel
    private let isHouseholdHome: Bool
    @State private var newName = ""
    @State private var renameID: String?
    @State private var renameDraft = ""
    @FocusState private var renameFocused: Bool
    @FocusState private var addFocused: Bool
    @State private var deletingItem: ShoppingItem?
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    @Environment(\.scenePhase) private var scenePhase
    @State private var isVisible = false

    init(apiClient: any APIClientProtocol, isHouseholdHome: Bool = false, model: ShoppingViewModel? = nil) {
        self.apiClient = apiClient
        self.isHouseholdHome = isHouseholdHome
        _model = State(wrappedValue: model ?? ShoppingViewModel(apiClient: apiClient))
        _inventoryModel = State(wrappedValue: InventoryViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            if isOverview {
                Section {
                    Text("The shopping list and your stock.").foregroundStyle(AppTheme.secondary)
                        .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 0))
                }.listRowBackground(Color.clear)
                Section {
                    NavigationLink(value: HouseholdDestination.shopping) {
                        FeatureEntryLabel(title: "Shopping list", subtitle: "\(model.remainingCount) to buy", systemImage: "cart")
                    }
                    .accessibilityLabel("Shopping list")
                    .accessibilityHint("\(model.remainingCount) to buy")
                    NavigationLink(value: HouseholdDestination.inventory) {
                        FeatureEntryLabel(title: "Inventory", subtitle: inventorySummary, systemImage: "cabinet")
                    }
                    .accessibilityLabel("Inventory")
                    .accessibilityHint(inventorySummary)
                }.listRowBackground(AppTheme.surface)
                inventorySection
            } else {
                if !model.canEdit {
                    Section {
                        if let date = model.capturedAt {
                            Label("Read-only copy", systemImage: "wifi.slash")
                            Text("Last updated \(date.formatted(date: .abbreviated, time: .shortened))")
                                .font(.caption).foregroundStyle(AppTheme.secondary)
                        } else {
                            Text(model.isLoading ? "Loading…" : "Shopping list unavailable.")
                        }
                        Button("Refresh") { Task { await model.load() } }
                    }.listRowBackground(AppTheme.surface)
                }
                Section {
                    Text("\(model.remainingCount) \(model.remainingCount == 1 ? "item" : "items") to buy")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 0))
                }.listRowBackground(Color.clear)
                Section {
                    HStack {
                        Button {
                            if newName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { addFocused = true }
                            else { add() }
                        } label: { Image(systemName: "plus").font(.title2) }
                        .frame(minWidth: 44, minHeight: 44)
                        .accessibilityLabel("Add item")
                        .buttonStyle(.borderless)
                        TextField("Item name", text: $newName, prompt: Text("Add an item"))
                            .focused($addFocused)
                            .accessibilityLabel("Item name")
                            .accessibilityIdentifier("Item name")
                            .submitLabel(.done)
                            .onSubmit { add() }
                    }
                    .disabled(!model.canEdit || model.changingIDs.contains("new-item"))
                }.listRowBackground(AppTheme.surface)
                Section {
                    if model.visibleItems.isEmpty { Text("The list is empty.").foregroundStyle(AppTheme.secondary) }
                    ForEach(model.visibleItems) { row($0) }
                }.listRowBackground(AppTheme.surface)
                if isHouseholdHome {
                    Section {
                        NavigationLink(value: HouseholdDestination.inventory) {
                            FeatureEntryLabel(title: "Inventory", subtitle: inventorySummary, systemImage: "cabinet")
                        }
                        .accessibilityLabel("Inventory")
                        .accessibilityHint(inventorySummary)
                    }.listRowBackground(AppTheme.surface)
                }
            }
        }
        .listStyle(.insetGrouped)
        .appListStyle()
        .navigationTitle(isHouseholdHome ? "Household" : "Shopping List")
        .onReceive(NotificationCenter.default.publisher(for: .familyHubIntentChanged)) { _ in Task { await refresh() } }
        .refreshable { await refresh() }
        .errorAlert($model.error)
        .toolbar {
            if let item = model.items.first(where: { $0.id == renameID }) {
                ToolbarItemGroup(placement: .keyboard) {
                    Spacer()
                    Button("Done") { saveRename(item) }
                        .accessibilityIdentifier("Save item name")
                        .disabled(!model.canEdit || renameDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || renameDraft.trimmingCharacters(in: .whitespacesAndNewlines).utf8.count > 200 || model.changingIDs.contains(item.id))
                }
            }
        }
        .confirmationDialog("Delete \(deletingItem?.name ?? "item")?", isPresented: Binding(
            get: { deletingItem != nil }, set: { if !$0 { deletingItem = nil } }
        ), titleVisibility: .visible) {
            if let item = deletingItem {
                Button("Delete Item", role: .destructive) { Task { await model.remove(item) } }
            }
        }
        .onAppear { isVisible = true; model.expirePurchases() }
        .onDisappear { isVisible = false }
        .task(id: scenePhase == .active && isVisible) { [shouldRefresh = scenePhase == .active && isVisible] in
            guard shouldRefresh else { return }
            await refresh()
            var lastRefresh = Date.now
            while !Task.isCancelled {
                model.expirePurchases()
                if lastRefresh.timeIntervalSinceNow < -30 { await refresh(); lastRefresh = .now }
                do { try await Task.sleep(for: .milliseconds(200)) } catch { return }
            }
        }
    }

    private var inventorySummary: String {
        switch inventoryModel.state {
        case .idle, .loading: return "Checking stock…"
        case .failed: return "Stock unavailable"
        case .loaded:
            let count = inventoryModel.totalRunningLow
            if inventoryModel.areas.isEmpty { return "Track stock around your home" }
            return count == 0 ? "All items in stock" : "\(count) \(count == 1 ? "item" : "items") running low"
        }
    }

    @ViewBuilder
    private var inventorySection: some View {
        switch inventoryModel.state {
        case .idle, .loading:
            Section { ProgressView("Checking stock") }.listRowBackground(AppTheme.surface)
        case .failed:
            Section("Inventory") {
                Text("Stock unavailable. Retry to check stock.")
                    .foregroundStyle(AppTheme.secondary)
                Button("Retry inventory") { Task { await inventoryModel.load() } }
            }.listRowBackground(AppTheme.surface)
        case .loaded:
            if !inventoryModel.runningLow.isEmpty {
                Section {
                    ForEach(inventoryModel.runningLow.prefix(4)) { entry in
                        lowStockRow(entry)
                    }
                } header: {
                    let layout = dynamicTypeSize.isAccessibilitySize
                        ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
                        : AnyLayout(HStackLayout())
                    layout {
                        Text("Running low").font(.system(.title2, design: .rounded, weight: .bold)).foregroundStyle(AppTheme.ink)
                        if !dynamicTypeSize.isAccessibilitySize { Spacer() }
                        if !isHouseholdHome {
                            NavigationLink("Inventory", value: HouseholdDestination.inventory).frame(minHeight: 44)
                        }
                    }.textCase(nil)
                }.listRowBackground(AppTheme.surface)
            }
        }
    }

    private func lowStockRow(_ entry: RunningLowItem) -> some View {
        let isOnList = model.items.contains { !$0.purchased && $0.inventoryItemID == entry.item.id }
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
            : AnyLayout(HStackLayout(spacing: 12))
        return layout {
            NavigationLink {
                AreaDetailView(areaID: entry.area.id, viewModel: inventoryModel)
            } label: {
                HStack(spacing: 12) {
                    AreaIconTile(area: entry.area).accessibilityHidden(true)
                    VStack(alignment: .leading, spacing: 4) {
                        Text(entry.item.name).foregroundStyle(AppTheme.ink)
                        Text(entry.area.name + " · " + entry.item.statusLabel + " · Low")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.secondary)
                    }
                    .fixedSize(horizontal: false, vertical: true)
                }
            }
            Button {
                Task { await model.addInventoryItem(entry.item) }
            } label: {
                Image(systemName: isOnList ? "checkmark.circle" : "plus.circle.fill")
                    .font(.title2)
                    .frame(width: 44, height: 44)
            }
            .buttonStyle(.borderless)
            .accessibilityLabel(isOnList ? "\(entry.item.name) is on the shopping list" : "Add \(entry.item.name) to shopping list")
            .disabled(isOnList || !model.canEdit || model.isChanging)
        }
    }

    private func refresh() async {
        await model.load()
        await inventoryModel.load()
    }

    private func add() {
        let name = newName
        Task {
            await model.add(name: name)
            if model.error == nil { newName = ""; addFocused = false }
        }
    }

    private var isOverview: Bool { isHouseholdHome && UIDevice.current.userInterfaceIdiom == .phone }

    private func row(_ item: ShoppingItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .center, spacing: 8) {
                Button { Task { _ = await model.toggle(item) } } label: {
                    Image(systemName: model.isPurchased(item) ? "checkmark.circle.fill" : "circle")
                        .font(.title2).frame(width: 44, height: 44)
                }
                .buttonStyle(.borderless)
                .accessibilityLabel(model.isPurchased(item) ? "Restore \(item.name)" : "Purchase \(item.name)")
                .disabled(!model.canEdit || renameID == item.id)
                if renameID == item.id {
                    TextField("Item name", text: $renameDraft, axis: .vertical)
                        .focused($renameFocused)
                        .accessibilityIdentifier("Rename item inline")
                        .submitLabel(.done)
                        .onSubmit { saveRename(item) }
                } else {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(item.name).strikethrough(model.isPurchased(item))
                        if model.isPurchased(item) {
                            TimelineView(.periodic(from: .now, by: 1)) { _ in
                                if model.changingIDs.contains(item.id) {
                                    Text("Saving purchase…")
                                } else if let seconds = model.remainingUndoSeconds(for: item) {
                                    Text(seconds > 0 ? "Hides in \(seconds)s" : "Hiding item…")
                                }
                            }.font(.caption).foregroundStyle(AppTheme.secondary)
                        }
                        if let area = inventoryModel.areas.first(where: { area in area.items.contains { $0.id == item.inventoryItemID } }) {
                            Text(area.name).font(.subheadline).foregroundStyle(AppTheme.secondary)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .contentShape(Rectangle())
                    .onTapGesture { beginRename(item) }
                    .accessibilityAddTraits(.isButton)
                    .accessibilityHint("Edit item name")
                }
                if renameID == item.id {
                    Button("Cancel") { renameID = nil; renameFocused = false }
                        .buttonStyle(.borderless).frame(minWidth: 44, minHeight: 44)
                } else if model.isPurchased(item) {
                    Button("Undo") { Task { _ = await model.toggle(item) } }
                        .buttonStyle(.borderless).frame(minWidth: 44, minHeight: 44)
                        .disabled(!model.canEdit)
                } else if renameID != item.id {
                    Menu {
                        Button("Delete", role: .destructive) { deletingItem = item }
                    } label: { Image(systemName: "ellipsis").frame(width: 44, height: 44) }
                    .accessibilityLabel("Actions for \(item.name)")
                    .disabled(!model.canEdit || model.changingIDs.contains(item.id))
                }
            }
            if renameID == item.id && renameDraft.trimmingCharacters(in: .whitespacesAndNewlines).utf8.count > 200 {
                Text("Use a shorter item name.").font(.footnote).foregroundStyle(AppTheme.destructive)
            }
            if model.changingIDs.contains(item.id) { ProgressView("Saving…").font(.caption) }
            if let error = model.rowErrors[item.id] {
                Text(error.localizedDescription).font(.footnote).foregroundStyle(AppTheme.destructive)
            }
        }
        .padding(.vertical, 4)
    }

    private func saveRename(_ item: ShoppingItem) {
        Task {
            if await model.rename(item, name: renameDraft) { renameID = nil; renameFocused = false }
        }
    }

    private func beginRename(_ item: ShoppingItem) {
        guard model.canEdit, !model.changingIDs.contains(item.id) else { return }
        renameID = item.id
        renameDraft = item.name
        renameFocused = true
    }
}
