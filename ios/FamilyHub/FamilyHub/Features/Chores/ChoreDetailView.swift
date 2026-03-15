import SwiftUI

struct ChoreDetailView: View {
    let chore: Chore
    let viewModel: ChoresViewModel

    @Environment(\.dismiss) private var dismiss
    @State private var isCompleting = false
    @State private var completionError: String?

    var body: some View {
        ZStack {
            Theme.background.ignoresSafeArea()
            List {
                // Assignee row
                Section {
                    HStack(spacing: 12) {
                        UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(viewModel.users[chore.assignedToUserID ?? ""]?.name ?? "Unassigned")
                                .font(.system(size: 14, weight: .medium))
                                .foregroundStyle(Theme.textPrimary)
                            if let badge = chore.badgeVariant {
                                StatusBadge(variant: badge)
                            }
                        }
                        Spacer()
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 13))
                                .foregroundStyle(chore.status == .overdue ? Theme.statusRed : Theme.textSecondary)
                        }
                    }
                    .listRowBackground(Theme.surface)
                }

                // Description
                if !chore.description.isEmpty {
                    Section("Description") {
                        Text(chore.description)
                            .font(.system(size: 14))
                            .foregroundStyle(Theme.textSecondary)
                            .listRowBackground(Theme.surface)
                    }
                }

                // Mark complete button
                if chore.status != .completed {
                    Section {
                        VStack(spacing: 8) {
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
                                        ProgressView().tint(Theme.statusGreen)
                                    } else {
                                        Image(systemName: "checkmark")
                                            .foregroundStyle(Theme.statusGreen)
                                        Text("Mark Complete")
                                            .font(.system(size: 15, weight: .semibold))
                                            .foregroundStyle(Theme.statusGreen)
                                    }
                                    Spacer()
                                }
                                .padding(.vertical, 14)
                                .background(Theme.doneButtonBg)
                                .overlay(
                                    RoundedRectangle(cornerRadius: 12)
                                        .stroke(Theme.doneButtonBorder, lineWidth: 1)
                                )
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                            .disabled(isCompleting)

                            if let completionError {
                                Text(completionError)
                                    .font(.system(size: 12))
                                    .foregroundStyle(Theme.statusRed)
                                    .multilineTextAlignment(.center)
                            }
                        }
                        .listRowBackground(Color.clear)
                        .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                    }
                }
            }
            .listStyle(.insetGrouped)
            .scrollContentBackground(.hidden)
        }
        .navigationTitle(chore.name)
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(Theme.background, for: .navigationBar)
    }
}
