import SwiftUI
import FamilyHubKit

struct CalendarView: View {
    @State private var viewModel: CalendarViewModel
    private let apiClient: any APIClientProtocol
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.calendar) private var calendar
    @Environment(\.timeZone) private var timeZone
    @Environment(\.accessibilityVoiceOverEnabled) private var voiceOverEnabled
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: CalendarViewModel(
            apiClient: apiClient,
            now: CalendarDemoData.isEnabled ? { CalendarDemoData.now } : Date.init,
            preferences: CalendarDemoData.isEnabled ? nil : .standard
        ))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                status
                content
            }
            .background(AppTheme.background)
            .toolbarBackground(AppTheme.background, for: .navigationBar)
            .overlay(alignment: .top) {
                if viewModel.isLoading {
                    ProgressView().controlSize(.small).padding(4)
                        .accessibilityLabel("Loading calendar")
                }
            }
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    if viewModel.viewMode != .year {
                        Button { viewModel.goBack() } label: {
                            HStack(spacing: 4) {
                                Image(systemName: "chevron.left")
                                Text(viewModel.viewMode == .month
                                     ? viewModel.selectedDate.formatted(.dateTime.year())
                                     : viewModel.selectedDate.formatted(.dateTime.month(.abbreviated)))
                            }
                        }
                        .accessibilityLabel(viewModel.viewMode == .month ? "Show year" : "Show month")
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    HStack(spacing: 8) {
                        Button("Today") { viewModel.goToToday() }
                            .frame(minHeight: 44)
                            .accessibilityIdentifier("calendar-today")
                        viewMenu
                    }
                }
            }
            .onChange(of: scenePhase) { _, phase in
                if phase == .active { Task { await viewModel.load(forceRefresh: true) } }
            }
            .onChange(of: calendar) { _, _ in updateCalendar() }
            .onChange(of: timeZone) { _, _ in updateCalendar() }
            .task(id: viewModel.requestKey) { await viewModel.load() }
            .onDisappear { viewModel.cancelLoads() }
        }
    }

    private func updateCalendar() {
        var calendar = calendar
        calendar.timeZone = timeZone
        viewModel.updateCalendar(calendar)
    }

    private var title: String {
        switch viewModel.viewMode {
        case .year: "Calendar"
        case .month: "Calendar"
        default: viewModel.selectedDate.formatted(.dateTime.month(.wide).day())
        }
    }

    private var viewMenu: some View {
        Menu {
            ForEach(CalendarViewMode.allCases, id: \.self) { mode in
                Button { viewModel.setView(mode) } label: {
                    Label(mode.rawValue, systemImage: viewModel.viewMode == mode ? "checkmark" : icon(for: mode))
                }.accessibilityIdentifier("calendar-mode-\(mode.rawValue)")
            }
        } label: {
            Image(systemName: "rectangle.topthird.inset.filled")
                .frame(minWidth: 44, minHeight: 44)
        }
        .accessibilityLabel("Calendar view")
        .accessibilityValue(viewModel.viewMode.rawValue)
        .accessibilityIdentifier("calendar-view-menu")
    }

    private func icon(for mode: CalendarViewMode) -> String {
        switch mode {
        case .year, .month: "calendar"
        case .day: "rectangle"
        case .week: "rectangle.split.3x1"
        }
    }

    @ViewBuilder
    private var status: some View {
        if let error = viewModel.loadError {
            HStack {
                Text(error.localizedDescription).font(.caption)
                Spacer()
                Button("Retry") { Task { await viewModel.load(forceRefresh: true) } }
            }
            .padding(12).background(AppTheme.surface)
            .accessibilityIdentifier("calendar-error")
        } else if let metadata = viewModel.currentMonthSnapshotMetadata, !metadata.isFresh {
            HStack {
                Text("Saved calendar · Last updated \(metadata.savedAt.formatted(date: .abbreviated, time: .shortened))")
                    .font(.caption)
                Spacer()
                Button("Retry") { Task { await viewModel.load(forceRefresh: true) } }
            }
            .padding(12).background(AppTheme.surface)
        }
    }

    @ViewBuilder
    private var content: some View {
        switch viewModel.viewMode {
        case .year:
            CalendarYearView(viewModel: viewModel)
        case .month:
            if dynamicTypeSize.isAccessibilitySize || voiceOverEnabled {
                accessibleMonth
            } else {
                CalendarMonthsView(viewModel: viewModel)
            }
        case .day, .week:
            if dynamicTypeSize.isAccessibilitySize || voiceOverEnabled {
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        datePicker
                        ForEach(viewModel.timelineDays, id: \.self) { day in
                            CalendarAgendaView(viewModel: viewModel, day: day, apiClient: apiClient)
                        }
                    }.padding()
                }.refreshable { await viewModel.load(forceRefresh: true) }
            } else {
                CalendarTimelineView(viewModel: viewModel, apiClient: apiClient)
            }
        }
    }

    private var dateSelection: Binding<Date> {
        Binding(get: { viewModel.selectedDate }, set: { viewModel.selectDate($0) })
    }

    private var datePicker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Date").font(.headline).accessibilityHidden(true)
            DatePicker("Date", selection: dateSelection, displayedComponents: .date)
                .labelsHidden()
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityIdentifier("calendar-date-picker")
        }
    }

    private var accessibleMonth: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                datePicker
                CalendarAgendaView(viewModel: viewModel, day: viewModel.selectedDate, apiClient: apiClient)
                Button("Open day") { viewModel.openDay(viewModel.selectedDate) }
            }.padding()
        }.refreshable { await viewModel.load(forceRefresh: true) }
    }
}
