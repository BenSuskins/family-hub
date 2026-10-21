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
        case create, created(CreatedToken), manage(APIToken)
        var id: String {
            switch self {
            case .create: return "create"
            case .created(let token): return "created-" + token.id
            case .manage(let token): return "manage-" + token.id
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
                Text("Connect trusted apps to Family Hub.")
                    .foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)
            if viewModel.tokens.isEmpty && !viewModel.isLoading {
                Section {
                    ContentUnavailableView("No Tokens", systemImage: "key", description: Text("Tap + to create a token."))
                }.listRowBackground(AppTheme.surface)
            }
            ForEach(viewModel.tokens) { token in
                Button { activeSheet = .manage(token) } label: {
                    HStack(spacing: 12) {
                        WarmIcon(symbol: "key")
                        VStack(alignment: .leading, spacing: 4) {
                            Text(token.name).foregroundStyle(AppTheme.ink)
                            Text("Created " + token.createdAt.formatted(date: .abbreviated, time: .omitted))
                                .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        }
                        Spacer()
                        Image(systemName: "chevron.right").foregroundStyle(AppTheme.secondary)
                    }
                }.buttonStyle(.plain)
            }
            .onDelete { offsets in
                pendingDeletion = offsets
            }
            .listRowBackground(AppTheme.surface)
            Section {
                Button("New token") { activeSheet = .create }.buttonStyle(AppSecondaryButtonStyle())
            } footer: {
                Text("Treat tokens like passwords. Delete a token to stop that app’s access.")
            }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
        }
        .appListStyle()
        .navigationTitle("API tokens")
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { activeSheet = .create } label: {
                    Image(systemName: "plus")
                }
                .accessibilityLabel("New Token")
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
                NameEntrySheet(title: "New token", name: $newTokenName, fieldTitle: "Token name", actionTitle: "Create token", explanation: "The token appears after you select Create.", fieldHelp: "Use a name that identifies the app or device.", primaryActionInContent: true) { name in
                    viewModel.actionError = nil
                    await viewModel.create(name: name)
                    let error = viewModel.newlyCreatedToken == nil ? viewModel.actionError : nil
                    viewModel.actionError = nil
                    return error
                }
            case .created(let token): CreatedTokenSheet(token: token)
            case .manage(let token): TokenDetailsSheet(token: token, model: viewModel)
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
            ScrollView {
                VStack(spacing: 24) {
                    Image(systemName: "checkmark").font(.largeTitle).foregroundStyle(AppTheme.accent).accessibilityHidden(true)
                    Text("Save your token").font(.system(.largeTitle, design: .rounded, weight: .bold))
                        .accessibilityAddTraits(.isHeader)
                    Text(token.name).foregroundStyle(AppTheme.secondary)
                    Text(token.plaintext)
                        .font(.system(.body, design: .monospaced)).textSelection(.enabled)
                        .fixedSize(horizontal: false, vertical: true)
                        .frame(maxWidth: .infinity, alignment: .leading).padding(16).groupedCard()
                    Text("Copy this token now. You cannot view it again.")
                        .font(.subheadline).frame(maxWidth: .infinity, alignment: .leading)
                        .padding(16).background(AppTheme.sage, in: RoundedRectangle(cornerRadius: 20))
                    Button(copied ? "Copied" : "Copy token") {
                        UIPasteboard.general.string = token.plaintext
                        copied = true
                        UIAccessibility.post(notification: .announcement, argument: "Copied")
                        Task {
                            try? await Task.sleep(for: .seconds(2))
                            copied = false
                        }
                    }
                    .buttonStyle(AppPrimaryButtonStyle())
                    .accessibilityIdentifier("Copy Token")
                }
                .frame(maxWidth: 520)
                .padding(20).padding(.top, 24)
                .frame(maxWidth: .infinity)
            }
            .background(AppTheme.background)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }.buttonStyle(AppCommitButtonStyle())
                }
            }
        }
    }

}

private struct TokenDetailsSheet: View {
    let token: APIToken
    @Bindable var model: APITokensViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var confirming = false
    @State private var isDeleting = false

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    LabeledContent("Name", value: token.name)
                    LabeledContent("Created", value: token.createdAt.formatted(date: .abbreviated, time: .omitted))
                    Text("The token value is available only when you create it.").font(.subheadline).foregroundStyle(AppTheme.secondary)
                }.listRowBackground(AppTheme.surface)
                Section {
                    Button("Delete token", role: .destructive) { confirming = true }.foregroundStyle(AppTheme.destructive).disabled(isDeleting)
                } footer: { Text("Delete this token to stop the app’s access.") }
                .listRowBackground(AppTheme.surface)
            }
            .appListStyle()
            .navigationTitle(token.name)
            .toolbar { Button("Done") { dismiss() }.disabled(isDeleting) }
            .confirmationDialog("Delete \(token.name)?", isPresented: $confirming, titleVisibility: .visible) {
                Button("Delete token", role: .destructive) {
                    Task {
                        guard let index = model.tokens.firstIndex(where: { $0.id == token.id }) else { dismiss(); return }
                        isDeleting = true
                        await model.delete(at: IndexSet([index]))
                        isDeleting = false
                        if !model.tokens.contains(where: { $0.id == token.id }) { dismiss() }
                    }
                }
            }
            .errorAlert($model.actionError)
        }
    }
}
