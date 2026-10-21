import SwiftUI
import FamilyHubKit

struct CalendarTimelineView: View {
    let viewModel: CalendarViewModel
    let apiClient: any APIClientProtocol
    @GestureState private var magnification = 1.0
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var allDaySheetDate: Date?
    private let axisWidth: CGFloat = 48

    private var hourHeight: CGFloat { min(120, max(44, viewModel.hourHeight * magnification)) }
    private var days: [Date] { viewModel.timelineDays }

    var body: some View {
        GeometryReader { geometry in
            VStack(spacing: 0) {
                dayStrip(width: geometry.size.width)
                untimedSection
                Divider()
                timeline
            }
            .frame(width: geometry.size.width, height: geometry.size.height, alignment: .top)
        }
        .simultaneousGesture(DragGesture(minimumDistance: 40).onEnded { value in
            if abs(value.translation.width) > abs(value.translation.height) * 1.5 {
                withAnimation(reduceMotion ? nil : .snappy) {
                    viewModel.move(value.translation.width < 0 ? 1 : -1)
                }
            }
        })
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("calendar-timeline")
        .sheet(isPresented: Binding(
            get: { allDaySheetDate != nil },
            set: { if !$0 { allDaySheetDate = nil } }
        )) {
            if let allDaySheetDate {
                AllDayItemsSheet(viewModel: viewModel, day: allDaySheetDate, apiClient: apiClient)
                    .presentationDetents([.medium, .large])
                    .presentationDragIndicator(.visible)
            }
        }
    }

    private func dayStrip(width: CGFloat) -> some View {
        HStack(spacing: 0) {
            if viewModel.viewMode == .day {
                ForEach(0..<7, id: \.self) { offset in
                    let date = viewModel.schedule.adding(.day, value: offset, to: viewModel.currentWeekStart)
                    dateButton(date).frame(width: width / 7)
                }
            } else {
                Color.clear.frame(width: axisWidth, height: 1)
                ForEach(days, id: \.self) { date in
                    dateButton(date).frame(width: (width - axisWidth) / CGFloat(days.count))
                }
            }
        }.frame(width: width, height: 62)
    }

