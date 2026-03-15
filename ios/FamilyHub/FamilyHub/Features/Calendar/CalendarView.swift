import SwiftUI

struct CalendarView: View {
    @State private var viewModel: CalendarViewModel
    private let columns = Array(repeating: GridItem(.flexible()), count: 7)
    private let weekdaySymbols = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

    private static let monthFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "MMMM yyyy"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: CalendarViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                ScrollView {
                    VStack(spacing: 0) {
                        if case .failed(let error) = viewModel.state {
                            Text(error.localizedDescription)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.statusRed)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                        }
                        calendarGrid
                            .padding(.horizontal, 14)
                        Rectangle().fill(Theme.borderDivider).frame(height: 1)
                        agendaSection
                    }
                }
                .refreshable { await viewModel.load() }
            }
            .navigationTitle(Self.monthFormatter.string(from: viewModel.currentMonth))
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button { viewModel.previousMonth() } label: {
                        Image(systemName: "chevron.left").foregroundStyle(Theme.accent)
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { viewModel.nextMonth() } label: {
                        Image(systemName: "chevron.right").foregroundStyle(Theme.accent)
                    }
                }
            }
        }
        .task { await viewModel.load() }
    }

    private var calendarGrid: some View {
        VStack(spacing: 4) {
            LazyVGrid(columns: columns, spacing: 0) {
                ForEach(Array(weekdaySymbols.enumerated()), id: \.offset) { _, symbol in
                    Text(symbol)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Theme.textMuted)
                        .frame(maxWidth: .infinity)
                }
            }
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(Array(daysInMonth.enumerated()), id: \.offset) { _, day in
                    if let day {
                        DayCell(
                            date: day,
                            isSelected: Calendar.current.isDate(day, inSameDayAs: viewModel.selectedDay ?? .distantPast),
                            hasChores: !viewModel.chores(for: day).isEmpty
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

    private var agendaSection: some View {
        Group {
            if let selectedDay = viewModel.selectedDay {
                let chores = viewModel.chores(for: selectedDay)
                if chores.isEmpty {
                    ContentUnavailableView(
                        "No chores on this day",
                        systemImage: "calendar",
                        description: Text("All clear!")
                    )
                    .frame(maxHeight: .infinity)
                } else {
                    List(chores) { chore in
                        HStack(spacing: 10) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(chore.name)
                                    .font(.system(size: 15, weight: .medium))
                                    .foregroundStyle(Theme.textPrimary)
                                if let badge = chore.badgeVariant {
                                    StatusBadge(variant: badge)
                                }
                            }
                            Spacer()
                        }
                        .padding(.vertical, 4)
                        .listRowBackground(Theme.surface)
                        .listRowSeparatorTint(Theme.borderDivider)
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                }
            } else {
                ContentUnavailableView("Select a day", systemImage: "calendar")
                    .frame(maxHeight: .infinity)
            }
        }
    }

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

private struct DayCell: View {
    let date: Date
    let isSelected: Bool
    let hasChores: Bool

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    var body: some View {
        VStack(spacing: 2) {
            Text(Self.dayFormatter.string(from: date))
                .font(.system(size: 14))
                .foregroundStyle(isSelected ? .white : Theme.textPrimary)
                .frame(width: 30, height: 30)
                .background(isSelected ? Theme.accent : Color.clear)
                .clipShape(Circle())
            Circle()
                .fill(hasChores ? Theme.statusAmber : Color.clear)
                .frame(width: 4, height: 4)
        }
    }
}
