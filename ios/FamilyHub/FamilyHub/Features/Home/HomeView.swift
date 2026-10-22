import SwiftUI
import FamilyHubKit

struct HomeView: View {
    @State private var viewModel: HomeViewModel
    @State private var mealsViewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var householdViewModel: HouseholdViewModel
    @State private var commuteViewModel: CommuteViewModel
    @State private var commuteRefreshID = 0
    @State private var avatarRefreshID = UUID()
    @State private var isVisible = false
    @Environment(\.scenePhase) private var scenePhase
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
        _householdViewModel = State(wrappedValue: HouseholdViewModel(apiClient: apiClient))
        _commuteViewModel = State(wrappedValue: CommuteViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        homeHeader
                        if householdViewModel.departures?.items.isEmpty == false {
                            HouseholdCardsView(model: householdViewModel, apiClient: apiClient, events: viewModel.todayEvents)
                                .padding(.top, 20)
                        }
                        StateContentView(state: viewModel.state, retry: { await viewModel.load() }) { stats in
                            mealsSection(stats)
                        }
                        CommuteCardsView(model: commuteViewModel)
                        if case .loaded(let stats) = viewModel.state {
                            agendaSection
                            choresSection(stats)
                        }
                    }
                    .padding(.bottom, 24)
                }
                .refreshable {
                    async let dashboard: Void = viewModel.load()
                    async let household: Void = householdViewModel.load(force: true)
                    async let commute: Void = commuteViewModel.refresh(force: true)
                    _ = await (dashboard, household, commute)
                }
                .onAppear { isVisible = true }
                .onDisappear {
                    isVisible = false
                    commuteViewModel.stop()
                }
                .task(id: scenePhase == .active && isVisible && activeSheet == nil ? commuteRefreshID : nil) {
                    guard scenePhase == .active, isVisible, activeSheet == nil else { return }
                    await commuteViewModel.run()
                }
                .task(id: scenePhase == .active && isVisible) { [shouldRefresh = scenePhase == .active && isVisible] in
                    guard shouldRefresh else { return }
                    await viewModel.load()
                    await householdViewModel.load(force: true)
                    while !Task.isCancelled {
                        do { try await Task.sleep(for: .seconds(30)) } catch { return }
                        await householdViewModel.load()
                    }
                }
            }
            .groupedBackground()
            .navigationBarTitleDisplayMode(.large)
            .appListStyle()
            .navigationTitle("Today")
            .errorAlert($viewModel.actionError)
            .onReceive(NotificationCenter.default.publisher(for: .familyHubIntentChanged)) { _ in Task { await viewModel.load() } }
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { activeSheet = .profile } label: {
                        UserAvatar(user: viewModel.currentUser, size: 32, apiClient: apiClient)
                            .id(avatarRefreshID)
                            .frame(width: 44, height: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Settings")
                    .accessibilityIdentifier("Open profile")
                }
            }
            .sheet(item: $activeSheet, onDismiss: {
                commuteRefreshID += 1
                avatarRefreshID = UUID()
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
        .task(id: navigation.choreRefreshID) { await viewModel.load() }
        .onChange(of: navigation.selectedTab) { _, tab in
            if tab == .today { Task { await viewModel.load() } }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase != .active { commuteViewModel.stop() }
            if phase == .active && isVisible { Task { await viewModel.load() } }
        }
        .onReceive(NotificationCenter.default.publisher(for: UIApplication.significantTimeChangeNotification)) { _ in
            commuteViewModel.tick()
        }
    }

    private var homeHeader: some View {
        Group {
            if dynamicTypeSize.isAccessibilitySize {
                stackedHeader
            } else {
                ViewThatFits(in: .horizontal) {
                    HStack(alignment: .firstTextBaseline, spacing: 12) {
                        homeDate.fixedSize()
                        Spacer(minLength: 0)
                        weatherLink.fixedSize()
                    }
                    stackedHeader
                }
            }
        }
        .font(.subheadline)
        .padding(.horizontal, 20)
    }

    private var stackedHeader: some View {
        VStack(alignment: .leading, spacing: 4) {
            homeDate
            weatherLink
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var homeDate: some View {
        Text(Date.now.formatted(.dateTime.weekday(.wide).day().month(.wide)))
            .foregroundStyle(AppTheme.secondary)
            .accessibilityIdentifier("home-date")
    }

    private var weatherLink: some View {
        NavigationLink {
            WeatherDetailView(apiClient: apiClient)
        } label: {
            Group {
                if let weather = householdViewModel.weather {
                    Label("\(weather.current.temperature, specifier: "%.0f")°C", systemImage: weather.current.symbol)
                        .accessibilityLabel("Weather at home: \(weather.current.temperature, specifier: "%.0f") degrees Celsius, \(weather.current.condition)")
                } else {
                    Label(householdViewModel.weatherMessage == "Loading weather…" ? "Loading weather…" : "Weather unavailable",
                          systemImage: "cloud.sun")
                }
            }
            .frame(minHeight: 44)
        }
        .buttonStyle(.borderless)
        .foregroundStyle(AppTheme.accent)
        .accessibilityIdentifier("home-weather")
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
                TimelineView(.periodic(from: .now, by: 60)) { context in
                    VStack(spacing: 0) {
                        ForEach(Array(viewModel.todayEvents.prefix(3).enumerated()), id: \.element.id) { index, event in
                            if index > 0 {
                                Divider().padding(.leading, 20)
                            }
                            NavigationLink {
                                EventDetailsView(event: event, apiClient: apiClient)
                            } label: {
                                EventRow(event: event, now: context.date)
                            }
                            .buttonStyle(.plain)
                            .accessibilityIdentifier("home-event-\(event.id)")
                            .accessibilityValue(HomeAgendaEventPresentation.isPast(event: event, at: context.date) ? "Past event" : "")
                        }
                    }
                    .groupedCard()
                }
                .padding(.horizontal, 20)
            }
        }
    }

    // MARK: - Today's Meals

    private func mealsSection(_ stats: DashboardStats) -> some View {
        let todayKey = APIDate.dayString(Date())
        return VStack(alignment: .leading, spacing: 0) {
            HomeSectionHeader(title: "Meal plan") {
                Button {
                    navigation.openMealPlan()
                } label: {
                    Text("View week")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(Color.accentColor)
                }
                .buttonStyle(.plain)
            }
            let layout = UIDevice.current.userInterfaceIdiom == .pad && !dynamicTypeSize.isAccessibilitySize
                ? AnyLayout(HStackLayout(spacing: 12))
                : AnyLayout(VStackLayout(spacing: 0))
            layout {
                ForEach(["breakfast", "lunch", "dinner"], id: \.self) { mealType in
                    if mealType != "breakfast" && UIDevice.current.userInterfaceIdiom == .phone { Divider().padding(.leading, 16) }
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
            .groupedCard()
            .padding(.horizontal, 20)
        }
    }

    // MARK: - Today's Chores

    private func choresSection(_ stats: DashboardStats) -> some View {
        let overdue = stats.choresOverdueList.filter { !viewModel.completedChoreIDs.contains($0.id) }
        let today = stats.choresDueTodayList.filter { !viewModel.completedChoreIDs.contains($0.id) }
        return VStack(alignment: .leading, spacing: 0) {
            if !overdue.isEmpty { choreGroup("Overdue", chores: overdue) }
            if !today.isEmpty { choreGroup("Today’s chores", chores: today) }
            if overdue.isEmpty && today.isEmpty {
                HomeSectionHeader(title: "Chores") { EmptyView() }
                Label("No chores due", systemImage: "checkmark.circle.fill")
                    .foregroundStyle(AppTheme.secondary)
                    .padding(.horizontal, 20)
            }
        }
    }

    private func choreGroup(_ title: String, chores: [Chore]) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HomeSectionHeader(title: title) { EmptyView() }
            VStack(spacing: 0) {
                ForEach(Array(chores.enumerated()), id: \.element.id) { index, chore in
                    if index > 0 { Divider().padding(.leading, 64) }
                    ChoreRowView(
                        chore: chore,
                        user: viewModel.users[chore.assignedToUserID ?? ""],
                        apiClient: apiClient,
                        onComplete: { Task { await viewModel.completeChore(id: chore.id) } },
                        onSkip: { Task { await viewModel.skipChore(id: chore.id) } }
                    )
                }
            }
            .groupedCard()
            .padding(.horizontal, 20)
        }
    }

}

