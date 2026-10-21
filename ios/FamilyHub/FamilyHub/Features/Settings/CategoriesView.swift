import SwiftUI
import FamilyHubKit

@Observable
final class CategoriesViewModel {
    var categories: [ChoreCategory] = []
    var isLoading = false
    var actionError: APIError?

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            categories = try await apiClient.fetchCategories()
        } catch {
            actionError = .from(error)
        }
    }

    func create(name: String) async {
        do {
            let created = try await apiClient.createCategory(name: name)
            categories.append(created)
        } catch {
            actionError = .from(error)
        }
    }

    func rename(category: ChoreCategory, to name: String) async {
        do {
            let updated = try await apiClient.updateCategory(id: category.id, name: name)
            if let index = categories.firstIndex(where: { $0.id == updated.id }) {
                categories[index] = updated
            }
        } catch {
            actionError = .from(error)
        }
    }

    func delete(at offsets: IndexSet) async {
        let toDelete = offsets.map { categories[$0] }
        for category in toDelete {
            do {
                try await apiClient.deleteCategory(id: category.id)
                categories.removeAll { $0.id == category.id }
            } catch {
                actionError = .from(error)
            }
        }
    }
}

struct CategoriesView: View {
    let apiClient: any APIClientProtocol

    @State private var viewModel: CategoriesViewModel
    @State private var pendingDeletion: IndexSet?
    @State private var activeSheet: CategorySheet?

    private enum CategorySheet: Identifiable {
        case create, edit(ChoreCategory)
        var id: String {
            switch self {
            case .create: return "create"
            case .edit(let category): return category.id
            }
        }
    }
    @State private var newCategoryName = ""
    @State private var editName = ""

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        self._viewModel = State(initialValue: CategoriesViewModel(apiClient: apiClient))
    }

    private var deletionName: String {
        guard let index = pendingDeletion?.first, viewModel.categories.indices.contains(index) else { return "category" }
        return viewModel.categories[index].name
    }

    var body: some View {
        List {
            Section { Text("Categories group chores for your family.").foregroundStyle(AppTheme.secondary) }.listRowBackground(AppTheme.surface)
            if viewModel.categories.isEmpty && !viewModel.isLoading {
                Section {
                    ContentUnavailableView("No Categories", systemImage: "tag", description: Text("Tap + to add a category."))
                }.listRowBackground(AppTheme.surface)
            }
            ForEach(viewModel.categories) { category in
                HStack {
                    Text(category.name)
                    Spacer()
                    Menu {
                        Button("Rename") { editName = category.name; activeSheet = .edit(category) }
                        Button("Delete", role: .destructive) {
                            if let index = viewModel.categories.firstIndex(where: { $0.id == category.id }) { pendingDeletion = IndexSet([index]) }
                        }
                    } label: { Image(systemName: "ellipsis").frame(width: 44, height: 44) }
                    .accessibilityLabel("Actions for \(category.name)")
                }
            .listRowBackground(AppTheme.surface)
                    .swipeActions(edge: .trailing) {
                        Button(role: .destructive) {
                            if let index = viewModel.categories.firstIndex(where: { $0.id == category.id }) {
                                pendingDeletion = IndexSet([index])
                            }
                        } label: {
                            Label("Delete", systemImage: "trash")
                        }
                        Button {
                            activeSheet = .edit(category)
                            editName = category.name
                        } label: {
                            Label("Rename", systemImage: "pencil")
                        }
                        .tint(AppTheme.accent)
                    }
            }
        }
        .appListStyle()
        .navigationTitle("Categories")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { activeSheet = .create } label: {
                    Image(systemName: "plus")
                }
                .accessibilityLabel("New Category")
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.categories.isEmpty {
                ProgressView()
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .create:
                NameEntrySheet(title: "New Category", name: $newCategoryName, fieldTitle: "Category name") { name in
                    viewModel.actionError = nil
                    await viewModel.create(name: name)
                    let error = viewModel.actionError
                    viewModel.actionError = nil
                    return error
                }
            case .edit(let category):
                NameEntrySheet(title: "Rename Category", name: $editName, fieldTitle: "Category name", explanation: "This changes the category for the whole family.") { name in
                    viewModel.actionError = nil
                    await viewModel.rename(category: category, to: name)
                    let error = viewModel.actionError
                    viewModel.actionError = nil
                    return error
                }
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
