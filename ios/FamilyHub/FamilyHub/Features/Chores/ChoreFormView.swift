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

    // Details
    @State private var name = ""
    @State private var description = ""
    @State private var categoryID: String = ""   // "" == No Category
    @State private var selectedAssignees: Set<String> = []

    // Schedule
    @State private var hasDueDate = false
    @State private var dueDate = Date()
    @State private var hasDueTime = false
    @State private var dueTime = Date()

    // Recurrence
    @State private var recurrenceType = "none"
    @State private var interval = 1
    @State private var weeklyDays: Set<String> = []
    @State private var dayOfMonth = 1
    @State private var customUnit = "days"

    // End conditions
    @State private var hasEndDate = false
    @State private var endDate = Date()
    @State private var hasCount = false
    @State private var occurrenceCount = 10

    @State private var recurOnComplete = false
    @State private var isSaving = false
    @State private var isDeleting = false
    @State private var showDeleteConfirm = false

    private let recurrenceOptions = ["none", "daily", "weekly", "monthly", "custom"]
    private let customUnits = ["days", "weeks", "months"]

    private var isEditing: Bool {
        if case .edit = mode { return true }
        return false
    }

    private var showsInterval: Bool {
        recurrenceType == "weekly" || recurrenceType == "monthly" || recurrenceType == "custom"
    }

    private var intervalUnitLabel: String {
        switch recurrenceType {
        case "weekly":  return interval == 1 ? "week" : "weeks"
        case "monthly": return interval == 1 ? "month" : "months"
        case "custom":  return customUnit
        default:        return ""
        }
    }

    var body: some View {
        NavigationStack {
            Form {
                detailsSection
                categorySection
                assigneesSection
                scheduleSection
                recurrenceSection
                if recurrenceType != "none" {
                    endConditionsSection
                }
                if isEditing {
                    deleteSection
                }
            }
            .navigationTitle(isEditing ? "Edit Chore" : "New Chore")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving {
                        ProgressView()
                    } else {
                        Button("Save") {
                            Task { await save() }
                        }
                        .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty || isDeleting)
                    }
                }
            }
            .onAppear {
                viewModel.actionError = nil
                populate()
            }
            .errorAlert($viewModel.actionError)
            .confirmationDialog(
                "Delete Chore?",
                isPresented: $showDeleteConfirm,
                titleVisibility: .visible
            ) {
                Button("Delete", role: .destructive) {
                    Task { await delete() }
                }
            } message: {
                Text("This permanently deletes \"\(name)\"\(isRecurringSeries ? " and its future occurrences" : "").")
            }
        }
    }

    private var isRecurringSeries: Bool {
        if case .edit(let chore) = mode { return chore.isRecurring }
        return false
    }

    private var deleteSection: some View {
        Section {
            Button(role: .destructive) {
                showDeleteConfirm = true
            } label: {
                HStack {
                    Spacer()
                    if isDeleting {
                        ProgressView()
                    } else {
                        Label("Delete Chore", systemImage: "trash")
                    }
                    Spacer()
                }
            }
            .disabled(isSaving || isDeleting)
        }
    }

    // MARK: - Sections

    private var detailsSection: some View {
        Section("Details") {
            TextField("Name", text: $name)
            TextField("Description", text: $description, axis: .vertical)
                .lineLimit(3...6)
        }
    }

    @ViewBuilder
    private var categorySection: some View {
        if !viewModel.categories.isEmpty {
            Section("Category") {
                Picker("Category", selection: $categoryID) {
                    Text("No Category").tag("")
                    ForEach(viewModel.categories) { category in
                        Text(category.name).tag(category.id)
                    }
                }
            }
        }
    }

    private var assigneesSection: some View {
        Section {
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
        } header: {
            Text("Assignees")
        } footer: {
            Text("Selected members rotate the chore between them. If none are selected, everyone is eligible.")
        }
    }

    private var scheduleSection: some View {
        Section("Schedule") {
            Toggle("Has Due Date", isOn: $hasDueDate)
            if hasDueDate {
                DatePicker("Due Date", selection: $dueDate, displayedComponents: .date)
            }
            Toggle("Has Due Time", isOn: $hasDueTime)
            if hasDueTime {
                DatePicker("Due Time", selection: $dueTime, displayedComponents: .hourAndMinute)
            }
        }
    }

    @ViewBuilder
    private var recurrenceSection: some View {
        Section("Recurrence") {
            Picker("Repeats", selection: $recurrenceType) {
                ForEach(recurrenceOptions, id: \.self) { option in
                    Text(option.capitalized).tag(option)
                }
            }

            if showsInterval {
                Stepper(value: $interval, in: 1...52) {
                    Text("Every \(interval) \(intervalUnitLabel)")
                }
            }

            if recurrenceType == "custom" {
                Picker("Unit", selection: $customUnit) {
                    ForEach(customUnits, id: \.self) { unit in
                        Text(unit.capitalized).tag(unit)
                    }
                }
            }

            if recurrenceType == "weekly" {
                weekdayPicker
            }

            if recurrenceType == "monthly" {
                Picker("Day of Month", selection: $dayOfMonth) {
                    ForEach(1...31, id: \.self) { day in
                        Text("\(day)").tag(day)
                    }
                }
            }

            if recurrenceType != "none" {
                Toggle("Recur After Completion", isOn: $recurOnComplete)
            }
        }
    }

    private var weekdayPicker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("On Days")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            HStack(spacing: 6) {
                ForEach(Chore.weekdayKeys, id: \.self) { key in
                    let label = Chore.dayShortLabels[key] ?? key.prefix(1).uppercased()
                    let isOn = weeklyDays.contains(key)
                    Button {
                        if isOn { weeklyDays.remove(key) } else { weeklyDays.insert(key) }
                    } label: {
                        Text(label)
                            .font(.system(size: 12, weight: .semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(isOn ? Color.accentColor : Color(UIColor.secondarySystemFill),
                                        in: RoundedRectangle(cornerRadius: 8))
                            .foregroundStyle(isOn ? .white : .primary)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private var endConditionsSection: some View {
        Section {
            Toggle("Ends On Date", isOn: $hasEndDate)
            if hasEndDate {
                DatePicker("End Date", selection: $endDate, displayedComponents: .date)
            }
            Toggle("Limit Occurrences", isOn: $hasCount)
            if hasCount {
                Stepper(value: $occurrenceCount, in: 1...365) {
                    Text("After \(occurrenceCount) times")
                }
            }
        } header: {
            Text("End Conditions")
        } footer: {
            Text("Optionally stop the series on a date or after a number of occurrences.")
        }
    }

    // MARK: - Populate / Save

    private func populate() {
        guard case .edit(let chore) = mode else { return }
        name = chore.name
        description = chore.description
        categoryID = chore.categoryID ?? ""

        if !chore.eligibleAssignees.isEmpty {
            selectedAssignees = Set(chore.eligibleAssignees)
        } else if let uid = chore.assignedToUserID {
            selectedAssignees = [uid]
        }

        if let d = parseDate(chore.dueDate) {
            hasDueDate = true
            dueDate = d
        }
        if let t = chore.dueTime, let parsed = parseTime(t) {
            hasDueTime = true
            dueTime = parsed
        }

        recurrenceType = chore.isRecurring ? chore.recurrenceType : "none"
        recurOnComplete = chore.recurOnComplete

        let config = chore.recurrenceConfig
        interval = max(config.interval, 1)
        customUnit = config.unit.isEmpty ? "days" : config.unit
        weeklyDays = Set(config.days.map { $0.lowercased() })
        dayOfMonth = config.dayOfMonth >= 1 ? config.dayOfMonth : 1

        if let until = parseDate(chore.recurrenceUntil) {
            hasEndDate = true
            endDate = until
        }
        if let count = chore.recurrenceCount, count > 0 {
            hasCount = true
            occurrenceCount = count
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }

        let dueDateStr: String? = hasDueDate ? formatDate(dueDate) : nil
        let dueTimeStr: String? = hasDueTime ? formatTime(dueTime) : nil
        let recurring = recurrenceType != "none"

        let request = ChoreRequest(
            name: name.trimmingCharacters(in: .whitespaces),
            description: description.trimmingCharacters(in: .whitespaces),
            assignees: Array(selectedAssignees),
            dueDate: dueDateStr,
            recurrenceType: recurring ? recurrenceType : nil,
            categoryId: categoryID.isEmpty ? nil : categoryID,
            dueTime: dueTimeStr,
            recurrenceInterval: showsInterval ? interval : nil,
            recurrenceDays: recurrenceType == "weekly" && !weeklyDays.isEmpty
                ? Chore.weekdayKeys.filter { weeklyDays.contains($0) } : nil,
            recurrenceDayOfMonth: recurrenceType == "monthly" ? dayOfMonth : nil,
            recurrenceUnit: recurrenceType == "custom" ? customUnit : nil,
            recurrenceUntil: recurring && hasEndDate ? formatDate(endDate) : nil,
            recurrenceCount: recurring && hasCount ? occurrenceCount : nil,
            recurOnComplete: recurring && recurOnComplete
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

    private func delete() async {
        guard case .edit(let chore) = mode else { return }
        isDeleting = true
        defer { isDeleting = false }
        if await viewModel.deleteChore(id: chore.id) {
            dismiss()
        }
        // On failure, viewModel.actionError is set and surfaced via errorAlert.
    }

    // MARK: - Date helpers

    private func parseDate(_ value: String?) -> Date? { APIDate.parse(value) }

    private func formatDate(_ date: Date) -> String { APIDate.dayString(date) }

    private func parseTime(_ value: String) -> Date? {
        APIDate.time.date(from: String(value.prefix(5)))
    }

    private func formatTime(_ date: Date) -> String {
        APIDate.time.string(from: date)
    }
}
