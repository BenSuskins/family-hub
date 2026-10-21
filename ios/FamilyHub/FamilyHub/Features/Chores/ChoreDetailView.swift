import SwiftUI
import FamilyHubKit

struct ChoreDetailView: View {
    let definition: ChoreDefinition
    @Bindable var viewModel: ChoresViewModel
    let apiClient: any APIClientProtocol
    @Environment(\.dismiss) private var dismiss
    @State private var editing = false
    @State private var deleting = false
    @State private var deletePending = false
    @State private var history: [Chore] = []
    @State private var cursor: String?
    @State private var loadingHistory = false
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    @State private var historyError: APIError?

    private var current: ChoreDefinition { viewModel.definition(id: definition.id) ?? definition }

    var body: some View {
        List {
            if !current.chore.description.isEmpty {
                Section { Text(current.chore.description).foregroundStyle(AppTheme.secondary) }
                    .listRowBackground(Color.clear)
            }
            if let task = current.currentTask {
                Section("Current task") {
                    let layout = dynamicTypeSize.isAccessibilitySize
                        ? AnyLayout(VStackLayout(alignment: .leading, spacing: 12))
                        : AnyLayout(HStackLayout(spacing: 12))
                    layout {
                        if !dynamicTypeSize.isAccessibilitySize { WarmIcon(symbol: "person") }
                        VStack(alignment: .leading, spacing: 4) {
                            Text(viewModel.users[task.assignedToUserID ?? ""]?.name ?? "Unassigned")
                            Text((task.status == .overdue ? "Overdue" : "Due") + (task.formattedDueDate.map { " · " + $0 } ?? " date not set"))
                                .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        }
                        if task.status == .overdue {
                            Text("Overdue").font(.caption.weight(.semibold)).foregroundStyle(AppTheme.destructive)
                                .padding(8).background(AppTheme.destructive.opacity(0.1), in: Capsule())
                        }
                    }
                    Link(destination: URL(string: "familyhub://today")!) {
                        HStack {
                            VStack(alignment: .leading, spacing: 4) {
                                Text("View on Today")
                                Text("Complete or skip the current task.").font(.subheadline).foregroundStyle(AppTheme.secondary)
                            }
                            Spacer()
                            Image(systemName: "chevron.right")
                        }
                    }
                }.listRowBackground(AppTheme.surface)
            }
            Section("Repeat rule") {
                LabeledContent("Repeat", value: current.chore.recurrenceSummary)
                LabeledContent("People", value: eligiblePeople)
                LabeledContent("Rotation", value: "One person at a time")
                if current.chore.recurOnComplete { Text("The next date starts from completion or Skip.").foregroundStyle(AppTheme.secondary) }
                if let time = current.chore.dueTime { LabeledContent("Due time", value: time) }
                if let count = current.chore.recurrenceCount { LabeledContent("Task limit", value: String(count)) }
                if let end = current.chore.recurrenceUntil { LabeledContent("End date", value: end) }
                if let category = viewModel.categoryName(current.chore.categoryID) { LabeledContent("Category", value: category) }
            }.listRowBackground(AppTheme.surface)
            Section("History") {
                ForEach(history) { task in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(task.name)
                        Text(historySummary(task)).font(.subheadline).foregroundStyle(AppTheme.secondary)
                    }
                }
                if loadingHistory { ProgressView() }
                else if historyError != nil { Button("Retry history") { Task { await loadHistory(reset: history.isEmpty) } } }
                else if history.isEmpty {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("No task history")
                        Text("Completed and skipped tasks appear here.").font(.subheadline).foregroundStyle(AppTheme.secondary)
                    }
                }
                if cursor?.isEmpty == false {
                    Button("Load more history") { Task { await loadHistory(reset: false) } }.disabled(loadingHistory)
                }
            }.listRowBackground(AppTheme.surface)
            if current.deletedAt == nil {
                Section {
                    Button("Delete Chore", role: .destructive) { deleting = true }.foregroundStyle(AppTheme.destructive).disabled(deletePending)
                }.listRowBackground(AppTheme.surface)
            }
        }
        .appListStyle()
        .navigationTitle(current.chore.name)
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            if current.deletedAt == nil {
                Button("Edit") { editing = true }.accessibilityIdentifier("Edit Chore")
            }
        }
        .sheet(isPresented: $editing, onDismiss: {
            if viewModel.definition(id: definition.id) == nil { dismiss() }
            else { Task { await loadHistory(reset: true) } }
        }) {
            ChoreFormView(mode: .edit(current.chore), viewModel: viewModel)
        }
        .confirmationDialog("Delete \(current.chore.name)?", isPresented: $deleting, titleVisibility: .visible) {
            Button("Delete Chore", role: .destructive) {
                Task {
                    deletePending = true
                    if await viewModel.deleteChore(id: current.id) { dismiss() }
                    deletePending = false
                }
            }
        } message: { Text("This removes the chore from Today and keeps its history.") }
        .errorAlert($viewModel.actionError)
        .task { await loadHistory(reset: true) }
    }

    private var eligiblePeople: String {
        let identifiers = current.chore.eligibleAssignees
        return identifiers.isEmpty ? "Everyone" : identifiers.map { viewModel.users[$0]?.name ?? "Unknown person" }.joined(separator: ", ")
    }

    private func historySummary(_ task: Chore) -> String {
        let outcome: String
        switch task.status {
        case .completed: outcome = "Completed"
        case .skipped: outcome = "Skipped"
        case .superseded: outcome = "Superseded"
        case .pending, .overdue: outcome = current.deletedAt == nil ? "Outstanding" : "Cancelled"
        }
        var parts = [outcome]
        if let actor = task.resolvedByUserID, let person = viewModel.users[actor] { parts.append(person.name) }
        if let date = task.formattedDueDate { parts.append(date) }
        return parts.joined(separator: " · ")
    }

    private func loadHistory(reset: Bool) async {
        guard !loadingHistory else { return }
        loadingHistory = true
        historyError = nil
        defer { loadingHistory = false }
        do {
            let page = try await apiClient.fetchChoreHistory(definitionID: definition.id, cursor: reset ? nil : cursor)
            history = reset ? page.items : history + page.items
            cursor = page.nextCursor
        } catch { historyError = .from(error) }
    }
}
