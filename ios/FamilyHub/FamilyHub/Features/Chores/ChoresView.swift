// ios/FamilyHub/Features/Chores/ChoresView.swift
import SwiftUI

struct ChoresView: View {
    @State private var viewModel: ChoresViewModel

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                switch viewModel.state {
                case .idle, .loading:
                    ProgressView()
                case .loaded:
                    choresList
                case .failed(let error):
                    ContentUnavailableView(
                        "Failed to load",
                        systemImage: "exclamationmark.triangle",
                        description: Text(error.localizedDescription)
                    )
                }
            }
            .navigationTitle("Chores")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Refresh") { Task { await viewModel.load() } }
                }
            }
            .alert("Error", isPresented: showError) {
                Button("OK") { viewModel.errorMessage = nil }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
        .task { await viewModel.load() }
    }

    private var showError: Binding<Bool> {
        Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )
    }

    private var choresList: some View {
        List {
            if !viewModel.pendingChores.isEmpty {
                Section("Pending") {
                    ForEach(viewModel.pendingChores) { chore in
                        NavigationLink {
                            ChoreDetailView(chore: chore, viewModel: viewModel)
                        } label: {
                            ChoreRow(chore: chore)
                        }
                    }
                }
            }
            if !viewModel.completedChores.isEmpty {
                Section("Completed") {
                    ForEach(viewModel.completedChores) { chore in
                        ChoreRow(chore: chore)
                    }
                }
            }
        }
    }
}

private struct ChoreRow: View {
    let chore: Chore

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(chore.name)
            if let dueDate = chore.dueDate {
                Text(dueDate.prefix(10)) // Show YYYY-MM-DD portion
                    .font(.caption)
                    .foregroundStyle(chore.status == .overdue ? .red : .secondary)
            }
        }
    }
}
