import SwiftUI

struct DashboardView: View {
    @State private var viewModel: DashboardViewModel

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: DashboardViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                switch viewModel.state {
                case .idle, .loading:
                    ProgressView()
                case .loaded(let stats):
                    dashboardContent(stats)
                case .failed(let error):
                    ContentUnavailableView(
                        "Failed to load",
                        systemImage: "exclamationmark.triangle",
                        description: Text(error.localizedDescription)
                    )
                }
            }
            .navigationTitle("Dashboard")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Refresh") {
                        Task { await viewModel.load() }
                    }
                }
            }
        }
        .task { await viewModel.load() }
    }

    @ViewBuilder
    private func dashboardContent(_ stats: DashboardStats) -> some View {
        List {
            Section {
                HStack(spacing: 16) {
                    StatCard(title: "Due Today", value: stats.choresDueToday, color: .blue)
                    StatCard(title: "Overdue", value: stats.choresOverdue, color: .red)
                }
                .listRowInsets(EdgeInsets())
                .listRowBackground(Color.clear)
            }

            if !stats.choresDueTodayList.isEmpty {
                Section("Due Today") {
                    ForEach(stats.choresDueTodayList) { chore in
                        Text(chore.name)
                    }
                }
            }

            if !stats.choresOverdueList.isEmpty {
                Section("Overdue") {
                    ForEach(stats.choresOverdueList) { chore in
                        Text(chore.name)
                            .foregroundStyle(.red)
                    }
                }
            }
        }
    }
}

private struct StatCard: View {
    let title: String
    let value: Int
    let color: Color

    var body: some View {
        VStack {
            Text("\(value)")
                .font(.largeTitle.bold())
                .foregroundStyle(color)
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding()
        .background(color.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}
