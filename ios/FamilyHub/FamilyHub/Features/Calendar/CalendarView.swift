// ios/FamilyHub/Features/Calendar/CalendarView.swift
import SwiftUI

struct CalendarView: View {
    @State private var viewModel: CalendarViewModel

    private let columns = Array(repeating: GridItem(.flexible()), count: 7)
    private let weekdaySymbols = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: CalendarViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                calendarGrid
                Divider()
                agendaList
            }
            .navigationTitle(monthTitle)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("< Prev") { viewModel.previousMonth() }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Next >") { viewModel.nextMonth() }
                }
            }
        }
        .task { await viewModel.load() }
    }

    private var monthTitle: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "MMMM yyyy"
        return formatter.string(from: viewModel.currentMonth)
    }

    private var calendarGrid: some View {
        VStack(spacing: 4) {
            // Weekday headers
            LazyVGrid(columns: columns, spacing: 0) {
                ForEach(Array(weekdaySymbols.enumerated()), id: \.offset) { _, symbol in
                    Text(symbol)
                        .font(.caption.bold())
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity)
                }
            }
            // Day cells
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(daysInMonth, id: \.self) { day in
                    DayCell(
                        date: day,
                        isSelected: Calendar.current.isDate(day, inSameDayAs: viewModel.selectedDay ?? .distantPast),
                        hasChores: !viewModel.chores(for: day).isEmpty
                    )
                    .onTapGesture {
                        viewModel.selectedDay = day
                    }
                }
            }
        }
        .padding()
    }

    private var agendaList: some View {
        Group {
            if let selectedDay = viewModel.selectedDay {
                let chores = viewModel.chores(for: selectedDay)
                if chores.isEmpty {
                    ContentUnavailableView("No chores", systemImage: "checkmark.circle")
                        .frame(maxHeight: .infinity)
                } else {
                    List(chores) { chore in
                        Text(chore.name)
                    }
                }
            } else {
                ContentUnavailableView("Select a day", systemImage: "calendar")
                    .frame(maxHeight: .infinity)
            }
        }
    }

    private var daysInMonth: [Date] {
        let calendar = Calendar(identifier: .iso8601)
        guard let range = calendar.range(of: .day, in: .month, for: viewModel.currentMonth),
              let firstDay = calendar.date(from: calendar.dateComponents([.year, .month], from: viewModel.currentMonth))
        else { return [] }

        // Pad with leading empty days (offset from Monday)
        let weekdayOffset = (calendar.component(.weekday, from: firstDay) + 5) % 7
        var days: [Date] = Array(repeating: Date.distantPast, count: weekdayOffset)
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

    var body: some View {
        let isPlaceholder = date == .distantPast
        VStack(spacing: 2) {
            Text(isPlaceholder ? "" : dayNumber)
                .font(.callout)
                .foregroundStyle(isSelected ? .white : .primary)
                .frame(width: 32, height: 32)
                .background(isSelected ? Color.blue : Color.clear)
                .clipShape(Circle())

            Circle()
                .fill(hasChores && !isPlaceholder ? Color.blue : Color.clear)
                .frame(width: 5, height: 5)
        }
    }

    private var dayNumber: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "d"
        return formatter.string(from: date)
    }
}