    private func dateButton(_ date: Date) -> some View {
        let calendar = viewModel.schedule.calendar
        let selected = calendar.isDate(date, inSameDayAs: viewModel.selectedDate)
        return Button { viewModel.selectDate(date) } label: {
            VStack(spacing: 3) {
                Text(date.formatted(.dateTime.weekday(.narrow))).font(.caption2).foregroundStyle(.secondary)
                Text(date.formatted(.dateTime.day())).font(.title3)
                    .frame(width: 34, height: 34)
                    .foregroundStyle(selected ? AppTheme.onAccent : AppTheme.ink)
                    .background(selected ? AppTheme.accent : .clear, in: Circle())
            }.frame(maxWidth: .infinity, minHeight: 44)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(date.formatted(date: .complete, time: .omitted))
        .accessibilityAddTraits(selected ? .isSelected : [])
    }

    @ViewBuilder
    private var untimedSection: some View {
        if viewModel.viewMode == .week && horizontalSizeClass == .compact {
            compactAllDaySection
        } else {
            expandedAllDaySection
        }
    }

    private var compactAllDaySection: some View {
        HStack(spacing: 0) {
            Text("All day")
                .font(.caption2)
                .foregroundStyle(AppTheme.secondary)
                .frame(width: axisWidth)
            GeometryReader { geometry in
                HStack(spacing: 1) {
                    ForEach(days, id: \.self) { day in
                        let items = viewModel.items(for: day, untimedOnly: true)
                        Button {
                            guard !items.isEmpty else { return }
                            allDaySheetDate = day
                        } label: {
                            VStack(alignment: .leading, spacing: 1) {
                                if let first = items.first {
                                    Text(first.title)
                                        .font(.system(size: 9, weight: .medium))
                                        .lineLimit(1)
                                        .truncationMode(.tail)
                                    if items.count > 1 {
                                        Text("+\(items.count - 1) ⌄")
                                            .font(.system(size: 9, weight: .semibold))
                                            .foregroundStyle(AppTheme.accent)
                                    }
                                }
                            }
                            .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                            .padding(.horizontal, 2)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .disabled(items.isEmpty)
                        .accessibilityLabel("\(day.formatted(date: .complete, time: .omitted)), \(items.count) all-day items")
                        .accessibilityHint(items.isEmpty ? "No all-day items" : "Show all-day items")
                        .accessibilityIdentifier("calendar-all-day-count-\(viewModel.schedule.dayKey(day))")
                        .frame(width: max(1, (geometry.size.width - CGFloat(days.count - 1)) / CGFloat(days.count)))
                    }
                }
            }
            .frame(height: 44)
        }
        .background(AppTheme.sage.opacity(0.55))
    }

    private var expandedAllDaySection: some View {
        HStack(alignment: .top, spacing: 0) {
            Text("All day").font(.caption2).foregroundStyle(.secondary)
                .frame(width: axisWidth).padding(.top, 8)
            GeometryReader { geometry in
                ScrollView(.vertical) {
                    HStack(alignment: .top, spacing: 1) {
                        ForEach(days, id: \.self) { day in
                            VStack(alignment: .leading, spacing: 4) {
                                let items = viewModel.items(for: day, untimedOnly: true)
                                if items.isEmpty { Text(" ").font(.caption) }
                                ForEach(items) { item in
                                    if case .event(let event) = item.content {
                                        NavigationLink { EventDetailsView(event: event, apiClient: apiClient) } label: {
                                            untimedLabel(item)
                                        }.buttonStyle(.plain)
                                    } else { untimedLabel(item) }
                                }
                            }
                            .frame(width: max(1, (geometry.size.width - CGFloat(days.count - 1)) / CGFloat(days.count)), alignment: .topLeading)
                        }
                    }.padding(.vertical, 6)
                }
            }
            .frame(height: untimedHeight)
        }
        .background(AppTheme.sage.opacity(0.3))
    }

    private var untimedHeight: CGFloat {
        let count = days.map { viewModel.items(for: $0, untimedOnly: true).count }.max() ?? 0
        return min(160, max(36, CGFloat(count) * 46 + 12))
    }

    private func untimedLabel(_ item: CalendarDisplayItem) -> some View {
        HStack(alignment: .top, spacing: 3) {
            if let symbol = item.symbol { Image(systemName: symbol).accessibilityHidden(true) }
            Text(item.title).lineLimit(2)
        }
        .font(.caption)
        .padding(5)
        .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
        .foregroundStyle(.primary)
        .background(item.color.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("\(item.kind), \(item.title)")
    }

    private var timeline: some View {
        ScrollViewReader { proxy in
            ScrollView {
                ZStack(alignment: .topLeading) {
                    VStack(spacing: 0) {
                        ForEach(0..<24, id: \.self) { hour in
                            Color.clear.frame(height: hourHeight).id(hour)
                        }
                    }
                    GeometryReader { geometry in
                        ZStack(alignment: .topLeading) {
                            hourLines(width: geometry.size.width)
                            HStack(alignment: .top, spacing: 0) {
                                Color.clear.frame(width: axisWidth)
                                ForEach(days, id: \.self) { day in
                                    dayColumn(day)
                                        .frame(width: max(1, (geometry.size.width - axisWidth) / CGFloat(days.count)))
                                        .overlay(alignment: .leading) { Rectangle().fill(AppTheme.border.opacity(0.65)).frame(width: 0.5) }
                                }
                            }
                            currentTime(width: geometry.size.width)
                        }
                    }
                }
                .frame(height: hourHeight * 24 + 44)
            }
            .task {
                await Task.yield()
                proxy.scrollTo(initialHour, anchor: .top)
            }
            .onChange(of: viewModel.navigationRevision) { _, _ in
                proxy.scrollTo(initialHour, anchor: .top)
            }
            .simultaneousGesture(MagnifyGesture().updating($magnification) { value, state, _ in
                state = value.magnification
            }.onEnded { value in
                viewModel.setHourHeight(viewModel.hourHeight * value.magnification)
            })
            .refreshable { await viewModel.load(forceRefresh: true) }
        }
    }

    private var initialHour: Int {
        if days.contains(where: { viewModel.schedule.calendar.isDate($0, inSameDayAs: viewModel.today) }) {
            return max(0, viewModel.schedule.calendar.component(.hour, from: viewModel.today) - 1)
        }
        return 8
    }

    private func hourLines(width: CGFloat) -> some View {
        ZStack(alignment: .topLeading) {
            ForEach(0...24, id: \.self) { hour in
                HStack(spacing: 4) {
                    Text(hourLabel(hour % 24)).font(.system(size: 10)).foregroundStyle(.secondary)
                        .frame(width: axisWidth - 4, alignment: .trailing)
                Rectangle().fill(AppTheme.border).frame(height: 0.5)
                }
                .frame(width: width, height: 16)
                .offset(y: CGFloat(hour) * hourHeight)
                .accessibilityHidden(true)
            }
        }
    }

    private func hourLabel(_ hour: Int) -> String {
        let calendar = viewModel.schedule.calendar
        let date = calendar.date(from: DateComponents(year: 2026, month: 1, day: 1, hour: hour))!
        return date.formatted(date: .omitted, time: .shortened)
    }

    private func dayColumn(_ day: Date) -> some View {
        GeometryReader { geometry in
            let placements = viewModel.schedule.placements(for: viewModel.events(for: day), on: day, minimumMinutes: 44 / Double(hourHeight) * 60)
            ZStack(alignment: .topLeading) {
                if placements.isEmpty, days.count == 1 {
                    Text(viewModel.hasLoaded(day) ? "No timed events" : "Loading items…")
                        .font(.subheadline).foregroundStyle(.secondary)
                        .padding(.leading, 16).offset(y: hourHeight * 8 + 18)
                }
                ForEach(placements) { placement in
                    let width = geometry.size.width / CGFloat(placement.columnCount)
                    NavigationLink {
                        EventDetailsView(event: placement.event, apiClient: apiClient)
                    } label: {
                        eventBlock(placement.event)
                            .frame(width: max(1, width - 3), height: max(44, (placement.endMinute - placement.startMinute) / 60 * hourHeight - 2), alignment: .topLeading)
                            .clipped()
                    }
                    .buttonStyle(.plain)
                    .offset(x: CGFloat(placement.column) * width + 1, y: placement.startMinute / 60 * hourHeight + 8)
                    .accessibilityLabel("\(placement.event.title), \(placement.event.startTime.formatted(date: .abbreviated, time: .shortened))")
                    .accessibilityIdentifier("calendar-event-\(placement.event.id)")
                }
            }
        }.frame(height: hourHeight * 24 + 44)
    }

    private func eventBlock(_ event: CalendarEvent) -> some View {
        let color = CalendarPalette.color(event.color)
        return HStack(alignment: .top, spacing: 4) {
            Rectangle().fill(color).frame(width: 3)
            VStack(alignment: .leading, spacing: 2) {
                Text(event.title).font(.caption.weight(.semibold))
                    .lineLimit(days.count > 2 ? nil : 2)
                    .multilineTextAlignment(.leading)
                if days.count < 3 {
                    Text(event.startTime.formatted(date: .omitted, time: .shortened)).font(.caption2)
                    if !event.location.isEmpty { Text(event.location).font(.caption2) }
                }
            }.padding(.vertical, 3)
            Spacer(minLength: 0)
        }
        .foregroundStyle(.primary)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(color.opacity(0.16), in: RoundedRectangle(cornerRadius: 4))
        .contentShape(Rectangle())
    }

    private func currentTime(width: CGFloat) -> some View {
        TimelineView(.periodic(from: .now, by: 60)) { _ in
            if let index = days.firstIndex(where: { viewModel.schedule.calendar.isDate($0, inSameDayAs: viewModel.today) }) {
                let dayWidth = (width - axisWidth) / CGFloat(days.count)
                HStack(spacing: 0) {
                    Circle().fill(.red).frame(width: 6, height: 6)
                    Rectangle().fill(.red).frame(height: 1)
                }
                .frame(width: dayWidth)
                .offset(x: axisWidth + CGFloat(index) * dayWidth,
                        y: viewModel.schedule.minuteOfDay(viewModel.today) / 60 * hourHeight + 5)
                .accessibilityLabel("Current time, \(viewModel.today.formatted(date: .omitted, time: .shortened))")
                .allowsHitTesting(false)
            }
        }
    }
}

private struct AllDayItemsSheet: View {
    let viewModel: CalendarViewModel
    let day: Date
    let apiClient: any APIClientProtocol
    @Environment(\.dismiss) private var dismiss

    private var items: [CalendarDisplayItem] {
        viewModel.items(for: day, untimedOnly: true)
    }

    var body: some View {
        NavigationStack {
            List {
                if items.isEmpty {
                    Text("No all-day items")
                        .foregroundStyle(AppTheme.secondary)
                        .listRowBackground(AppTheme.surface)
                } else {
                    ForEach(items) { item in
                        CalendarItemRow(item: item, apiClient: apiClient)
                            .listRowBackground(AppTheme.surface)
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(AppTheme.background)
            .navigationTitle(day.formatted(.dateTime.weekday(.wide).day().month(.wide)))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                        .accessibilityIdentifier("calendar-all-day-done")
                }
            }
            .tint(AppTheme.accent)
        }
        .accessibilityIdentifier("calendar-all-day-sheet")
    }
}
