import SwiftUI

@Observable
final class CategoriesViewModel {
    var categories: [Category] = []
    var isLoading = false
    var errorMessage: String?

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
            errorMessage = "Failed to load categories"
        }
    }

    func create(name: String) async {
        do {
            let created = try await apiClient.createCategory(name: name)
            categories.append(created)
        } catch {
            errorMessage = "Failed to create category"
        }
    }

    func rename(category: Category, to name: String) async {
        do {
            let updated = try await apiClient.updateCategory(id: category.id, name: name)
            if let index = categories.firstIndex(where: { $0.id == updated.id }) {
                categories[index] = updated
            }
        } catch {
            errorMessage = "Failed to rename category"
        }
    }

    func delete(at offsets: IndexSet) async {
        let toDelete = offsets.map { categories[$0] }
        categories.remove(atOffsets: offsets)
        for category in toDelete {
            do {
                try await apiClient.deleteCategory(id: category.id)
            } catch {
                errorMessage = "Failed to delete \(category.name)"
            }
        }
    }
}

struct CategoriesView: View {
    let apiClient: any APIClientProtocol

    @State private var viewModel: CategoriesViewModel
    @State private var showingAddSheet = false
    @State private var newCategoryName = ""
    @State private var editingCategory: Category?
    @State private var editName = ""

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        self._viewModel = State(initialValue: CategoriesViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            ForEach(viewModel.categories) { category in
                Text(category.name)
                    .swipeActions(edge: .trailing) {
                        Button(role: .destructive) {
                            if let index = viewModel.categories.firstIndex(where: { $0.id == category.id }) {
                                Task { await viewModel.delete(at: IndexSet([index])) }
                            }
                        } label: {
                            Label("Delete", systemImage: "trash")
                        }
                        Button {
                            editingCategory = category
                            editName = category.name
                        } label: {
                            Label("Rename", systemImage: "pencil")
                        }
                        .tint(.blue)
                    }
            }
        }
        .navigationTitle("Categories")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { showingAddSheet = true } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.categories.isEmpty {
                ProgressView()
            } else if viewModel.categories.isEmpty && !viewModel.isLoading {
                ContentUnavailableView("No Categories", systemImage: "tag", description: Text("Tap + to add a category."))
            }
        }
        .alert("New Category", isPresented: $showingAddSheet) {
            TextField("Name", text: $newCategoryName)
            Button("Add") {
                let name = newCategoryName
                newCategoryName = ""
                Task { await viewModel.create(name: name) }
            }
            Button("Cancel", role: .cancel) { newCategoryName = "" }
        }
        .alert("Rename", isPresented: Binding(
            get: { editingCategory != nil },
            set: { if !$0 { editingCategory = nil } }
        )) {
            TextField("Name", text: $editName)
            Button("Save") {
                guard let category = editingCategory else { return }
                let name = editName
                editingCategory = nil
                Task { await viewModel.rename(category: category, to: name) }
            }
            Button("Cancel", role: .cancel) { editingCategory = nil }
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
