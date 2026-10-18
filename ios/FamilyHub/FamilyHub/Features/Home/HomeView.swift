import SwiftUI
import FamilyHubKit

struct HomeView: View {
    @State private var viewModel: HomeViewModel
    @State private var mealsViewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var activeSheet: ActiveSheet?

    private enum ActiveSheet: Identifiable {
        case profile, meal(EditingMeal)
        var id: String {
            switch self {
            case .profile: return "profile"
            case .meal(let meal): return meal.id
            }
        }
    }
    @State private var selectedRecipe: Recipe?
    @Bindable var navigation: AppNavigation
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol, navigation: AppNavigation) {
        self.apiClient = apiClient
        self.navigation = navigation
        _viewModel = State(wrappedValue: HomeViewModel(apiClient: apiClient))
        _mealsViewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { stats in
                            Text(Date.now.formatted(date: .complete, time: .omitted))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 20)
                            agendaSection
                            mealsSection(stats)
                            choresSection(stats)
                        }
                        HomeSectionHeader(title: "Household") { EmptyView() }
                        Button {
                            navigation.householdPath = [.shopping]
                            navigation.selectedTab = .household
                        } label: {
                            Label("Shopping List", systemImage: "cart")
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(16)
                                .groupedCard()
                        }
                        .buttonStyle(.borderless)
                        .padding(.horizontal, 16)
                        .padding(.bottom, 12)
                        HouseholdCardsView(apiClient: apiClient)
                    }
                    .padding(.bottom, 24)
                }
                .refreshable { await viewModel.load() }
            }
            .groupedBackground()
            .navigationBarTitleDisplayMode(.inline)
            .navigationTitle("Today")
            .errorAlert($viewModel.actionError)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { activeSheet = .profile } label: {
                        UserAvatar(user: viewModel.currentUser, size: 32, apiClient: apiClient)
                            .frame(width: 44, height: 44)
                    }
                    .accessibilityLabel("Open profile")
                }
            }
            .sheet(item: $activeSheet, onDismiss: {
                Task { await viewModel.load() }
            }) { sheet in
                switch sheet {
                case .profile: ProfileView(apiClient: apiClient)
                case .meal(let meal): MealEditSheet(meal: meal, viewModel: mealsViewModel, apiClient: apiClient)
                }
            }
            .navigationDestination(item: $selectedRecipe) { recipe in
                RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: recipesViewModel)
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Agenda

    @ViewBuilder
    private var agendaSection: some View {
        if !viewModel.todayEvents.isEmpty {
            VStack(alignment: .leading, spacing: 0) {
                HomeSectionHeader(title: "Agenda") {
                    Button {
                        navigation.selectedTab = .calendar
                    } label: {
                        Text("Calendar")
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(Color.accentColor)
                    }
                    .buttonStyle(.plain)
                }
                VStack(spacing: 0) {
                    ForEach(Array(viewModel.todayEvents.prefix(3).enumerated()), id: \.element.id) { index, event in
                        if index > 0 {
                            Divider().padding(.leading, 20)
                        }
                        NavigationLink {
                            EventDetailsView(event: event, apiClient: apiClient)
                        } label: {
                            EventRow(event: event)
                        }.buttonStyle(.plain)
                    }
                }
                .groupedCard(radius: 16)
                .padding(.horizontal, 16)
            }
        }
    }

    // MARK: - Today's Meals

    private func mealsSection(_ stats: DashboardStats) -> some View {
        let todayKey = APIDate.dayString(Date())
        return VStack(alignment: .leading, spacing: 0) {
            HomeSectionHeader(title: "Today's meals") {
                Button {
                    navigation.openMealPlan()
                } label: {
                    Text("Plan")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(Color.accentColor)
                }
                .buttonStyle(.plain)
            }
            let layout = dynamicTypeSize.isAccessibilitySize
                ? AnyLayout(VStackLayout(spacing: 12))
                : AnyLayout(HStackLayout(spacing: 12))
            layout {
                ForEach(["breakfast", "lunch", "dinner"], id: \.self) { mealType in
                    let plan = stats.todayMeals.first(where: { $0.mealType == mealType })
                    TodayMealCard(mealType: mealType, mealPlan: plan, apiClient: apiClient) {
                        if let recipeID = plan?.recipeID {
                            selectedRecipe = Recipe(
                                id: recipeID,
                                title: plan?.name ?? "",
                                steps: nil, ingredients: nil, mealType: nil,
                                servings: nil, prepTime: nil, cookTime: nil,
                                sourceURL: nil, categoryID: nil, hasImage: true
                            )
                        } else {
                            activeSheet = .meal(EditingMeal(
                                date: todayKey,
                                mealType: mealType,
                                name: plan?.name ?? "",
                                recipeID: plan?.recipeID
                            ))
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
        }
    }

    // MARK: - Today's Chores

    private func choresSection(_ stats: DashboardStats) -> some View {
        let allDue = (stats.choresOverdueList + stats.choresDueTodayList)
            .filter { !viewModel.completedChoreIDs.contains($0.id) }
        return VStack(alignment: .leading, spacing: 0) {
            HomeSectionHeader(title: "Today's chores") {
                Button {
                    navigation.selectedTab = .tasks
                } label: {
                    Text("Manage")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(Color.accentColor)
                }
            }

            if allDue.isEmpty {
                HStack(spacing: 8) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(Color.appGreen)
                    Text("All caught up!")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 8)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(allDue.enumerated()), id: \.element.id) { index, chore in
                        if index > 0 {
                            Divider().padding(.leading, 64)
                        }
                        ChoreRowView(
                            chore: chore,
                            user: viewModel.users[chore.assignedToUserID ?? ""],
                            apiClient: apiClient,
                            onComplete: {
                                Task { await viewModel.completeChore(id: chore.id) }
                            }
                        )
                    }
                }
                .groupedCard(radius: 16)
                .padding(.horizontal, 16)
            }
        }
    }
}

