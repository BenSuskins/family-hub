import SwiftUI

struct HomeView: View {
    @State private var viewModel: HomeViewModel
    @State private var mealsViewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var showProfile = false
    @State private var editingMeal: EditingMeal?
    private let apiClient: any APIClientProtocol

    private static let dateFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "EEEE, MMMM d"
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

    var body: some View {
        NavigationStack {
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
                        upNextSection
                        mealsSection(stats)
                        choresSection(stats)
                    }
                }
                .padding(.bottom, 24)
            }
            .meshBackground()
            .refreshable { await viewModel.load() }
            .navigationTitle("Today")
            .navigationBarTitleDisplayMode(.large)
            .navigationSubtitle(Self.dateFormatter.string(from: Date()))
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showProfile = true
                    } label: {
                        UserAvatar(user: viewModel.currentUser, size: 32, apiClient: apiClient)
                    }
                    .buttonStyle(.plain)
                }
            }
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

    // MARK: - Up Next

    @ViewBuilder
    private var upNextSection: some View {
        if !viewModel.todayEvents.isEmpty {
            VStack(alignment: .leading, spacing: 0) {
                HStack {
                    SectionHeaderLabel(text: "Up next")
                    Spacer()
                }
                VStack(spacing: 0) {
                    ForEach(Array(viewModel.todayEvents.prefix(3).enumerated()), id: \.element.id) { index, event in
                        if index > 0 {
                            Divider().padding(.leading, 20)
                        }
                        EventRow(event: event, users: viewModel.users, timeFormatter: Self.timeFormatter)
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
            SectionHeaderLabel(text: "Today's Meals")
            VStack(spacing: 0) {
                ForEach(Array(["breakfast", "lunch", "dinner"].enumerated()), id: \.element) { index, mealType in
                    let plan = stats.todayMeals.first(where: { $0.mealType == mealType })
                    if index > 0 {
                        Divider().padding(.leading, 80)
                    }
                    if plan?.recipeID != nil {
                        RecipeMealRow(
                            mealType: mealType,
                            plan: plan!,
                            apiClient: apiClient,
                            recipesViewModel: recipesViewModel,
                            onEdit: {
                                editingMeal = EditingMeal(date: todayKey, mealType: mealType, name: plan!.name, recipeID: plan!.recipeID)
                            },
                            thumbSize: 52,
                            thumbCornerRadius: 12,
                            nameFontSize: 17,
                            nameFontWeight: .medium,
                            minRowHeight: 64
                        )
                    } else {
                        Button {
                            editingMeal = EditingMeal(date: todayKey, mealType: mealType, name: plan?.name ?? "", recipeID: nil)
                        } label: {
                            MealSlotRow(slot: mealType.capitalized, mealPlan: plan, apiClient: apiClient)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .glassCard(radius: 18)
            .padding(.horizontal, 16)
        }
    }

    // MARK: - Today's Chores

    private func choresSection(_ stats: DashboardStats) -> some View {
        let allDue = stats.choresOverdueList + stats.choresDueTodayList
        return VStack(alignment: .leading, spacing: 0) {
            HStack {
                SectionHeaderLabel(text: "Today's chores")
                Spacer()
                NavigationLink {
                    ChoresListView(apiClient: apiClient)
                } label: {
                    Text("Manage")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(.secondary)
                }
                .padding(.trailing, 20)
                .padding(.top, 24)
                .padding(.bottom, 8)
            }

            if allDue.isEmpty {
                HStack {
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
                        ChoreRow(
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

// MARK: - EventRow

private struct EventRow: View {
    let event: CalendarEvent
    let users: [String: User]
    let timeFormatter: DateFormatter

    var body: some View {
        HStack(alignment: .center, spacing: 14) {
            RoundedRectangle(cornerRadius: 2)
                .fill(Color(hex: event.color) ?? .accentColor)
                .frame(width: 4, height: 36)

            VStack(alignment: .leading, spacing: 2) {
                if event.allDay {
                    Text("All day")
                        .font(.system(size: 13, weight: .semibold, design: .default))
                        .foregroundStyle(.secondary)
                } else {
                    Text(timeFormatter.string(from: event.startTime))
                        .font(.system(size: 15, weight: .semibold).monospacedDigit())
                    if let end = event.endTime {
                        Text(timeFormatter.string(from: end))
                            .font(.system(size: 12).monospacedDigit())
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .frame(width: 60, alignment: .leading)

            VStack(alignment: .leading, spacing: 2) {
                Text(event.title)
                    .font(.system(size: 16))
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

// MARK: - MealSlotRow

private struct MealSlotRow: View {
    let slot: String
    let mealPlan: MealPlan?
    let apiClient: any APIClientProtocol

    var body: some View {
        HStack(spacing: 14) {
            RecipeThumbView(
                recipeID: mealPlan?.recipeID,
                apiClient: apiClient,
                size: 52,
                cornerRadius: 12
            )

            VStack(alignment: .leading, spacing: 2) {
                Text(slot.uppercased())
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(.secondary)
                    .kerning(0.5)
                Text(mealPlan?.name ?? "Not planned")
                    .font(.system(size: 17, weight: .medium))
                    .foregroundStyle(mealPlan != nil ? .primary : .tertiary)
                    .lineLimit(1)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(minHeight: 64)
    }
}

// MARK: - ChoreRow

private struct ChoreRow: View {
    let chore: Chore
    let user: User?
    let apiClient: any APIClientProtocol
    let onComplete: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            UserAvatar(user: user, size: 32, apiClient: apiClient)

            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.system(size: 16, weight: .medium))
                    .strikethrough(chore.status == .completed)
                    .foregroundStyle(chore.status == .completed ? .tertiary : .primary)
                    .lineLimit(1)
                HStack(spacing: 4) {
                    if let user {
                        Text(user.name)
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                        Text("·")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                    if chore.status == .overdue {
                        Text("Overdue")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(.red)
                    } else if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer(minLength: 0)

            Button(action: onComplete) {
                ZStack {
                    Circle()
                        .strokeBorder(Color.appGreen, lineWidth: 1.5)
                        .frame(width: 28, height: 28)
                        .opacity(chore.status == .completed ? 0 : 1)
                    Circle()
                        .fill(Color.appGreen)
                        .frame(width: 28, height: 28)
                        .opacity(chore.status == .completed ? 1 : 0)
                    Image(systemName: "checkmark")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(.white)
                        .opacity(chore.status == .completed ? 1 : 0)
                }
            }
            .buttonStyle(.plain)
            .sensoryFeedback(.success, trigger: chore.status == .completed)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(minHeight: 56)
    }
}

