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
            .refreshable { await viewModel.load() }
            .navigationTitle(Self.monthFormatter.string(from: viewModel.currentMonth))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button { viewModel.previousMonth() } label: {
                        Image(systemName: "chevron.left")
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { viewModel.nextMonth() } label: {
                        Image(systemName: "chevron.right")
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
                } else {
                    List(chores) { chore in
                        HStack(spacing: 10) {
                            RoundedRectangle(cornerRadius: 2)
                                .fill(chore.status == .overdue ? .red : Color.accentColor)
                                .frame(width: 4, height: 32)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(chore.name)
                                    .font(.body)
                                if let badge = chore.badge {
                                    Text(badge.label)
                                        .font(.caption)
                                        .foregroundStyle(badge.color)
                                }
                            }
                            Spacer()
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            } else {
                ContentUnavailableView("Select a day", systemImage: "calendar")
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
    let isToday: Bool
    let hasChores: Bool

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
                .fill(hasChores ? .orange : Color.clear)
                .frame(width: 4, height: 4)
        }
    }
}
