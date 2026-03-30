import SwiftUI

struct ChoresListView: View {
    @State private var viewModel: ChoresViewModel
    @State private var selectedTab: Tab = .pending
    private let apiClient: any APIClientProtocol

    enum Tab: String, CaseIterable {
        case pending = "Pending"
        case completed = "Completed"
    }

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            Picker("Filter", selection: $selectedTab) {
                ForEach(Tab.allCases, id: \.self) { tab in
                    Text(tab.rawValue).tag(tab)
                }
            }
            .pickerStyle(.segmented)
            .listRowBackground(Color.clear)
            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))

            if case .failed(let error) = viewModel.state {
                Section {
                    Text(error.localizedDescription)
                        .foregroundStyle(.red)
                }
            }

            if selectedTab == .pending {
                pendingContent
            } else {
                completedContent
            }
        }
        .listStyle(.insetGrouped)
        .refreshable { await viewModel.load() }
        .navigationTitle("Chores")
        .task { await viewModel.load() }
    }

    @ViewBuilder
    private var pendingContent: some View {
        if !viewModel.overdueChores.isEmpty {
            Section("Overdue") {
                ForEach(viewModel.overdueChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            }
        }

        if !viewModel.dueSoonChores.isEmpty {
            Section("Due Soon") {
                ForEach(viewModel.dueSoonChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            }
        }

        if viewModel.overdueChores.isEmpty && viewModel.dueSoonChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("All done!", systemImage: "checkmark.circle.fill")
            }
        }
    }

    @ViewBuilder
    private var completedContent: some View {
        if viewModel.completedChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("No completed chores", systemImage: "clock")
            }
        } else {
            Section {
                ForEach(viewModel.completedChores) { chore in
                    choreRow(chore, isCompleted: true)
                }
            }
        }
    }

    private func choreRow(_ chore: Chore, isCompleted: Bool) -> some View {
        NavigationLink {
            ChoreDetailView(chore: chore, viewModel: viewModel)
        } label: {
            HStack(spacing: 10) {
                if isCompleted {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                        .font(.title2)
                } else {
                    UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32, apiClient: apiClient)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text(chore.name)
                        .font(.body)
                    HStack(spacing: 6) {
                        if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                            Text(user.name)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.caption2)
                                .foregroundStyle(isCompleted ? Color.secondary :
                                    (chore.status == .overdue ? Color.red : Color.orange))
                        }
                    }
                }
            }
        }
        .swipeActions(edge: .leading) {
            if !isCompleted {
                Button {
                    Task { await viewModel.complete(choreID: chore.id) }
                } label: {
                    Label("Done", systemImage: "checkmark")
                }
                .tint(.green)
            }
        }
    }
}
