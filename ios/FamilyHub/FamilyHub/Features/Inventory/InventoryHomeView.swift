import SwiftUI
import FamilyHubKit

struct InventoryHomeView: View {
    @State private var viewModel: InventoryViewModel
    @State private var activeSheet: ActiveSheet?
    @State private var deletingArea: InventoryArea?
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    enum ActiveSheet: Identifiable {
        case addArea
        case editArea(InventoryArea)
        var id: String {
            switch self {
            case .addArea: return "add"
            case .editArea(let area): return "edit-\(area.id)"
            }
        }
    }

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: InventoryViewModel(apiClient: apiClient))
    }

    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { _ in
            List {
                Section("Areas") {
                    if viewModel.areas.isEmpty {
                        Text("Add an area to track stock.").foregroundStyle(AppTheme.secondary)
                    }
                    ForEach(viewModel.areas) { area in
                        let layout = dynamicTypeSize.isAccessibilitySize
                            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
                            : AnyLayout(HStackLayout())
                        layout {
                        NavigationLink {
                            AreaDetailView(areaID: area.id, viewModel: viewModel)
                        } label: {
                            HStack(spacing: 12) {
                                if !dynamicTypeSize.isAccessibilitySize {
                                    AreaIconTile(area: area, size: 38).accessibilityHidden(true)
                                }
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(area.name).font(.headline)
                                    Text("\(area.items.count) items · \(area.lowCount) low")
                                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                                }
                            }
                        }
                            Menu {
                                Button("Edit Area", systemImage: "pencil") { activeSheet = .editArea(area) }
                                Button("Delete Area", systemImage: "trash", role: .destructive) { deletingArea = area }
                            } label: { Image(systemName: "ellipsis").frame(width: 44, height: 44) }
                            .accessibilityLabel("Actions for \(area.name)")
                        }
                        .swipeActions {
                            Button("Delete", role: .destructive) { deletingArea = area }
                            Button("Edit") { activeSheet = .editArea(area) }
                        }
                        .contextMenu {
                            Button("Edit Area", systemImage: "pencil") { activeSheet = .editArea(area) }
                            Button("Delete Area", systemImage: "trash", role: .destructive) { deletingArea = area }
                        }
                    }
                    Button("Add Area", systemImage: "plus") { activeSheet = .addArea }
                }.listRowBackground(AppTheme.surface)
                if !viewModel.runningLow.isEmpty {
                    Section("Running Low") {
                        ForEach(viewModel.runningLow) { entry in
                            HStack {
                                NavigationLink {
                                    AreaDetailView(areaID: entry.area.id, viewModel: viewModel)
                                } label: {
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text(entry.item.name)
                                        Text(entry.area.name + " · " + entry.item.statusLabel)
                                            .font(.subheadline).foregroundStyle(AppTheme.secondary)
                                    }
                                }
                                Button {
                                    Task { await viewModel.addToShopping(entry.item) }
                                } label: {
                                    Image(systemName: "cart.badge.plus").frame(width: 44, height: 44)
                                }
                                .buttonStyle(.borderless)
                                .accessibilityLabel("Add \(entry.item.name) to shopping list")
                            }
                        }
                    }.listRowBackground(AppTheme.surface)
                }
            }
            .refreshable { await viewModel.load() }
        }
        .appListStyle()
        .navigationTitle("Inventory")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("New Area", systemImage: "plus") { activeSheet = .addArea }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .addArea: AreaFormView(mode: .create, viewModel: viewModel)
            case .editArea(let area): AreaFormView(mode: .edit(area), viewModel: viewModel)
            }
        }
        .confirmationDialog("Delete \(deletingArea?.name ?? "area") and its items?", isPresented: Binding(
            get: { deletingArea != nil }, set: { if !$0 { deletingArea = nil } }
        ), titleVisibility: .visible) {
            if let area = deletingArea {
                Button("Delete Area", role: .destructive) { Task { await viewModel.deleteArea(id: area.id) } }
            }
        }
        .errorAlert($viewModel.actionError)
        .alert("Shopping List", isPresented: Binding(
            get: { viewModel.shoppingMessage != nil }, set: { if !$0 { viewModel.shoppingMessage = nil } }
        )) { Button("OK") { viewModel.shoppingMessage = nil } }
        message: { Text(viewModel.shoppingMessage ?? "") }
        .task { await viewModel.load() }
        .onReceive(NotificationCenter.default.publisher(for: .familyHubIntentChanged)) { _ in Task { await viewModel.load() } }
        .onChange(of: scenePhase) { _, phase in if phase == .active { Task { await viewModel.load() } } }
    }
}
