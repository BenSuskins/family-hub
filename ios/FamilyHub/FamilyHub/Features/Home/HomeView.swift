import SwiftUI

struct HomeView: View {
    @State private var viewModel: HomeViewModel
    @State private var mealsViewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var showProfile = false
    @State private var editingMeal: EditingMeal?
    @State private var showRecipesView = false
    private let apiClient: any APIClientProtocol

    private static let dayDateFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "EEEE · MMMM d"
        return f
    }()
    private static let timeFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "h:mm a"
        f.amSymbol = "AM"; f.pmSymbol = "PM"
        return f
    }()
    private static let dateKeyFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: HomeViewModel(apiClient: apiClient))
        _mealsViewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    private var greeting: String {
        let hour = Calendar.current.component(.hour, from: Date())
        if hour < 12 { return "Good morning," }
        if hour < 17 { return "Good afternoon," }
        return "Good evening,"
    }

    private var firstName: String {
        viewModel.currentUser?.name.components(separatedBy: " ").first ?? "there"
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                greetingHeader
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        switch viewModel.state {
                        case .idle, .loading:
                            ProgressView()
                                .frame(maxWidth: .infinity)
                                .padding(.top, 40)
                        case .failed(let error):
                            Text(error.localizedDescription)
                                .foregroundStyle(.red)
                                .padding()
                        case .loaded(let stats):
                            agendaSection
                            mealsSection(stats)
                            choresSection(stats)
                        }
                    }
                    .padding(.bottom, 24)
                }
                .refreshable { await viewModel.load() }
            }
            .meshBackground()
            .navigationBarTitleDisplayMode(.inline)
            .toolbar(.hidden, for: .navigationBar)
            .sheet(isPresented: $showProfile) {
                ProfileView(apiClient: apiClient)
            }
            .sheet(item: $editingMeal, onDismiss: {
                Task { await viewModel.load() }
            }) { meal in
                MealEditSheet(meal: meal, viewModel: mealsViewModel, apiClient: apiClient)
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Greeting Header

    private var greetingHeader: some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 2) {
                Text(Self.dayDateFormatter.string(from: Date()).uppercased())
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(.secondary)
                    .kerning(0.5)
                    .padding(.bottom, 2)
                Text(greeting)
                    .font(.system(size: 18, weight: .light))
                    .foregroundStyle(.secondary)
                Text(firstName)
                    .font(.system(size: 36, weight: .black))
                    .foregroundStyle(.primary)
                    .tracking(-0.5)
            }
            Spacer()
            Button {
                showProfile = true
            } label: {
                UserAvatar(user: viewModel.currentUser, size: 44, apiClient: apiClient)
            }
            .buttonStyle(.plain)
            .padding(.top, 4)
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 8)
    }

    // MARK: - Agenda

    @ViewBuilder
    private var agendaSection: some View {
        if !viewModel.todayEvents.isEmpty {
            VStack(alignment: .leading, spacing: 0) {
                HomeSectionHeader(title: "Agenda") {
                    NavigationLink {
                        // Navigate to calendar detail
                    } label: {
                        Text("Calendar")
                            .font(.system(size: 15, weight: .medium))
                            .foregroundStyle(Color.accentColor)
                    }
                }
                VStack(spacing: 0) {
                    ForEach(Array(viewModel.todayEvents.prefix(3).enumerated()), id: \.element.id) { index, event in
                        if index > 0 {
                            Divider().padding(.leading, 20)
                        }
                        EventRow(event: event, timeFormatter: Self.timeFormatter)
                    }
                }
                .glassCard(radius: 16)
                .padding(.horizontal, 16)
            }
        }
    }

    // MARK: - Today's Meals

    private func mealsSection(_ stats: DashboardStats) -> some View {
        let todayKey = Self.dateKeyFormatter.string(from: Date())
        return VStack(alignment: .leading, spacing: 0) {
            HomeSectionHeader(title: "Today's meals") {
                NavigationLink {
                    MealsView(apiClient: apiClient)
                } label: {
                    Text("Plan")
                        .font(.system(size: 15, weight: .medium))
                        .foregroundStyle(Color.accentColor)
                }
            }
            HStack(spacing: 12) {
                ForEach(["breakfast", "lunch", "dinner"], id: \.self) { mealType in
                    let plan = stats.todayMeals.first(where: { $0.mealType == mealType })
                    TodayMealCard(mealType: mealType, mealPlan: plan, apiClient: apiClient) {
                        if plan?.recipeID != nil {
                            showRecipesView = true
                        } else {
                            editingMeal = EditingMeal(
                                date: todayKey,
                                mealType: mealType,
                                name: plan?.name ?? "",
                                recipeID: plan?.recipeID
                            )
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
            .navigationDestination(isPresented: $showRecipesView) {
                RecipesView(apiClient: apiClient)
            }
        }
    }

    // MARK: - Today's Chores

    private func choresSection(_ stats: DashboardStats) -> some View {
        let allDue = (stats.choresOverdueList + stats.choresDueTodayList)
            .filter { !viewModel.completedChoreIDs.contains($0.id) }
        return VStack(alignment: .leading, spacing: 0) {
            HomeSectionHeader(title: "Today's chores") {
                NavigationLink {
                    ChoresListView(apiClient: apiClient)
                } label: {
                    Text("Manage")
                        .font(.system(size: 15, weight: .medium))
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
                .glassCard(radius: 16)
                .padding(.horizontal, 16)
            }
        }
    }
}

// MARK: - HomeSectionHeader

private struct HomeSectionHeader<Action: View>: View {
    let title: String
    @ViewBuilder var action: () -> Action

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            Text(title)
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(.primary)
            Spacer()
            action()
        }
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

    @State private var recipeImageData: Data?

    private var hasContent: Bool { !(mealPlan?.name ?? "").isEmpty }
    private var hasRecipe: Bool { mealPlan?.recipeID != nil }
    private var hasTextOnly: Bool { hasContent && !hasRecipe }

    private var mealIcon: String {
        switch mealType {
        case "breakfast": return "sun.horizon.fill"
        case "lunch":     return "sun.max.fill"
        default:          return "moon.stars.fill"
        }
    }

    private var iconTint: Color {
        switch mealType {
        case "breakfast": return .orange
        case "lunch":     return .teal
        default:          return .indigo
        }
    }

    var body: some View {
        Button(action: onTap) {
            VStack(alignment: .leading, spacing: 0) {
                iconBadge
                Spacer(minLength: 12)
                VStack(alignment: .leading, spacing: 3) {
                    Text(mealType.uppercased())
                        .font(.system(size: 10, weight: .bold))
                        .foregroundStyle(hasRecipe ? AnyShapeStyle(.white.opacity(0.85)) : AnyShapeStyle(.tertiary))
                        .kerning(0.8)
                    Text(hasContent ? mealPlan!.name : "–")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(hasRecipe ? AnyShapeStyle(.white) : (hasTextOnly ? AnyShapeStyle(.primary) : AnyShapeStyle(.tertiary)))
                        .lineLimit(3)
                        .minimumScaleFactor(0.85)
                }
            }
            .padding(12)
            .frame(maxWidth: .infinity, minHeight: 124, alignment: .leading)
            .background(cardBackground)
            .clipShape(RoundedRectangle(cornerRadius: 16))
        }
        .buttonStyle(.plain)
        .task(id: mealPlan?.recipeID) {
            recipeImageData = nil
            guard let id = mealPlan?.recipeID else { return }
            recipeImageData = try? await apiClient.fetchRecipeImage(id: id)
        }
    }

    @ViewBuilder
    private var iconBadge: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 7)
                .fill(hasRecipe ? Color.white.opacity(0.2) : iconTint.opacity(0.16))
                .frame(width: 22, height: 22)
            Image(systemName: mealIcon)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(hasRecipe ? .white : iconTint)
        }
    }

    @ViewBuilder
    private var cardBackground: some View {
        if hasRecipe {
            if let data = recipeImageData, let uiImage = UIImage(data: data) {
                ZStack {
                    Image(uiImage: uiImage)
                        .resizable()
                        .scaledToFill()
                    Color.black.opacity(0.35)
                }
            } else {
                LinearGradient(
                    colors: [iconTint.opacity(0.7), iconTint.opacity(0.4)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            }
        } else if hasTextOnly {
            LinearGradient(
                colors: [iconTint.opacity(0.18), iconTint.opacity(0.08)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        } else {
            Color(UIColor.secondarySystemGroupedBackground)
        }
    }
}

// MARK: - EventRow

private struct EventRow: View {
    let event: CalendarEvent
    let timeFormatter: DateFormatter

    private var dotColor: Color {
        Color(hex: event.color) ?? .accentColor
    }

    var body: some View {
        HStack(alignment: .center, spacing: 12) {
            // Time block
            VStack(alignment: .trailing, spacing: 1) {
                if event.allDay {
                    Text("All day")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(.secondary)
                } else {
                    Text(timeFormatter.string(from: event.startTime))
                        .font(.system(size: 15, weight: .bold).monospacedDigit())
                    if let end = event.endTime {
                        Text(timeFormatter.string(from: end))
                            .font(.system(size: 12).monospacedDigit())
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .frame(width: 62, alignment: .trailing)

            // Colored dot
            Circle()
                .fill(dotColor)
                .frame(width: 8, height: 8)

            // Title + calendar name
            VStack(alignment: .leading, spacing: 2) {
                Text(event.title)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(1)
                if !event.description.isEmpty {
                    Text(event.description)
                        .font(.system(size: 13))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }

            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 13)
    }
}
