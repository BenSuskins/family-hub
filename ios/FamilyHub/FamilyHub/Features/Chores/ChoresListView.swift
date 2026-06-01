import SwiftUI

struct ChoresListView: View {
    @State private var viewModel: ChoresViewModel
    @State private var mode: Mode = .all
    @State private var scope: Scope = .all
    private let apiClient: any APIClientProtocol
    @State private var activeSheet: ActiveSheet?

    enum Mode: String, CaseIterable {
        case all = "All Chores"
        case manage = "Manage"
    }

    enum Scope: String, CaseIterable {
        case all = "All"
        case mine = "Mine"
        case overdue = "Overdue"
    }

    /// Single sheet driver. Stacking multiple `.sheet` modifiers on one view is a
    /// known SwiftUI pitfall, so create/edit share one enum-driven presentation.
    enum ActiveSheet: Identifiable {
        case create
        case edit(Chore)

        var id: String {
            switch self {
            case .create:          return "create"
            case .edit(let chore): return "edit-\(chore.id)"
            }
        }
    }

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                segmented(Array(Mode.allCases), selection: $mode) { $0.rawValue }
                    .padding(.horizontal, 16)
                    .padding(.bottom, 14)

                if case .failed(let error) = viewModel.state {
                    ErrorStateView(error: error) { await viewModel.load() }
                        .frame(maxWidth: .infinity)
                        .padding(.top, 40)
                }

                switch mode {
                case .all:
                    scopeSegmented
                        .padding(.horizontal, 16)
                        .padding(.bottom, 14)
                    choresContent
                case .manage:
                    manageContent
                }

