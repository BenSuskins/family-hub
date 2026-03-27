import SwiftUI

struct ChoreDetailView: View {
    let chore: Chore
    let viewModel: ChoresViewModel

    @Environment(\.dismiss) private var dismiss
    @State private var isCompleting = false
    @State private var completionError: String?

    var body: some View {
        List {
            Section {
                HStack(spacing: 12) {
                    UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(viewModel.users[chore.assignedToUserID ?? ""]?.name ?? "Unassigned")
                            .font(.body.weight(.medium))
                        if let badge = chore.badge {
                            Text(badge.label)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(badge.color)
                        }
                    }
                    Spacer()
                    if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.subheadline)
                            .foregroundStyle(chore.status == .overdue ? .red : .secondary)
                    }
                }
            }

            if !chore.description.isEmpty {
                Section("Description") {
                    Text(chore.description)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }

            if chore.status != .completed {
                Section {
                    Button {
                        Task {
                            isCompleting = true
                            completionError = nil
                            let success = await viewModel.complete(choreID: chore.id)
                            isCompleting = false
                            if success { dismiss() } else { completionError = viewModel.errorMessage }
                        }
                    } label: {
                        HStack {
                            Spacer()
                            if isCompleting {
                                ProgressView()
                            } else {
                                Label("Mark Complete", systemImage: "checkmark")
                            }
                            Spacer()
                        }
                    }
                    .tint(.green)
                    .disabled(isCompleting)

                    if let completionError {
                        Text(completionError)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(chore.name)
        .navigationBarTitleDisplayMode(.inline)
    }
}