// MARK: - HomeSectionHeader

private struct HomeSectionHeader<Action: View>: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let title: String
    @ViewBuilder var action: () -> Action

    var body: some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 4))
            : AnyLayout(HStackLayout(alignment: .firstTextBaseline))
        layout {
            Text(title)
                .font(.title2.weight(.bold))
                .foregroundStyle(.primary)
            if !dynamicTypeSize.isAccessibilitySize { Spacer() }
            action().frame(minHeight: 44)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 20)
        .padding(.top, 28)
        .padding(.bottom, 10)
    }
}

// MARK: - TodayMealCard

private struct TodayMealCard: View {
    let mealType: String
    let mealPlan: MealPlan?
    let apiClient: any APIClientProtocol
    var onTap: () -> Void


    private var hasContent: Bool { !(mealPlan?.name ?? "").isEmpty }
    private var hasRecipe: Bool { mealPlan?.recipeID != nil }

    private var mealIcon: String {
        switch mealType {
        case "breakfast": return "sun.horizon.fill"
        case "lunch":     return "sun.max.fill"
        default:          return "moon.stars.fill"
        }
    }

    var body: some View {
        Button(action: onTap) {
            VStack(alignment: .leading, spacing: 8) {
                if hasRecipe {
                    RecipeThumbView(recipeID: mealPlan?.recipeID, apiClient: apiClient, size: 64)
                        .accessibilityHidden(true)
                }
                Label(mealType.capitalized, systemImage: mealIcon)
                    .font(.caption)
                    .foregroundStyle(Color.secondary)
                Text(hasContent ? mealPlan!.name : "Add meal")
                    .font(.body.weight(.medium))
                    .foregroundStyle(Color.primary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(12)
            .frame(maxWidth: .infinity, minHeight: 124, alignment: .topLeading)
            .groupedCard()
        }
        .buttonStyle(.borderless)
        .accessibilityElement(children: .combine)
    }
}

// MARK: - EventRow

private struct EventRow: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let event: CalendarEvent

    private var dotColor: Color {
        Color(hex: event.color) ?? .accentColor
    }

    var body: some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
            : AnyLayout(HStackLayout(alignment: .center, spacing: 12))
        layout {
            // Time block
            VStack(alignment: .trailing, spacing: 1) {
                if event.allDay {
                    Text("All day")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.secondary)
                } else {
                    Text(event.startTime, style: .time)
                        .font(.subheadline.weight(.bold).monospacedDigit())
                    if let end = event.endTime {
                        Text(end, style: .time)
                            .font(.caption.monospacedDigit())
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .frame(minWidth: 62, alignment: .trailing)

            // Colored dot
            if !dynamicTypeSize.isAccessibilitySize {
                Circle().fill(dotColor).frame(width: 8, height: 8)
                    .accessibilityHidden(true)
            }

            // Title + calendar name
            VStack(alignment: .leading, spacing: 2) {
                Text(event.title)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.primary)
                    .fixedSize(horizontal: false, vertical: true)
                if !event.description.isEmpty {
                    Text(event.description)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }

            if !dynamicTypeSize.isAccessibilitySize { Spacer(minLength: 0) }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 16)
        .padding(.vertical, 13)
    }
}
