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
    @State private var historyError: APIError?

    private var current: ChoreDefinition { viewModel.definition(id: definition.id) ?? definition }

    var body: some View {
        List {
            Section("Chore setup") {
                if !current.chore.description.isEmpty { Text(current.chore.description) }
                LabeledContent("Repeat", value: current.chore.recurrenceSummary)
                if current.chore.recurOnComplete { Text("The next date starts from completion or Skip.").foregroundStyle(.secondary) }
                LabeledContent("Eligible people", value: eligiblePeople)
                if let time = current.chore.dueTime { LabeledContent("Due time", value: time) }
                if let count = current.chore.recurrenceCount { LabeledContent("Task limit", value: String(count)) }
                if let end = current.chore.recurrenceUntil { LabeledContent("End date", value: end) }
                if let category = viewModel.categoryName(current.chore.categoryID) { LabeledContent("Category", value: category) }
            }
            if let task = current.currentTask {
                Section("Current task") {
                    LabeledContent("Assignee", value: viewModel.users[task.assignedToUserID ?? ""]?.name ?? "Unassigned")
                    if let date = task.formattedDueDate { LabeledContent("Due", value: date) }
                    Text("Complete or skip this task on Home.").font(.footnote).foregroundStyle(.secondary)
                }
            }
            Section("History") {
                ForEach(history) { task in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(task.name)
                        Text(historySummary(task)).font(.subheadline).foregroundStyle(.secondary)
                    }
                }
                if loadingHistory { ProgressView() }
                else if historyError != nil { Button("Retry history") { Task { await loadHistory(reset: history.isEmpty) } } }
                else if history.isEmpty { Text("No task history").foregroundStyle(.secondary) }
                if cursor?.isEmpty == false {
                    Button("Load more history") { Task { await loadHistory(reset: false) } }.disabled(loadingHistory)
                }
            }
            if current.deletedAt == nil {
                Section {
                    Button("Edit Chore") { editing = true }
                    Button("Delete Chore", role: .destructive) { deleting = true }.disabled(deletePending)
                }
            }
        }
        .navigationTitle(current.chore.name)
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $editing, onDismiss: {
            if viewModel.definition(id: definition.id) == nil { dismiss() }
            else { Task { await loadHistory(reset: true) } }
        }) {
            ChoreFormView(mode: .edit(current.chore), viewModel: viewModel)
        }
        .confirmationDialog("Delete this chore?", isPresented: $deleting, titleVisibility: .visible) {
            Button("Delete Chore", role: .destructive) {
                Task {
                    deletePending = true
                    if await viewModel.deleteChore(id: current.id) { dismiss() }
                    deletePending = false
                }
            }
        } message: { Text("This removes the chore from Home and keeps its history.") }
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