// MARK: - HomeSectionHeader

struct HomeSectionHeader<Action: View>: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let title: String
    @ViewBuilder var action: () -> Action

    var body: some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 4))
            : AnyLayout(HStackLayout(alignment: .firstTextBaseline))
        layout {
            Text(title)
                .font(.system(.title2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.ink)
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

    private var mealIcon: String {
        switch mealType {
        case "breakfast": return "sun.horizon.fill"
        case "lunch":     return "sun.max.fill"
        default:          return "moon.stars.fill"
        }
    }

    var body: some View {
        Button(action: onTap) {
            let layout = UIDevice.current.userInterfaceIdiom == .pad
                ? AnyLayout(VStackLayout(alignment: .leading, spacing: 14))
                : AnyLayout(HStackLayout(spacing: 14))
            layout {
                if hasContent {
                    RecipeThumbView(recipeID: mealPlan?.recipeID, apiClient: apiClient, size: 42, placeholderSymbol: mealIcon)
                        .accessibilityHidden(true)
                } else { WarmIcon(symbol: "fork.knife") }
                VStack(alignment: .leading, spacing: 4) {
                    Text(hasContent ? mealPlan!.name : mealType.capitalized)
                        .font(.body.weight(.medium)).foregroundStyle(AppTheme.ink)
                    Text(hasContent ? mealType.capitalized : "Not planned")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                }
                .fixedSize(horizontal: false, vertical: true)
                Spacer(minLength: 8)
                Image(systemName: hasContent ? "chevron.right" : "plus")
                    .foregroundStyle(AppTheme.secondary).accessibilityHidden(true)
            }
            .padding(16)
            .frame(maxWidth: .infinity, minHeight: 76, alignment: .leading)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityElement(children: .combine)
        .accessibilityIdentifier("today-meal-\(mealType)")
    }
}

// MARK: - EventRow

private struct EventRow: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let event: CalendarEvent
    let now: Date

    private var isPast: Bool {
        HomeAgendaEventPresentation.isPast(event: event, at: now)
    }

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
                        .foregroundStyle(AppTheme.secondary)
                } else {
                    Text(event.startTime, style: .time)
                        .font(.subheadline.weight(.bold).monospacedDigit())
                    if let end = event.endTime {
                        Text(end, style: .time)
                            .font(.caption.monospacedDigit())
                            .foregroundStyle(AppTheme.secondary)
                    }
                }
            }
            .foregroundStyle(isPast ? AppTheme.secondary : AppTheme.ink)
            .frame(minWidth: 62, alignment: .trailing)

            // Colored dot
            if !dynamicTypeSize.isAccessibilitySize {
                Circle().fill(dotColor.opacity(isPast ? 0.45 : 1)).frame(width: 8, height: 8)
                    .accessibilityHidden(true)
            }

            // Title + calendar name
            VStack(alignment: .leading, spacing: 2) {
                Text(event.title)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(isPast ? AppTheme.secondary : AppTheme.ink)
                    .fixedSize(horizontal: false, vertical: true)
                if !event.description.isEmpty {
                    Text(event.description)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.secondary)
                        .lineLimit(2)
                        .fixedSize(horizontal: false, vertical: true)
                        .accessibilityIdentifier("home-event-description-\(event.id)")
                }
            }

            if !dynamicTypeSize.isAccessibilitySize { Spacer(minLength: 0) }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 20)
        .padding(.vertical, 13)
    }
}
