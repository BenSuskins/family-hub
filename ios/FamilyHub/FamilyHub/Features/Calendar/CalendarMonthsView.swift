import SwiftUI
import FamilyHubKit

struct CalendarMonthsView: View {
    let viewModel: CalendarViewModel
    let apiClient: any APIClientProtocol
    @State private var months: [Date] = []
    @State private var visibleMonth: Date?
    @State private var scrollPhase: ScrollPhase = .idle
    @State private var isZooming = false
    @GestureState private var magnification = 1.0
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    init(viewModel: CalendarViewModel, apiClient: any APIClientProtocol) {
        self.viewModel = viewModel
        self.apiClient = apiClient
        _months = State(initialValue: (-12...12).map {
            viewModel.schedule.adding(.month, value: $0, to: viewModel.currentMonth)
        })
        _visibleMonth = State(initialValue: viewModel.currentMonth)
    }

    private var rowHeight: Double {
        switch viewModel.monthDisplay {
        case .compact, .list: 48
        case .stacked: 64 * viewModel.monthScale * magnification
        case .details: 98 * viewModel.monthScale * magnification
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            CalendarWeekdayHeader(calendar: viewModel.schedule.calendar)
            if viewModel.monthDisplay == .list {
                ScrollView {
                    CalendarMonthGrid(viewModel: viewModel, month: viewModel.currentMonth,
                                      rowHeight: rowHeight, showsTitle: true)
                    Divider()
                    CalendarAgendaView(viewModel: viewModel, day: viewModel.selectedDate, apiClient: apiClient)
                        .padding(.horizontal)
                }
                .refreshable { await viewModel.load(forceRefresh: true) }
                .gesture(DragGesture(minimumDistance: 40).onEnded { value in
                    if abs(value.translation.width) > abs(value.translation.height) {
                        viewModel.move(value.translation.width < 0 ? 1 : -1)
                    }
                })
            } else {
                scrollingMonths
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("calendar-months")
    }

    private var scrollingMonths: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(spacing: 16) {
                    ForEach(months, id: \.self) { month in
                        CalendarMonthGrid(viewModel: viewModel, month: month,
                                          rowHeight: max(44, rowHeight), showsTitle: true)
                            .id(month)
                    }
                }.scrollTargetLayout()
            }
            .scrollPosition(id: $visibleMonth, anchor: .top)
            .scrollDisabled(isZooming)
            .onScrollPhaseChange { _, phase in scrollPhase = phase }
            .task(id: "\(viewModel.monthDisplay)-\(viewModel.monthScale)") {
                await Task.yield()
                proxy.scrollTo(viewModel.currentMonth, anchor: .top)
            }
            .onChange(of: viewModel.navigationRevision) { _, _ in
                proxy.scrollTo(viewModel.currentMonth, anchor: .top)
            }
            .onChange(of: viewModel.currentMonth) { _, month in
                guard visibleMonth != month else { return }
                if !months.contains(month) {
                    months = (-12...12).map { viewModel.schedule.adding(.month, value: $0, to: month) }
                }
                withAnimation(reduceMotion ? nil : .snappy) { proxy.scrollTo(month, anchor: .top) }
            }
            .onChange(of: visibleMonth) { _, month in
                guard let month, !isZooming,
                      scrollPhase == .interacting || scrollPhase == .decelerating else { return }
                if month != viewModel.currentMonth { viewModel.selectDate(month) }
                if let first = months.first, month <= months[min(2, months.count - 1)] {
                    months.insert(contentsOf: (-12 ... -1).map { viewModel.schedule.adding(.month, value: $0, to: first) }, at: 0)
                }
                if let last = months.last, month >= months[max(0, months.count - 3)] {
                    months += (1...12).map { viewModel.schedule.adding(.month, value: $0, to: last) }
                }
            }
            .simultaneousGesture(MagnifyGesture().updating($magnification) { value, state, _ in
                state = value.magnification
            }.onChanged { _ in
                isZooming = true
            }.onEnded { value in
                isZooming = false
                if viewModel.monthDisplay == .compact, value.magnification > 1.15 {
                    viewModel.setMonthDisplay(.stacked)
                } else if viewModel.monthDisplay == .stacked, value.magnification > 1.15 {
                    viewModel.setMonthDisplay(.details)
                } else if viewModel.monthDisplay == .stacked, value.magnification < 0.85 {
                    viewModel.setMonthDisplay(.compact)
                } else if viewModel.monthDisplay == .details, value.magnification < 0.7, viewModel.monthScale <= 0.8 {
                    viewModel.setMonthDisplay(.stacked)
                } else {
                    viewModel.setMonthScale(viewModel.monthScale * value.magnification)
                }
            })
            .refreshable { await viewModel.load(forceRefresh: true) }
        }
    }
}

