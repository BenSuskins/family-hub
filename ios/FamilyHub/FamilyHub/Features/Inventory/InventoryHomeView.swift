import SwiftUI

struct InventoryHomeView: View {
    @State private var viewModel: InventoryViewModel
    private let apiClient: any APIClientProtocol
    @State private var activeSheet: ActiveSheet?

    /// Single sheet driver — stacking multiple `.sheet` modifiers on one view is a
    /// known SwiftUI pitfall, so create/edit share one enum-driven presentation.
    enum ActiveSheet: Identifiable {
        case addArea
        case editArea(InventoryArea)

        var id: String {
            switch self {
            case .addArea:           return "add"
            case .editArea(let area): return "edit-\(area.id)"
            }
        }
    }

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: InventoryViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { _ in
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        subtitle

                        if !viewModel.runningLow.isEmpty {
                            runningLowSection
                        }

                        areasSection
                        Spacer(minLength: 24)
                    }
                }
                .refreshable { await viewModel.load() }
            }
            .meshBackground()
            .navigationTitle("Inventory")
            .navigationBarTitleDisplayMode(.large)
            .navigationDestination(for: String.self) { areaID in
                AreaDetailView(areaID: areaID, viewModel: viewModel)
            }
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { activeSheet = .addArea } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(item: $activeSheet) { sheet in
                switch sheet {
                case .addArea:
                    AreaFormView(mode: .create, viewModel: viewModel)
                case .editArea(let area):
                    AreaFormView(mode: .edit(area), viewModel: viewModel)
                }
            }
            .errorAlert($viewModel.actionError)
        }
        .task { await viewModel.load() }
    }

    private var subtitle: some View {
        Text("\(viewModel.areas.count) areas · \(viewModel.totalRunningLow) running low")
            .font(.system(size: 14))
            .foregroundStyle(.secondary)
            .padding(.horizontal, 20)
            .padding(.top, 2)
            .padding(.bottom, 4)
    }

    // MARK: - Running low

    private var runningLowSection: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Running Low", color: InventoryStyle.low)
            VStack(spacing: 0) {
                ForEach(Array(viewModel.runningLow.enumerated()), id: \.element.id) { index, entry in
                    if index > 0 { Divider().padding(.leading, 30) }
                    NavigationLink(value: entry.area.id) {
                        runningLowRow(entry)
                    }
                    .buttonStyle(.plain)
                }
            }
            .glassCard(radius: 16)
            .padding(.horizontal, 16)
        }
    }

    private func runningLowRow(_ entry: RunningLowItem) -> some View {
        HStack(spacing: 12) {
            Circle()
                .fill(InventoryStyle.low)
                .frame(width: 8, height: 8)
            VStack(alignment: .leading, spacing: 1) {
                Text(entry.item.name)
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(.primary)
                Text(entry.area.name)
                    .font(.system(size: 13))
                    .foregroundStyle(.secondary)
            }
            Spacer(minLength: 8)
            Text(entry.item.statusLabel)
                .font(.system(size: 14, weight: .semibold))
                .monospacedDigit()
                .foregroundStyle(InventoryStyle.low)
            Image(systemName: "chevron.right")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 11)
        .frame(minHeight: 54)
        .contentShape(Rectangle())
    }

    // MARK: - Areas

    private var areasSection: some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeaderLabel(text: "Areas")
            VStack(spacing: 0) {
                if viewModel.areas.isEmpty {
                    emptyAreasRow
                } else {
                    ForEach(Array(viewModel.areas.enumerated()), id: \.element.id) { index, area in
                        if index > 0 { Divider().padding(.leading, 64) }
                        NavigationLink(value: area.id) {
                            areaRow(area)
                        }
                        .buttonStyle(.plain)
                        .contextMenu {
                            Button { activeSheet = .editArea(area) } label: {
                                Label("Edit Area", systemImage: "pencil")
                            }
                            Button(role: .destructive) {
                                Task { await viewModel.deleteArea(id: area.id) }
                            } label: {
                                Label("Delete Area", systemImage: "trash")
                            }
                        }
                    }
                    Divider().padding(.leading, 64)
                }
                addAreaRow
            }
            .glassCard(radius: 16)
            .padding(.horizontal, 16)
        }
    }

    private func areaRow(_ area: InventoryArea) -> some View {
        HStack(spacing: 13) {
            AreaIconTile(area: area, size: 38)
            VStack(alignment: .leading, spacing: 2) {
                Text(area.name)
                    .font(.system(size: 17, weight: .medium))
                    .foregroundStyle(.primary)
                HStack(spacing: 6) {
                    Text("\(area.items.count) items")
                    if area.lowCount > 0 {
                        Text("·")
                        Text("\(area.lowCount) low")
                            .foregroundStyle(InventoryStyle.low)
                            .fontWeight(.semibold)
                    }
                }
                .font(.system(size: 13))
                .foregroundStyle(.secondary)
            }
            Spacer(minLength: 8)
            Image(systemName: "chevron.right")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 11)
        .frame(minHeight: 62)
        .contentShape(Rectangle())
    }

    private var addAreaRow: some View {
        Button { activeSheet = .addArea } label: {
            HStack(spacing: 13) {
                RoundedRectangle(cornerRadius: 11, style: .continuous)
                    .fill(Color(.tertiarySystemFill))
                    .frame(width: 38, height: 38)
                    .overlay {
                        Image(systemName: "plus")
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundStyle(Color.accentColor)
                    }
                Text("Add area")
                    .font(.system(size: 17))
                    .foregroundStyle(Color.accentColor)
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 11)
            .frame(minHeight: 56)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private var emptyAreasRow: some View {
        Text("Add an area to start tracking stock.")
            .font(.system(size: 15))
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
    }
}
