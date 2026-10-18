import SwiftUI
import FamilyHubKit

struct ChoresListView: View {
    @State private var viewModel: ChoresViewModel
    @State private var creating = false
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { definitions in
            List {
                Section {
                    Toggle("Show inactive chores", isOn: $viewModel.showInactive)
                }
                Section("Configured chores") {
                    if definitions.isEmpty {
                        ContentUnavailableView("No chores configured", systemImage: "checklist", description: Text("Add a chore and choose when it repeats."))
                    }
                    ForEach(definitions) { definition in
                        NavigationLink {
                            ChoreDetailView(definition: definition, viewModel: viewModel, apiClient: apiClient)
                        } label: {
                            VStack(alignment: .leading, spacing: 5) {
                                Text(definition.chore.name).font(.body.weight(.medium))
                                Text(ruleSummary(definition.chore)).font(.subheadline).foregroundStyle(.secondary)
                                Text(assigneeSummary(definition.chore)).font(.caption).foregroundStyle(.secondary)
                                if !definition.active {
                                    Text(definition.deletedAt == nil ? "Inactive" : "Deleted").font(.caption).foregroundStyle(.secondary)
                                }
                            }
                            .padding(.vertical, 5)
                        }
                    }
                }
            }
            .refreshable { await viewModel.load() }
        }
        .navigationTitle("Chores")
        .toolbar {
            Button("New Chore", systemImage: "plus") { creating = true }
        }
        .sheet(isPresented: $creating) {
            ChoreFormView(mode: .create, viewModel: viewModel)
        }
        .errorAlert($viewModel.actionError)
        .task(id: viewModel.showInactive) { await viewModel.load() }
        .onReceive(NotificationCenter.default.publisher(for: .familyHubChoresChanged)) { _ in
            Task { await viewModel.load(refresh: true) }
        }
    }

    private func ruleSummary(_ chore: Chore) -> String {
        var parts = [chore.recurrenceSummary]
        if chore.recurOnComplete { parts.append("after completion") }
        if let time = chore.dueTime { parts.append(time) }
        return parts.joined(separator: " · ")
    }

    private func assigneeSummary(_ chore: Chore) -> String {
        if chore.eligibleAssignees.isEmpty { return "Everyone" }
        return chore.eligibleAssignees.map { viewModel.users[$0]?.name ?? "Unknown person" }.joined(separator: ", ")
    }
}