struct CalendarWeekdayHeader: View {
    let calendar: Calendar

    var body: some View {
        HStack(spacing: 0) {
            ForEach(0..<7, id: \.self) { index in
                let weekday = (calendar.firstWeekday - 1 + index) % 7
                Text(calendar.veryShortStandaloneWeekdaySymbols[weekday])
                    .font(.caption.weight(.medium)).foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .accessibilityLabel(calendar.weekdaySymbols[weekday])
            }
        }.padding(.vertical, 8)
    }
}

private struct CalendarMonthGrid: View {
    let viewModel: CalendarViewModel
    let month: Date
    let rowHeight: Double
    let showsTitle: Bool
    private let columns = Array(repeating: GridItem(.flexible(), spacing: 0), count: 7)

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if showsTitle {
                Text(month.formatted(.dateTime.month(.wide)))
                    .font(.largeTitle.bold()).foregroundStyle(.red)
                    .padding(.horizontal, 16).frame(height: 60)
                    .accessibilityAddTraits(.isHeader)
            }
            LazyVGrid(columns: columns, spacing: 0) {
                ForEach(Array(viewModel.schedule.monthDays(containing: month).enumerated()), id: \.offset) { _, date in
                    if let date {
                        CalendarMonthDay(viewModel: viewModel, date: date, height: rowHeight)
                    } else {
                        Color.clear.frame(height: rowHeight)
                    }
                }
            }
        }
        .frame(height: (showsTitle ? 60 : 0) + Double(viewModel.schedule.monthDays(containing: month).count / 7) * rowHeight)
    }
}

private struct CalendarMonthDay: View {
    let viewModel: CalendarViewModel
    let date: Date
    let height: Double

    private var items: [CalendarDisplayItem] { viewModel.items(for: date) }
    private var isToday: Bool { viewModel.schedule.calendar.isDate(date, inSameDayAs: viewModel.today) }
    private var isSelected: Bool { viewModel.schedule.calendar.isDate(date, inSameDayAs: viewModel.selectedDate) }
    private var limit: Int { max(1, Int((height - 36) / 16)) }

    var body: some View {
        Button {
            if viewModel.monthDisplay == .list { viewModel.selectDate(date) }
            else { viewModel.openDay(date) }
        } label: {
            VStack(spacing: 2) {
                Text(date.formatted(.dateTime.day()))
                    .font(.body)
                    .foregroundStyle(isToday || (isSelected && viewModel.monthDisplay == .list) ? Color.white : Color.primary)
                    .frame(width: 30, height: 30)
                    .background(isToday ? Color.red : isSelected && viewModel.monthDisplay == .list ? Color.primary : .clear, in: Circle())
                switch viewModel.monthDisplay {
                case .compact, .list:
                    Capsule().fill(items.first?.color ?? .clear).frame(width: 24, height: 4)
                case .stacked:
                    ForEach(items.prefix(limit)) { item in
                        Capsule().fill(item.color).frame(height: 4).padding(.horizontal, 4)
                    }
                case .details:
                    ForEach(items.prefix(limit)) { item in
                        HStack(spacing: 1) {
                            if let symbol = item.symbol { Image(systemName: symbol).font(.system(size: 8)) }
                            Text(item.title).font(.system(size: 10, weight: .medium)).lineLimit(1)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 2).padding(.vertical, 1)
                        .foregroundStyle(.primary)
                        .background(item.color.opacity(0.13), in: RoundedRectangle(cornerRadius: 3))
                    }
                }
                if items.count > limit, viewModel.monthDisplay == .details || viewModel.monthDisplay == .stacked {
                    Text("+\(items.count - limit)").font(.system(size: 9)).foregroundStyle(.secondary)
                }
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 2).padding(.top, 2)
            .frame(maxWidth: .infinity).frame(height: height)
            .clipped()
            .contentShape(Rectangle())
            .overlay(alignment: .top) {
                Rectangle().fill(Color(.separator).opacity(0.5)).frame(height: 0.5)
            }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(date.formatted(date: .complete, time: .omitted))
        .accessibilityValue("\(items.count) items" + (isToday ? ", Today" : ""))
        .accessibilityHint(viewModel.monthDisplay == .list ? "Show items for this date" : "Open day timeline")
        .accessibilityIdentifier("calendar-date-\(viewModel.schedule.dayKey(date))")
    }
}

struct CalendarYearView: View {
    let viewModel: CalendarViewModel
    @State private var years: [Int] = []
    @State private var visibleYear: Int?
    @State private var scrollPhase: ScrollPhase = .idle
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    init(viewModel: CalendarViewModel) {
        self.viewModel = viewModel
        let year = viewModel.schedule.calendar.component(.year, from: viewModel.selectedDate)
        _years = State(initialValue: Array((year - 3)...(year + 3)))
        _visibleYear = State(initialValue: year)
    }

