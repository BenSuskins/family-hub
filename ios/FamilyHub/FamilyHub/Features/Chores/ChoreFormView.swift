import SwiftUI

struct ChoreFormView: View {
    enum FormMode {
        case create
        case edit(Chore)
    }

    let mode: FormMode
    @Bindable var viewModel: ChoresViewModel
    var onSave: ((Chore) -> Void)?

    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var description = ""
    @State private var selectedAssignees: Set<String> = []
    @State private var hasDueDate = false
    @State private var dueDate = Date()
    @State private var recurrenceType = "none"
    @State private var isSaving = false

    private let recurrenceOptions = ["none", "daily", "weekly", "monthly"]

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Details") {
                    TextField("Name", text: $name)
                    TextField("Description", text: $description, axis: .vertical)
                        .lineLimit(3...6)
                }

                Section("Assignees") {
                    if viewModel.users.isEmpty {
                        Text("No users available")
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(viewModel.users.values).sorted(by: { $0.name < $1.name })) { user in
                            Button {
                                if selectedAssignees.contains(user.id) {
                                    selectedAssignees.remove(user.id)
                                } else {
                                    selectedAssignees.insert(user.id)
                                }
                            } label: {
                                HStack {
                                    Text(user.name)
                                        .foregroundStyle(.primary)
                                    Spacer()
                                    if selectedAssignees.contains(user.id) {
                                        Image(systemName: "checkmark")
                                            .foregroundStyle(.blue)
                                    }
                                }
                            }
                        }
                    }
                }

                Section("Schedule") {
                    Toggle("Has Due Date", isOn: $hasDueDate)
                    if hasDueDate {
                        DatePicker("Due Date", selection: $dueDate, displayedComponents: .date)
                    }
                    Picker("Recurrence", selection: $recurrenceType) {
                        ForEach(recurrenceOptions, id: \.self) { option in
                            Text(option.capitalized).tag(option)
                        }
                    }
                }
            }
            .navigationTitle(isEditing ? "Edit Chore" : "New Chore")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await save() }
                    }
                    .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty || isSaving)
                }
            }
            .onAppear { populate() }
        }
    }

    private func populate() {
        guard case .edit(let chore) = mode else { return }
        name = chore.name
        description = chore.description
        if let uid = chore.assignedToUserID {
            selectedAssignees = [uid]
        }
        if let dateStr = chore.dueDate {
            let iso = ISO8601DateFormatter()
            let fmt = DateFormatter()
            fmt.dateFormat = "yyyy-MM-dd"
            fmt.locale = Locale(identifier: "en_US_POSIX")
            if let d = iso.date(from: dateStr) ?? fmt.date(from: String(dateStr.prefix(10))) {
                hasDueDate = true
                dueDate = d
            }
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }

        let dueDateStr: String? = hasDueDate ? {
            let fmt = DateFormatter()
            fmt.dateFormat = "yyyy-MM-dd"
            return fmt.string(from: dueDate)
        }() : nil

        let request = ChoreRequest(
            name: name.trimmingCharacters(in: .whitespaces),
            description: description.trimmingCharacters(in: .whitespaces),
            assignees: Array(selectedAssignees),
            dueDate: dueDateStr,
            recurrenceType: recurrenceType == "none" ? nil : recurrenceType
        )

        if case .edit(let chore) = mode {
            if let updated = await viewModel.updateChore(id: chore.id, request) {
                onSave?(updated)
                dismiss()
            }
        } else {
            if let created = await viewModel.createChore(request) {
                onSave?(created)
                dismiss()
            }
        }
    }
}
