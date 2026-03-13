// ios/FamilyHub/Features/Chores/ChoreDetailView.swift
import SwiftUI

struct ChoreDetailView: View {
    let chore: Chore
    let viewModel: ChoresViewModel

    @Environment(\.dismiss) private var dismiss
    @State private var isCompleting = false

    var body: some View {
        List {
            Section {
                LabeledContent("Status", value: chore.status.rawValue.capitalized)
                if let dueDate = chore.dueDate {
                    LabeledContent("Due", value: String(dueDate.prefix(10)))
                }
            }

            if !chore.description.isEmpty {
                Section("Description") {
                    Text(chore.description)
                }
            }

            if chore.status != .completed {
                Section {
                    Button {
                        Task {
                            isCompleting = true
                            await viewModel.complete(choreID: chore.id)
                            isCompleting = false
                            dismiss()
                        }
                    } label: {
                        HStack {
                            Spacer()
                            if isCompleting {
                                ProgressView()
                            } else {
                                Text("Mark Complete")
                                    .bold()
                            }
                            Spacer()
                        }
                    }
                    .disabled(isCompleting)
                }
            }
        }
        .navigationTitle(chore.name)
        .navigationBarTitleDisplayMode(.inline)
    }
}
