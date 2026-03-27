import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var selectedScope: Scope = .plan
    private let apiClient: any APIClientProtocol

    enum Scope: String, CaseIterable {
        case plan = "Plan"
        case recipes = "Recipes"
    }

    private let mealTypes = ["breakfast", "lunch", "dinner"]

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "EEE d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dateKeyFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "yyyy-MM-dd"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let weekTitleFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d MMM"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("View", selection: $selectedScope) {
                    ForEach(Scope.allCases, id: \.self) { scope in
                        Text(scope.rawValue).tag(scope)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.vertical, 8)

                if selectedScope == .plan {
                    planView
                } else {
                    recipesView
                }
            }
            .navigationTitle("Meals")
            .toolbar {
                if selectedScope == .plan {
                    ToolbarItem(placement: .navigationBarLeading) {
                        Button { viewModel.previousWeek() } label: {
                            Image(systemName: "chevron.left")
                        }
                    }
                    ToolbarItem(placement: .principal) {
                        Text(weekTitle)
                            .font(.headline)
                    }
                    ToolbarItem(placement: .navigationBarTrailing) {
                        Button { viewModel.nextWeek() } label: {
                            Image(systemName: "chevron.right")
                        }
                    }
                }
            }
        }
        .task {
            await viewModel.load()
            await recipesViewModel.load()
        }
    }

    private var weekTitle: String {
        let start = viewModel.currentWeek
        let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
        return "\(Self.weekTitleFormatter.string(from: start)) – \(Self.weekTitleFormatter.string(from: end))"
    }

    @ViewBuilder
    private var planView: some View {
        switch viewModel.state {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let error):
            ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
        case .loaded(let meals):
            List {
                ForEach(0..<7, id: \.self) { offset in
                    let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                    let dateKey = Self.dateKeyFormatter.string(from: date)
                    Section(Self.dayFormatter.string(from: date)) {
                        ForEach(mealTypes, id: \.self) { mealType in
                            let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                            HStack {
                                Text(mealType.capitalized)
                                    .foregroundStyle(.secondary)
                                Spacer()
                                Text(plan?.name ?? "—")
                                    .foregroundStyle(plan != nil ? .primary : .tertiary)
                            }
                        }
                    }
                }
            }
            .listStyle(.insetGrouped)
            .refreshable { await viewModel.load() }
        }
    }

    @ViewBuilder
    private var recipesView: some View {
        switch recipesViewModel.state {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let error):
            ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
        case .loaded:
            ScrollView {
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                    ForEach(recipesViewModel.filteredRecipes) { recipe in
                        NavigationLink {
                            RecipeDetailView(recipe: recipe, apiClient: apiClient)
                        } label: {
                            RecipeCard(recipe: recipe)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal)
            }
            .refreshable { await recipesViewModel.load() }
            .searchable(text: $recipesViewModel.searchQuery, prompt: "Search recipes")
        }
    }
}

private struct RecipeCard: View {
    let recipe: Recipe

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            RoundedRectangle(cornerRadius: 8)
                .fill(Color(.tertiarySystemFill))
                .aspectRatio(4/3, contentMode: .fit)
                .overlay {
                    Image(systemName: "fork.knife")
                        .foregroundStyle(.tertiary)
                        .font(.title2)
                }
            VStack(alignment: .leading, spacing: 4) {
                Text(recipe.title)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(2)
                HStack(spacing: 8) {
                    if let prep = recipe.prepTime {
                        Label("\(prep) prep", systemImage: "clock")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let servings = recipe.servings {
                        Label("\(servings)", systemImage: "person.2")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .padding(.horizontal, 6)
            .padding(.bottom, 8)
        }
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
