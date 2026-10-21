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

    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { definitions in
            List {
                Section {
                    Text("Set up chores and repeat rules.\nComplete today’s chores on Today.")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                }.listRowBackground(Color.clear)
                Section {
                    Link(destination: URL(string: "familyhub://today")!) {
                        HStack {
                            VStack(alignment: .leading, spacing: 4) {
                                Text("Today’s chores").foregroundStyle(AppTheme.ink)
                                let nextDay = Calendar.current.date(byAdding: .day, value: 1, to: Calendar.current.startOfDay(for: Date())) ?? Date()
                                let count = definitions.compactMap(\.currentTask).filter { task in
                                    task.status == .overdue || task.dueDate.flatMap(APIDate.parse).map { $0 < nextDay } == true
                                }.count
                                Text(count == 1 ? "1 chore needs attention" : "\(count) chores need attention")
                                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                            }
                            Spacer()
                            Image(systemName: "chevron.right").foregroundStyle(AppTheme.secondary)
                        }
                    }.accessibilityLabel("Today’s chores")
                }.listRowBackground(AppTheme.surface)
                Section("Chore rules") {
                    if definitions.isEmpty {
                        ContentUnavailableView("No chores configured", systemImage: "checklist", description: Text("Add a chore and choose when it repeats."))
                    }
                    ForEach(definitions) { definition in
                        NavigationLink {
                            ChoreDetailView(definition: definition, viewModel: viewModel, apiClient: apiClient)
                        } label: {
                            VStack(alignment: .leading, spacing: 5) {
                                Text(definition.chore.name).font(.body.weight(.medium))
                                Text(ruleSummary(definition.chore) + " · " + assigneeSummary(definition.chore)).font(.subheadline).foregroundStyle(AppTheme.secondary)
                                if !definition.active {
                                    Text(definition.deletedAt == nil ? "Inactive" : "Deleted").font(.caption).foregroundStyle(AppTheme.secondary)
                                }
                            }
                            .padding(.vertical, 5)
                        }
                    }
                }.listRowBackground(AppTheme.surface)
                Section {
                    Toggle("Show inactive chores", isOn: $viewModel.showInactive)
                }.listRowBackground(AppTheme.surface)
            }
            .appListStyle()
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
        .onReceive(NotificationCenter.default.publisher(for: .familyHubIntentChanged)) { _ in
            Task { await viewModel.load(refresh: true) }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { Task { await viewModel.load(refresh: true) } }
        }
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
