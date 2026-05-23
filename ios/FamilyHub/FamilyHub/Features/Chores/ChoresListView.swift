import SwiftUI

struct ChoresListView: View {
    @State private var viewModel: ChoresViewModel
    @State private var scope: Scope = .all
    private let apiClient: any APIClientProtocol
    @State private var showCreateForm = false

    enum Scope: String, CaseIterable {
        case all = "All"
        case mine = "Mine"
        case overdue = "Overdue"
    }

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                scopeSegmented
                    .padding(.horizontal, 16)
                    .padding(.bottom, 14)

                if case .failed(let error) = viewModel.state {
                    Text(error.localizedDescription)
                        .foregroundStyle(.red)
                        .padding(.horizontal, 20)
                }

                choresContent

                Spacer(minLength: 24)
            }
        }
        .meshBackground()
        .navigationTitle("Chores")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { showCreateForm = true } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showCreateForm) {
            ChoreFormView(mode: .create, viewModel: viewModel)
        }
        .refreshable { await viewModel.load() }
        .task { await viewModel.load() }
    }

    // MARK: - Segmented control

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

    // MARK: - Content

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
                    ChoreListRow(
                        chore: chore,
                        user: viewModel.users[chore.assignedToUserID ?? ""],
                        apiClient: apiClient,
                        onComplete: {
                            Task {
                                await viewModel.complete(choreID: chore.id)
                            }
                        }
                    )
                }
                .buttonStyle(.plain)
            }
        }
        .glassCard(radius: 16)
        .padding(.horizontal, 16)
    }
}

// MARK: - ChoreListRow

private struct ChoreListRow: View {
    let chore: Chore
    let user: User?
    let apiClient: any APIClientProtocol
    let onComplete: () -> Void

    private var isOverdue: Bool { chore.status == .overdue }

    var body: some View {
        HStack(spacing: 12) {
            UserAvatar(user: user, size: 32, apiClient: apiClient)

            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.system(size: 16, weight: .medium))
                    .strikethrough(chore.status == .completed)
                    .foregroundStyle(chore.status == .completed ? .tertiary : .primary)
                    .lineLimit(1)
                HStack(spacing: 4) {
                    if let user {
                        Text(user.name)
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                        Text("·")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                    if isOverdue {
                        Text("Overdue")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(.red)
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 13))
                                .foregroundStyle(.secondary)
                        }
                    } else if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer(minLength: 0)

            Button(action: onComplete) {
                ZStack {
                    Circle()
                        .strokeBorder(Color.appGreen, lineWidth: 1.5)
                        .frame(width: 28, height: 28)
                        .opacity(chore.status == .completed ? 0 : 1)
                    Circle()
                        .fill(Color.appGreen)
                        .frame(width: 28, height: 28)
                        .opacity(chore.status == .completed ? 1 : 0)
                    Image(systemName: "checkmark")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(.white)
                        .opacity(chore.status == .completed ? 1 : 0)
                }
            }
            .buttonStyle(.plain)
            .sensoryFeedback(.success, trigger: chore.status == .completed)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(minHeight: 60)
    }
}