                Spacer(minLength: 24)
            }
        }
        .meshBackground()
        .navigationTitle("Manage Chores")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { activeSheet = .create } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .create:
                ChoreFormView(mode: .create, viewModel: viewModel)
            case .edit(let chore):
                ChoreFormView(mode: .edit(chore), viewModel: viewModel)
            }
        }
        .refreshable { await viewModel.load() }
        .task { await viewModel.load() }
    }

    // MARK: - Segmented controls

    private func segmented<T: Hashable>(
        _ options: [T],
        selection: Binding<T>,
        label: @escaping (T) -> String
    ) -> some View {
        HStack(spacing: 2) {
            ForEach(options, id: \.self) { option in
                Button {
                    withAnimation(.spring(duration: 0.2)) { selection.wrappedValue = option }
                } label: {
                    Text(label(option))
                        .font(.system(size: 13, weight: .medium))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 7)
                        .background(
                            selection.wrappedValue == option
                                ? Color(UIColor.secondarySystemGroupedBackground)
                                : Color.clear,
                            in: RoundedRectangle(cornerRadius: 7)
                        )
                        .shadow(
                            color: selection.wrappedValue == option ? .black.opacity(0.08) : .clear,
                            radius: 1, x: 0, y: 1
                        )
                }
                .buttonStyle(.plain)
                .foregroundStyle(Color.primary)
            }
        }
        .padding(2)
        .background(Color(UIColor.quaternarySystemFill), in: RoundedRectangle(cornerRadius: 9))
    }

    private var scopeSegmented: some View {
        HStack(spacing: 2) {
            ForEach(Scope.allCases, id: \.self) { s in
                Button {
                    withAnimation(.spring(duration: 0.2)) { scope = s }
                } label: {
                    HStack(spacing: 4) {
                        Text(s.rawValue)
                            .font(.system(size: 13, weight: .medium))
                        if s == .overdue && !viewModel.overdueChores.isEmpty {
                            Text("\(viewModel.overdueChores.count)")
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 5)
                                .padding(.vertical, 1)
                                .background(Color.red, in: Capsule())
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 7)
                    .background(
                        scope == s
                            ? Color(UIColor.secondarySystemGroupedBackground)
                            : Color.clear,
                        in: RoundedRectangle(cornerRadius: 7)
                    )
                    .shadow(
                        color: scope == s ? .black.opacity(0.08) : .clear,
                        radius: 1, x: 0, y: 1
                    )
                }
                .buttonStyle(.plain)
                .foregroundStyle(Color.primary)
            }
        }
        .padding(2)
        .background(Color(UIColor.quaternarySystemFill), in: RoundedRectangle(cornerRadius: 9))
    }

    // MARK: - All Chores content

    @ViewBuilder
    private var choresContent: some View {
        switch scope {
        case .all:
            allContent
        case .mine:
            mineContent
        case .overdue:
            overdueOnlyContent
        }
    }

    private var allContent: some View {
        VStack(alignment: .leading, spacing: 0) {
            if !viewModel.overdueChores.isEmpty {
                choreSectionHeader("Overdue", color: .red)
                choreSection(viewModel.overdueChores)
                    .padding(.bottom, 16)
            }
            if !viewModel.todayChores.isEmpty {
                SectionHeaderLabel(text: "Today")
                choreSection(viewModel.todayChores)
                    .padding(.bottom, 16)
            }
            if !viewModel.upcomingChores.isEmpty {
                SectionHeaderLabel(text: "Upcoming")
                choreSection(viewModel.upcomingChores)
            }
            if viewModel.overdueChores.isEmpty && viewModel.todayChores.isEmpty && viewModel.upcomingChores.isEmpty {
                if case .loaded = viewModel.state {
                    ContentUnavailableView("All caught up!", systemImage: "checkmark.circle.fill")
                        .padding(.top, 40)
                }
            }
        }
    }

    private var mineContent: some View {
        let myChores = allPendingChores.filter { $0.assignedToUserID == viewModel.currentUserID }
        return Group {
            if myChores.isEmpty {
                if case .loaded = viewModel.state {
                    ContentUnavailableView("Nothing assigned to you", systemImage: "checkmark.circle")
                        .padding(.top, 40)
                }
            } else {
                choreSection(myChores)
                    .padding(.top, 8)
            }
        }
    }

    private var overdueOnlyContent: some View {
        Group {
            if viewModel.overdueChores.isEmpty {
                if case .loaded = viewModel.state {
                    ContentUnavailableView("No overdue chores", systemImage: "clock.badge.checkmark")
                        .padding(.top, 40)
                }
            } else {
                VStack(alignment: .leading, spacing: 0) {
                    choreSectionHeader("Overdue", color: .red)
                    choreSection(viewModel.overdueChores)
                }
            }
        }
    }

    private func choreSectionHeader(_ label: String, color: Color = .secondary) -> some View {
        HStack {
            Text(label.uppercased())
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(color)
                .kerning(0.5)
            Spacer()
        }
        .padding(.horizontal, 20)
        .padding(.top, 24)
        .padding(.bottom, 8)
    }

    private var allPendingChores: [Chore] {
        viewModel.overdueChores + viewModel.todayChores + viewModel.upcomingChores
    }

    // MARK: - Chore section card

    private func choreSection(_ chores: [Chore]) -> some View {
        VStack(spacing: 0) {
            ForEach(Array(chores.enumerated()), id: \.element.id) { index, chore in
                if index > 0 {
                    Divider().padding(.leading, 64)
                }
                NavigationLink {
                    ChoreDetailView(chore: chore, viewModel: viewModel, apiClient: apiClient)
                } label: {
                    ChoreRowView(
                        chore: chore,
                        user: viewModel.users[chore.assignedToUserID ?? ""],
                        apiClient: apiClient,
                        onComplete: {
                            Task {
                                await viewModel.complete(choreID: chore.id)
                            }
                        },
                        minHeight: 60,
                        showDateWhenOverdue: true
                    )
                }
                .buttonStyle(.plain)
            }
        }
        .glassCard(radius: 16)
        .padding(.horizontal, 16)
    }

    // MARK: - Manage content

    @ViewBuilder
    private var manageContent: some View {
        let series = viewModel.series
        if series.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView {
                    Label("No chores yet", systemImage: "checklist")
                } description: {
                    Text("Create a chore to start managing your family's tasks.")
                } actions: {
                    Button("New Chore") { activeSheet = .create }
                        .buttonStyle(.borderedProminent)
                }
                .padding(.top, 24)
            }
        } else {
            manageSection(series)
        }
    }

    private func manageSection(_ chores: [Chore]) -> some View {
        VStack(spacing: 0) {
            ForEach(Array(chores.enumerated()), id: \.element.id) { index, chore in
                if index > 0 {
                    Divider().padding(.leading, 16)
                }
                Button {
                    activeSheet = .edit(chore)
                } label: {
                    manageRow(chore)
                }
                .buttonStyle(.plain)
                .contextMenu {
                    Button { activeSheet = .edit(chore) } label: {
                        Label("Edit", systemImage: "pencil")
                    }
                    Button(role: .destructive) {
                        Task { await viewModel.deleteChore(id: chore.id) }
                    } label: {
                        Label("Delete", systemImage: "trash")
                    }
                }
            }
        }
        .glassCard(radius: 16)
        .padding(.horizontal, 16)
    }

    private func manageRow(_ chore: Chore) -> some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                Text(chore.name)
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(.primary)
                HStack(spacing: 6) {
                    Label(chore.recurrenceSummary, systemImage: chore.isRecurring ? "repeat" : "calendar")
                        .labelStyle(.titleAndIcon)
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                    if let category = viewModel.categoryName(chore.categoryID) {
                        Text("· \(category)")
                            .font(.system(size: 12))
                            .foregroundStyle(.secondary)
                    }
                }
                if !chore.eligibleAssignees.isEmpty {
                    Text(assigneeSummary(chore))
                        .font(.system(size: 12))
                        .foregroundStyle(.tertiary)
                }
            }
            Spacer(minLength: 0)
            Image(systemName: "chevron.right")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(.tertiary)
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