    private var currentYear: Int { viewModel.schedule.calendar.component(.year, from: viewModel.selectedDate) }

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    ForEach(years, id: \.self) { year in
                        VStack(alignment: .leading, spacing: 20) {
                            Text(String(year)).font(.largeTitle.bold()).foregroundStyle(.red).accessibilityAddTraits(.isHeader)
                            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 16), count: dynamicTypeSize.isAccessibilitySize ? 1 : 3), spacing: 24) {
                                ForEach(1...12, id: \.self) { month in
                                    if let date = viewModel.schedule.calendar.date(from: DateComponents(year: year, month: month, day: 1)) {
                                        miniMonth(date)
                                    }
                                }
                            }
                        }.id(year)
                    }
                }.padding().scrollTargetLayout()
            }
            .scrollPosition(id: $visibleYear, anchor: .top)
            .onScrollPhaseChange { _, phase in scrollPhase = phase }
            .task {
                await Task.yield()
                proxy.scrollTo(currentYear, anchor: .top)
            }
            .onChange(of: viewModel.navigationRevision) { _, _ in
                proxy.scrollTo(currentYear, anchor: .top)
            }
            .onChange(of: currentYear) { _, year in
                if !years.contains(year) { years = Array((year - 3)...(year + 3)) }
                proxy.scrollTo(year, anchor: .top)
            }
            .onChange(of: visibleYear) { _, year in
                guard let year, scrollPhase == .interacting || scrollPhase == .decelerating else { return }
                if year != currentYear,
                   let date = viewModel.schedule.calendar.date(from: DateComponents(year: year, month: 1, day: 1)) {
                    viewModel.selectDate(date)
                }
                if year == years.first { years.insert(contentsOf: (year - 3)..<year, at: 0) }
                if year == years.last { years += Array((year + 1)...(year + 3)) }
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("calendar-years")
    }

    private func miniMonth(_ date: Date) -> some View {
        Button { viewModel.openMonth(date) } label: {
            VStack(alignment: .leading, spacing: 8) {
                Text(date.formatted(.dateTime.month(.wide))).font(.subheadline.bold()).foregroundStyle(.red)
                LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 0), count: 7), spacing: 5) {
                    ForEach(Array(viewModel.schedule.monthDays(containing: date).enumerated()), id: \.offset) { _, day in
                        if let day {
                            Text(day.formatted(.dateTime.day())).font(.system(size: 9))
                                .frame(maxWidth: .infinity)
                                .foregroundStyle(viewModel.schedule.calendar.isDate(day, inSameDayAs: viewModel.today) ? .red : .primary)
                        } else { Text(" ").font(.system(size: 9)) }
                    }
                }
            }.frame(height: 132, alignment: .top).contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(date.formatted(.dateTime.month(.wide).year()))
        .accessibilityHint("Open month")
    }
}
