import SwiftUI

struct ChoresView: View {
    @State private var viewModel: ChoresViewModel
    @State private var selectedTab: Tab = .pending

    enum Tab { case pending, completed }

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                VStack(spacing: 0) {
                    segmentControl
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)

                    if case .failed(let error) = viewModel.state {
                        Text(error.localizedDescription)
                            .font(.system(size: 13))
                            .foregroundStyle(Theme.statusRed)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                    }

                    List {
                        if selectedTab == .pending {
                            pendingContent
                        } else {
                            completedContent
                        }
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                    .refreshable { await viewModel.load() }
                }
            }
            .navigationTitle("Chores")
            .navigationBarTitleDisplayMode(.large)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Image(systemName: "line.3.horizontal.decrease")
                        .foregroundStyle(Theme.textMuted)
                }
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Segment control

    private var segmentControl: some View {
        HStack(spacing: 0) {
            segmentButton("Pending", tab: .pending)
            segmentButton("Completed", tab: .completed)
        }
        .padding(3)
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }

    private func segmentButton(_ label: String, tab: Tab) -> some View {
        Button {
            withAnimation(.easeInOut(duration: 0.15)) { selectedTab = tab }
        } label: {
            Text(label)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(selectedTab == tab ? Theme.textPrimary : Theme.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
                .background(selectedTab == tab ? Theme.surfaceElevated : Color.clear)
                .clipShape(RoundedRectangle(cornerRadius: 8))
        }
        .buttonStyle(.plain)
    }

    // MARK: - List sections

    @ViewBuilder
    private var pendingContent: some View {
        if !viewModel.overdueChores.isEmpty {
            Section {
                ForEach(viewModel.overdueChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            } header: {
                sectionHeader("Overdue", color: Theme.statusRed)
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Theme.surface)
            .listRowSeparatorTint(Theme.borderDivider)
        }

        if !viewModel.dueSoonChores.isEmpty {
            Section {
                ForEach(viewModel.dueSoonChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            } header: {
                sectionHeader("Due Soon", color: Theme.statusAmber)
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Theme.surface)
            .listRowSeparatorTint(Theme.borderDivider)
        }

        if viewModel.overdueChores.isEmpty && viewModel.dueSoonChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("All done!", systemImage: "checkmark.circle.fill")
                    .listRowBackground(Color.clear)
            }
        }
    }

    @ViewBuilder
    private var completedContent: some View {
        if viewModel.completedChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("No completed chores", systemImage: "clock")
                    .listRowBackground(Color.clear)
            }
        } else {
            Section {
                ForEach(viewModel.completedChores) { chore in
                    choreRow(chore, isCompleted: true)
                }
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Theme.surface)
            .listRowSeparatorTint(Theme.borderDivider)
        }
    }

    // MARK: - Row

    private func choreRow(_ chore: Chore, isCompleted: Bool) -> some View {
        NavigationLink {
            ChoreDetailView(chore: chore, viewModel: viewModel)
        } label: {
            HStack(spacing: 10) {
                if isCompleted {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(Theme.statusGreen)
                        .font(.system(size: 32))
                        .frame(width: 32, height: 32)
                } else {
                    UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text(chore.name)
                        .font(.system(size: 15, weight: .medium))
                        .foregroundStyle(Theme.textPrimary)
                    HStack(spacing: 6) {
                        if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                            Text(user.name)
                                .font(.system(size: 12))
                                .foregroundStyle(Theme.textSecondary)
                        }
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 11))
                                .foregroundStyle(isCompleted ? Theme.textMuted :
                                    (chore.status == .overdue ? Theme.statusRed : Theme.statusAmber))
                        }
                    }
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(Theme.textMuted)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
        }
        .buttonStyle(.plain)
    }

    private func sectionHeader(_ title: String, color: Color) -> some View {
        Text(title)
            .font(.system(size: 11, weight: .semibold))
            .textCase(.uppercase)
            .tracking(0.6)
            .foregroundStyle(color)
            .padding(.horizontal, 14)
            .padding(.vertical, 6)
            .listRowInsets(EdgeInsets())
    }
}
