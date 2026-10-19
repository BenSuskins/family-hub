import SwiftUI
import FamilyHubKit

struct ChoresListView: View {
    @State private var viewModel: ChoresViewModel
    @State private var mode: Mode = .all
    @State private var scope: Scope = .all
    @State private var activeSheet: ActiveSheet?
    @State private var deletingChore: Chore?
    private let apiClient: any APIClientProtocol

    enum Mode: String, CaseIterable { case all = "Chores", manage = "Manage" }
    enum Scope: String, CaseIterable { case all = "All", mine = "Mine", overdue = "Overdue" }
    enum ActiveSheet: Identifiable {
        case create
        case edit(Chore)
        var id: String {
            switch self {
            case .create: return "create"
            case .edit(let chore): return "edit-\(chore.id)"
            }
        }
    }

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { _ in
            List {
                Section {
                    Picker("Mode", selection: $mode) {
                        ForEach(Mode.allCases, id: \.self) { Text($0.rawValue).tag($0) }
                    }
                    if mode == .all {
                        Picker("Show", selection: $scope) {
                            ForEach(Scope.allCases, id: \.self) { Text($0.rawValue).tag($0) }
                        }
                    }
                }
                if mode == .manage {
                    Section("Chores") {
                        if viewModel.series.isEmpty {
                            ContentUnavailableView("No chores yet", systemImage: "checklist")
                        }
                        ForEach(viewModel.series) { chore in
                            Button { activeSheet = .edit(chore) } label: { manageRow(chore) }
                                .buttonStyle(.borderless)
                                .swipeActions {
                                    Button("Delete", role: .destructive) { deletingChore = chore }
                                }
                        }
                    }
                } else if scope == .mine {
                    choreSection("Mine", chores: pendingChores.filter { $0.assignedToUserID == viewModel.currentUserID })
                } else if scope == .overdue {
                    choreSection("Overdue", chores: viewModel.overdueChores)
                } else {
                    if pendingChores.isEmpty {
                        ContentUnavailableView("No chores due", systemImage: "checkmark.circle")
                    }
                    if !viewModel.overdueChores.isEmpty { choreSection("Overdue", chores: viewModel.overdueChores) }
                    if !viewModel.todayChores.isEmpty { choreSection("Today", chores: viewModel.todayChores) }
                    if !viewModel.upcomingChores.isEmpty { choreSection("Upcoming", chores: viewModel.upcomingChores) }
                }
            }
            .refreshable { await viewModel.load() }
        }
        .navigationTitle("Chores")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("New Chore", systemImage: "plus") { activeSheet = .create }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .create: ChoreFormView(mode: .create, viewModel: viewModel)
            case .edit(let chore): ChoreFormView(mode: .edit(chore), viewModel: viewModel)
            }
        }
        .errorAlert($viewModel.actionError)
        .confirmationDialog("Delete this chore?", isPresented: Binding(
            get: { deletingChore != nil }, set: { if !$0 { deletingChore = nil } }
        ), titleVisibility: .visible) {
            if let chore = deletingChore {
                Button("Delete Chore", role: .destructive) {
                    Task { await viewModel.deleteChore(id: chore.id) }
                }
            }
        }
        .task { await viewModel.load() }
        .onReceive(NotificationCenter.default.publisher(for: .familyHubIntentChanged)) { _ in Task { await viewModel.load() } }
        .onChange(of: scenePhase) { _, phase in if phase == .active { Task { await viewModel.load() } } }
    }

    private var pendingChores: [Chore] {
        viewModel.overdueChores + viewModel.todayChores + viewModel.upcomingChores
    }

    private func choreSection(_ title: String, chores: [Chore]) -> some View {
        Section(title) {
            if chores.isEmpty { Text("No chores due").foregroundStyle(.secondary) }
            ForEach(chores) { chore in
                HStack(spacing: 12) {
                    NavigationLink {
                        ChoreDetailView(chore: chore, viewModel: viewModel, apiClient: apiClient)
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(chore.name).font(.body)
                            if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                                Text(user.name).font(.subheadline).foregroundStyle(.secondary)
                            }
                            if let date = chore.formattedDueDate {
                                Text(date).font(.caption).foregroundStyle(.secondary)
                            }
                        }
                    }
                    Button {
                        Task { await viewModel.complete(choreID: chore.id) }
                    } label: {
                        CheckCircleView(isSelected: chore.status == .completed)
                            .frame(width: 44, height: 44)
                    }
                    .buttonStyle(.borderless)
                    .accessibilityLabel("Complete \(chore.name)")
                }
            }
        }
    }

    private func manageRow(_ chore: Chore) -> some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                Text(chore.name)
                    .font(.body.weight(.medium))
                    .foregroundStyle(.primary)
                HStack(spacing: 6) {
                    Label(chore.recurrenceSummary, systemImage: chore.isRecurring ? "repeat" : "calendar")
                        .labelStyle(.titleAndIcon)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let category = viewModel.categoryName(chore.categoryID) {
                        Text("· \(category)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                if !chore.eligibleAssignees.isEmpty {
                    Text(assigneeSummary(chore))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer(minLength: 0)
            Image(systemName: "chevron.right")
                .font(.footnote.weight(.semibold))
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(minHeight: 60)
        .contentShape(Rectangle())
    }

    private func assigneeSummary(_ chore: Chore) -> String {
        let names = chore.eligibleAssignees.compactMap { viewModel.users[$0]?.name }
        if names.isEmpty {
            return "\(chore.eligibleAssignees.count) assignee\(chore.eligibleAssignees.count == 1 ? "" : "s")"
        }
        return names.joined(separator: ", ")
    }
}
