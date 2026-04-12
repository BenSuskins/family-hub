import SwiftUI

struct CalendarView: View {
    @State private var viewModel: CalendarViewModel
    private let apiClient: any APIClientProtocol
    private let columns = Array(repeating: GridItem(.flexible()), count: 7)
    private let weekdaySymbols = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

    private static let monthFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "MMMM yyyy"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let weekTitleFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d MMM"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dayTitleFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "EEEE, d MMMM"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: CalendarViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("View", selection: $viewModel.viewMode) {
                    ForEach(CalendarViewMode.allCases, id: \.self) { mode in
                        Text(mode.rawValue).tag(mode)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.vertical, 8)

                Group {
                    switch viewModel.viewMode {
                    case .month:
                        monthView
                    case .week:
                        weekView
                    case .day:
                        dayView
                    }
                }
                .animation(.spring(duration: 0.3), value: viewModel.viewMode)
            }
            .refreshable { await viewModel.load(forceRefresh: true) }
            .navigationTitle(navigationTitle)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button { navigateBack() } label: {
                        Image(systemName: "chevron.left")
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    HStack(spacing: 16) {
                        Button("Today") { viewModel.goToToday() }
                            .font(.subheadline)
                        Button { navigateForward() } label: {
                            Image(systemName: "chevron.right")
                        }
                    }
                }
            }
        }
        .task { await viewModel.load() }
        .onChange(of: viewModel.viewMode) {
            Task { await viewModel.load() }
        }
    }

    // MARK: - Navigation

    private var navigationTitle: String {
        switch viewModel.viewMode {
        case .month:
            return Self.monthFormatter.string(from: viewModel.currentDate)
        case .week:
            let start = viewModel.currentWeekStart
            let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
            return "\(Self.weekTitleFormatter.string(from: start)) – \(Self.weekTitleFormatter.string(from: end))"
        case .day:
            return Self.dayTitleFormatter.string(from: viewModel.currentDate)
        }
    }

    private func navigateBack() {
        switch viewModel.viewMode {
        case .month: viewModel.previousMonth()
        case .week: viewModel.previousWeek()
        case .day: viewModel.previousDay()
        }
    }

    private func navigateForward() {
        switch viewModel.viewMode {
        case .month: viewModel.nextMonth()
        case .week: viewModel.nextWeek()
        case .day: viewModel.nextDay()
        }
    }

    // MARK: - Month View

    private var monthView: some View {
        ScrollView {
            VStack(spacing: 0) {
                if case .failed(let error) = viewModel.state {
                    Text(error.localizedDescription)
                        .font(.caption)
                        .foregroundStyle(.red)
                        .padding()
                }
                calendarGrid
                    .padding(.horizontal)
                Divider()
                agendaSection
            }
        }
    }

    private var calendarGrid: some View {
        VStack(spacing: 4) {
            LazyVGrid(columns: columns, spacing: 0) {
                ForEach(Array(weekdaySymbols.enumerated()), id: \.offset) { _, symbol in
                    Text(symbol)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.tertiary)
                        .frame(maxWidth: .infinity)
                }
            }
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(Array(daysInMonth.enumerated()), id: \.offset) { _, day in
                    if let day {
                        DayCell(
                            date: day,
                            isSelected: Calendar.current.isDate(day, inSameDayAs: viewModel.selectedDay ?? .distantPast),
                            isToday: Calendar.current.isDateInToday(day),
                            hasItems: viewModel.hasItems(for: day)
                        )
                        .onTapGesture { viewModel.selectedDay = day }
                    } else {
                        Color.clear.frame(height: 36)
                    }
                }
            }
        }
        .padding(.vertical, 10)
    }

    // MARK: - Week View

    private var weekView: some View {
        List {
            ForEach(0..<7, id: \.self) { offset in
                let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeekStart)!
                Section(Self.dayTitleFormatter.string(from: date)) {
                    dayItemsContent(for: date)
                }
            }
        }
        .listStyle(.insetGrouped)
    }

    // MARK: - Day View

    private var dayView: some View {
        List {
            dayItemsContent(for: viewModel.currentDate)
        }
        .listStyle(.insetGrouped)
    }

    // MARK: - Shared Day Items

    @ViewBuilder
    private func dayItemsContent(for date: Date) -> some View {
        let chores = viewModel.chores(for: date)
        let events = viewModel.events(for: date)
        let meals = viewModel.meals(for: date)

        if chores.isEmpty && events.isEmpty && meals.isEmpty {
            Text("No items")
                .foregroundStyle(.secondary)
        }

        ForEach(events) { event in
            eventRow(event)
        }

        ForEach(chores) { chore in
            choreRow(chore)
        }

        ForEach(meals) { meal in
            mealRow(meal)
        }
    }

    private func eventRow(_ event: CalendarEvent) -> some View {
        HStack(spacing: 10) {
            RoundedRectangle(cornerRadius: 2)
                .fill(Color(hex: event.color) ?? Color.accentColor)
                .frame(width: 4, height: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(event.title)
                    .font(.body)
                HStack(spacing: 6) {
                    if event.allDay {
                        Text("All Day")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        Text(event.startTime, style: .time)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if !event.location.isEmpty {
                        Label(event.location, systemImage: "mappin")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                }
            }
            Spacer()
            Image(systemName: "calendar.badge.clock")
                .foregroundStyle(.secondary)
                .font(.caption)
        }
    }

    private func choreRow(_ chore: Chore) -> some View {
        HStack(spacing: 10) {
            RoundedRectangle(cornerRadius: 2)
                .fill(chore.status == .overdue ? .red : .orange)
                .frame(width: 4, height: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.body)
                HStack(spacing: 6) {
                    if let badge = chore.badge {
                        Text(badge.label)
                            .font(.caption)
                            .foregroundStyle(badge.color)
                    }
                    if let userId = chore.assignedToUserID, let user = viewModel.users[userId] {
                        Text(user.name)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            Spacer()
            if let userId = chore.assignedToUserID {
                UserAvatar(user: viewModel.users[userId], size: 24, apiClient: apiClient)
            }
        }
    }

    private func mealRow(_ meal: MealPlan) -> some View {
        HStack(spacing: 10) {
            RoundedRectangle(cornerRadius: 2)
                .fill(Color.green)
                .frame(width: 4, height: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(meal.name)
                    .font(.body)
                Text(meal.mealType.capitalized)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Image(systemName: "fork.knife")
                .foregroundStyle(.secondary)
                .font(.caption)
        }
    }

    // MARK: - Agenda (Month mode)

    private var agendaSection: some View {
        Group {
            if let selectedDay = viewModel.selectedDay {
                let chores = viewModel.chores(for: selectedDay)
                let events = viewModel.events(for: selectedDay)
                let meals = viewModel.meals(for: selectedDay)
                if chores.isEmpty && events.isEmpty && meals.isEmpty {
                    ContentUnavailableView(
                        "No items on this day",
                        systemImage: "calendar",
                        description: Text("All clear!")
                    )
                } else {
                    LazyVStack(spacing: 0) {
                        ForEach(events) { event in
                            eventRow(event)
                                .padding(.horizontal)
                                .padding(.vertical, 8)
                            Divider()
                        }
                        ForEach(chores) { chore in
                            choreRow(chore)
                                .padding(.horizontal)
                                .padding(.vertical, 8)
                            Divider()
                        }
                        ForEach(meals) { meal in
                            mealRow(meal)
                                .padding(.horizontal)
                                .padding(.vertical, 8)
                            Divider()
                        }
                    }
                }
            } else {
                ContentUnavailableView("Select a day", systemImage: "calendar")
            }
        }
    }

    // MARK: - Month Grid Data

    private var daysInMonth: [Date?] {
        let calendar = Calendar(identifier: .iso8601)
        guard let range = calendar.range(of: .day, in: .month, for: viewModel.currentMonth),
              let firstDay = calendar.date(from: calendar.dateComponents([.year, .month], from: viewModel.currentMonth))
        else { return [] }
        let weekdayOffset = (calendar.component(.weekday, from: firstDay) + 5) % 7
        var days: [Date?] = Array(repeating: nil, count: weekdayOffset)
        days += range.compactMap { day in
            calendar.date(byAdding: .day, value: day - 1, to: firstDay)
        }
        return days
    }
}

// MARK: - Supporting Views

private struct DayCell: View {
    let date: Date
    let isSelected: Bool
    let isToday: Bool
    let hasItems: Bool

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    var body: some View {
        VStack(spacing: 2) {
            Text(Self.dayFormatter.string(from: date))
                .font(.subheadline)
                .foregroundStyle(isSelected ? .white : .primary)
                .frame(width: 30, height: 30)
                .background(isSelected ? Color.accentColor : (isToday ? Color.accentColor.opacity(0.15) : Color.clear))
                .clipShape(Circle())
            Circle()
                .fill(hasItems ? .orange : Color.clear)
                .frame(width: 4, height: 4)
        }
    }
}

// MARK: - Color from hex

extension Color {
    init?(hex: String) {
        let cleaned = hex.trimmingCharacters(in: .whitespacesAndNewlines).replacingOccurrences(of: "#", with: "")
        guard cleaned.count == 6, let rgb = UInt64(cleaned, radix: 16) else { return nil }
        self.init(
            red: Double((rgb >> 16) & 0xFF) / 255,
            green: Double((rgb >> 8) & 0xFF) / 255,
            blue: Double(rgb & 0xFF) / 255
        )
    }
}
