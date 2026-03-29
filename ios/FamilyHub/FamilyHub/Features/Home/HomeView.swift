import SwiftUI

struct HomeView: View {
    @State private var viewModel: HomeViewModel
    @State private var showProfile = false
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: HomeViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            List {
                switch viewModel.state {
                case .idle, .loading:
                    Section {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    }
                case .failed(let error):
                    Section {
                        Text(error.localizedDescription)
                            .foregroundStyle(.red)
                    }
                case .loaded(let stats):
                    statsSection(stats)
                    mealsSection
                    choreSection(stats)
                }
            }
            .listStyle(.insetGrouped)
            .refreshable { await viewModel.load() }
            .navigationTitle("Home")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showProfile = true
                    } label: {
                        UserAvatar(user: nil, size: 32)
                    }
                }
            }
            .sheet(isPresented: $showProfile) {
                ProfileView()
            }
        }
        .task { await viewModel.load() }
    }

    @ViewBuilder
    private func choreSection(_ stats: DashboardStats) -> some View {
        let allDue = stats.choresOverdueList + stats.choresDueTodayList
        Section {
            if allDue.isEmpty {
                Label("All caught up!", systemImage: "checkmark.circle")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(allDue) { chore in
                    choreRow(chore)
                        .swipeActions(edge: .leading) {
                            Button {
                                Task { await viewModel.completeChore(id: chore.id) }
                            } label: {
                                Label("Done", systemImage: "checkmark")
                            }
                            .tint(.green)
                        }
                }
            }
            NavigationLink {
                ChoresListView(apiClient: apiClient)
            } label: {
                Text("See All Chores")
            }
        } header: {
            Text("Chores")
        }
    }

    private func choreRow(_ chore: Chore) -> some View {
        HStack(spacing: 10) {
            UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.body)
                HStack(spacing: 6) {
                    if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                        Text(user.name)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if let badge = chore.badge {
                        Text(badge.label)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(badge.color)
                    }
                    if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.caption2)
                            .foregroundStyle(chore.status == .overdue ? .red : .orange)
                    }
                }
            }
        }
    }

    private var mealsSection: some View {
        Section {
            mealRow(label: "Lunch", name: nil)
            mealRow(label: "Dinner", name: nil)
        } header: {
            Text("Today's Meals")
        }
    }

    private func mealRow(label: String, name: String?) -> some View {
        HStack {
            Text(label)
                .foregroundStyle(.secondary)
            Spacer()
            Text(name ?? "—")
                .foregroundStyle(name != nil ? .primary : .tertiary)
        }
    }

    private func statsSection(_ stats: DashboardStats) -> some View {
        Section {
            HStack(spacing: 12) {
                statItem(value: stats.choresDueToday + stats.choresOverdue, label: "Chores due")
                Divider()
                statItem(value: 0, label: "Meals planned")
                Divider()
                statItem(value: 0, label: "Events")
            }
            .padding(.vertical, 4)
        } header: {
            Text("This Week")
        }
    }

    private func statItem(value: Int, label: String) -> some View {
        VStack(spacing: 2) {
            Text("\(value)")
                .font(.title2.bold())
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }
}
