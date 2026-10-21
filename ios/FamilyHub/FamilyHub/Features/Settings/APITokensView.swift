import SwiftUI
import FamilyHubKit

@Observable
final class APITokensViewModel {
    var tokens: [APIToken] = []
    var isLoading = false
    var actionError: APIError?
    var newlyCreatedToken: CreatedToken?

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            tokens = try await apiClient.fetchTokens()
        } catch {
            actionError = .from(error)
        }
    }

    func create(name: String) async {
        do {
            let created = try await apiClient.createToken(name: name)
            newlyCreatedToken = created
            await load()
        } catch {
            actionError = .from(error)
        }
    }

    func delete(at offsets: IndexSet) async {
        let toDelete = offsets.map { tokens[$0] }
        for token in toDelete {
            do {
                try await apiClient.deleteToken(id: token.id)
                tokens.removeAll { $0.id == token.id }
            } catch {
                actionError = .from(error)
            }
        }
    }
}

struct APITokensView: View {
    let apiClient: any APIClientProtocol

    @State private var viewModel: APITokensViewModel
    @State private var pendingDeletion: IndexSet?
    @State private var activeSheet: TokenSheet?

    private enum TokenSheet: Identifiable {
        case create, created(CreatedToken)
        var id: String {
            switch self {
            case .create: return "create"
            case .created(let token): return token.id
            }
        }
    }
    @State private var newTokenName = ""

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        self._viewModel = State(initialValue: APITokensViewModel(apiClient: apiClient))
    }

    private var deletionName: String {
        guard let index = pendingDeletion?.first, viewModel.tokens.indices.contains(index) else { return "token" }
        return viewModel.tokens[index].name
    }

    var body: some View {
        List {
            Section {
                Text("Tokens connect other apps to Family Hub. Treat them like passwords.")
                    .foregroundStyle(AppTheme.secondary)
            }.listRowBackground(AppTheme.surface)
            if viewModel.tokens.isEmpty && !viewModel.isLoading {
                Section {
                    ContentUnavailableView("No Tokens", systemImage: "key", description: Text("Tap + to create a token."))
                }.listRowBackground(AppTheme.surface)
            }
            ForEach(viewModel.tokens) { token in
                VStack(alignment: .leading, spacing: 2) {
                    Text(token.name).font(.body)
                    Text(token.createdAt.formatted(date: .abbreviated, time: .omitted))
                        .font(.caption)
                        .foregroundStyle(AppTheme.secondary)
                }
            }
            .onDelete { offsets in
                pendingDeletion = offsets
            }
            .listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle("API Tokens")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { activeSheet = .create } label: {
                    Image(systemName: "plus")
                }
                .accessibilityLabel("New Token")
            }
            ToolbarItem(placement: .navigationBarLeading) {
                EditButton()
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.tokens.isEmpty {
                ProgressView()
            }
        }
        .sheet(item: $activeSheet, onDismiss: {
            if let token = viewModel.newlyCreatedToken {
                viewModel.newlyCreatedToken = nil
                activeSheet = .created(token)
            }
        }) { sheet in
            switch sheet {
            case .create:
                NameEntrySheet(title: "New Token", name: $newTokenName, fieldTitle: "Token name", actionTitle: "Create token", explanation: "Copy the token after creation. The app shows it once.") { name in
                    viewModel.actionError = nil
                    await viewModel.create(name: name)
                    let error = viewModel.newlyCreatedToken == nil ? viewModel.actionError : nil
                    viewModel.actionError = nil
                    return error
                }
            case .created(let token): CreatedTokenSheet(token: token)
            }
        }
        .confirmationDialog("Delete \(deletionName)?", isPresented: Binding(
            get: { pendingDeletion != nil }, set: { if !$0 { pendingDeletion = nil } }
        ), titleVisibility: .visible) {
            if let offsets = pendingDeletion {
                Button("Delete", role: .destructive) { Task { await viewModel.delete(at: offsets) } }
            }
        }
        .errorAlert($viewModel.actionError)
        .task { await viewModel.load() }
    }
}

private struct CreatedTokenSheet: View {
    let token: CreatedToken
    @State private var copied = false
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Label("Token Created", systemImage: "checkmark.circle")
                    Text("Copy this token now. The app shows it once.")
                        .foregroundStyle(AppTheme.secondary)
                }.listRowBackground(AppTheme.surface)
                Section(token.name) {
                    Text(token.plaintext)
                        .font(.system(.body, design: .monospaced))
                        .textSelection(.enabled)
                    Button(copied ? "Copied" : "Copy Token", systemImage: "doc.on.doc") {
                        UIPasteboard.general.string = token.plaintext
                        copied = true
                        UIAccessibility.post(notification: .announcement, argument: "Copied")
                        Task {
                            try? await Task.sleep(for: .seconds(2))
                            copied = false
                        }
                    }
                }.listRowBackground(AppTheme.surface)
            }
            .appListStyle()
            .navigationTitle("New Token")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) { Button("Done") { dismiss() } }
            }
        }
    }
}
