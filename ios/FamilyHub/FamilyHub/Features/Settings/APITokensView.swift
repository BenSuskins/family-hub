import SwiftUI

@Observable
final class APITokensViewModel {
    var tokens: [APIToken] = []
    var isLoading = false
    var errorMessage: String?
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
            errorMessage = "Failed to load tokens"
        }
    }

    func create(name: String) async {
        do {
            let created = try await apiClient.createToken(name: name)
            newlyCreatedToken = created
            await load()
        } catch {
            errorMessage = "Failed to create token"
        }
    }

    func delete(at offsets: IndexSet) async {
        let toDelete = offsets.map { tokens[$0] }
        tokens.remove(atOffsets: offsets)
        for token in toDelete {
            do {
                try await apiClient.deleteToken(id: token.id)
            } catch {
                errorMessage = "Failed to revoke token"
            }
        }
    }
}

struct APITokensView: View {
    let apiClient: any APIClientProtocol

    @State private var viewModel: APITokensViewModel
    @State private var showingAddSheet = false
    @State private var newTokenName = ""

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        self._viewModel = State(initialValue: APITokensViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            ForEach(viewModel.tokens) { token in
                VStack(alignment: .leading, spacing: 2) {
                    Text(token.name).font(.body)
                    Text(token.createdAt.formatted(date: .abbreviated, time: .omitted))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .onDelete { offsets in
                Task { await viewModel.delete(at: offsets) }
            }
        }
        .navigationTitle("API Tokens")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { showingAddSheet = true } label: {
                    Image(systemName: "plus")
                }
            }
            ToolbarItem(placement: .navigationBarLeading) {
                EditButton()
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.tokens.isEmpty {
                ProgressView()
            } else if viewModel.tokens.isEmpty && !viewModel.isLoading {
                ContentUnavailableView("No Tokens", systemImage: "key", description: Text("Tap + to create a token."))
            }
        }
        .alert("New Token", isPresented: $showingAddSheet) {
            TextField("Name", text: $newTokenName)
            Button("Create") {
                let name = newTokenName
                newTokenName = ""
                Task { await viewModel.create(name: name) }
            }
            Button("Cancel", role: .cancel) { newTokenName = "" }
        }
        .sheet(item: $viewModel.newlyCreatedToken) { created in
            CreatedTokenSheet(token: created)
        }
        .alert("Error", isPresented: Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )) {
            Button("OK") { viewModel.errorMessage = nil }
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
        .task { await viewModel.load() }
    }
}

private struct CreatedTokenSheet: View {
    let token: CreatedToken
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 56))
                    .foregroundStyle(.green)

                VStack(spacing: 8) {
                    Text("Token Created")
                        .font(.title2.weight(.semibold))
                    Text("Copy this token now — it will not be shown again.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }

                VStack(alignment: .leading, spacing: 6) {
                    Text(token.name).font(.caption).foregroundStyle(.secondary)
                    Text(token.plaintext)
                        .font(.system(.body, design: .monospaced))
                        .textSelection(.enabled)
                        .padding(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(Color(.secondarySystemBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                }
                .padding(.horizontal)

                Button {
                    UIPasteboard.general.string = token.plaintext
                } label: {
                    Label("Copy Token", systemImage: "doc.on.doc")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .padding(.horizontal)

                Spacer()
            }
            .padding(.top, 32)
            .navigationTitle("New Token")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
        }
    }
}
