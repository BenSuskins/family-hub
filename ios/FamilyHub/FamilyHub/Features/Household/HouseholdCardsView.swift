import SwiftUI
import FamilyHubKit

struct HouseholdCardsView: View {
    let model: HouseholdViewModel
    let apiClient: any APIClientProtocol
    let events: [CalendarEvent]

    private func modelEvent(_ departure: Departure) -> CalendarEvent? {
        events.first { $0.id == departure.eventID }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            TimelineView(.periodic(from: .now, by: 30)) { context in
                if let departure = model.departures?.items.first(where: { $0.countdown(at: context.date) != nil }) {
                    VStack(alignment: .leading, spacing: 0) {
                        DepartureCard(departure: departure, now: context.date, compact: true)
                        Divider().padding(.horizontal, 20)
                        NavigationLink {
                            DepartureEventView(departure: departure, initialEvent: modelEvent(departure), apiClient: apiClient)
                        } label: {
                            HStack { Text("View event"); Spacer(); Image(systemName: "chevron.right") }
                                .padding(20).frame(minHeight: 44)
                        }.buttonStyle(.plain).foregroundStyle(AppTheme.accent)
                    }
                    .background(AppTheme.departure, in: RoundedRectangle(cornerRadius: 28))
                }
            }
        }
        .padding(.horizontal, 20)
        .padding(.bottom, 16)
    }
}

private struct DepartureEventView: View {
    let departure: Departure
    let initialEvent: CalendarEvent?
    let apiClient: any APIClientProtocol
    @State private var event: CalendarEvent?
    @State private var error: APIError?
    @State private var isLoading = true

    var body: some View {
        Group {
            if let event = event ?? initialEvent {
                EventDetailsView(event: event, apiClient: apiClient)
            } else if let error {
                ErrorStateView(error: error) { await load() }
            } else if isLoading {
                ProgressView("Loading event")
            } else {
                ContentUnavailableView("Event unavailable", systemImage: "calendar", description: Text("The event may have changed. Refresh Calendar to check it."))
            }
        }
        .task { if initialEvent == nil { await load() } }
    }

    private func load() async {
        isLoading = true
        error = nil
        defer { isLoading = false }
        do {
            let day = APIDate.parse(departure.startTime) ?? Date()
            let response = try await apiClient.fetchCalendar(view: "day", date: day)
            event = response.events.first { $0.id == departure.eventID }
        } catch { self.error = .from(error) }
    }
}

struct DepartureCard: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let departure: Departure
    let now: Date
    var compact = false
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if dynamicTypeSize.isAccessibilitySize {
                Text("Next departure").font(.subheadline.weight(.semibold)).foregroundStyle(AppTheme.clay)
            } else {
                Label("NEXT DEPARTURE", systemImage: "clock").font(.subheadline.weight(.semibold)).foregroundStyle(AppTheme.clay)
            }
            if let countdown = departure.countdown(at: now) {
                Text(countdown).font(.system(.title, design: .rounded, weight: .bold))
            }
            if compact {
                let start = APIDate.parse(departure.startTime)?.formatted(date: .omitted, time: .shortened) ?? ""
                Text(departure.title + (start.isEmpty ? "" : " · " + start)).foregroundStyle(AppTheme.secondary)
                Text("\(Int(ceil(Double(departure.drivingSeconds) / 60))) min drive").foregroundStyle(AppTheme.secondary)
            } else {
            Text(departure.title).font(.headline)
            Text(departure.destination).foregroundStyle(AppTheme.secondary)
            if let date = APIDate.parse(departure.leaveAt) {
                Text("Leave at \(date.formatted(date: .omitted, time: .shortened)) · \(Int(ceil(Double(departure.drivingSeconds) / 60))) min drive")
                    .font(.subheadline)
            }
            Text("Current driving estimate, including traffic.").font(.caption).foregroundStyle(AppTheme.secondary)
            Text("Google Maps").font(.caption).foregroundStyle(AppTheme.secondary)
            }
        }.frame(maxWidth: .infinity, alignment: .leading).padding(20)
            .background(AppTheme.departure, in: RoundedRectangle(cornerRadius: 28))
    }
}

struct WeatherDetailView: View {
    let apiClient: any APIClientProtocol
    @State private var model: HouseholdViewModel
    @Environment(\.scenePhase) private var scenePhase
    @State private var isVisible = false
    @ScaledMetric(relativeTo: .largeTitle) private var temperatureSize = 64.0
    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _model = State(wrappedValue: HouseholdViewModel(apiClient: apiClient))
    }
    var body: some View {
        List {
            if let weather = model.weather {
                Section {
                    VStack(spacing: 12) {
                        Image(systemName: weather.current.symbol).font(.largeTitle).accessibilityHidden(true)
                        Text("\(weather.current.temperature, specifier: "%.0f")°")
                            .font(.system(size: temperatureSize, weight: .regular, design: .rounded))
                            .accessibilityLabel("\(weather.current.temperature, specifier: "%.0f") degrees Celsius")
                        Text(weather.current.condition).font(.headline)
                        Text("High \(weather.high, specifier: "%.0f")° · Low \(weather.low, specifier: "%.0f")°")
                            .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        ForEach(weather.advice, id: \.self) { Text($0).font(.subheadline) }
                    }.frame(maxWidth: .infinity).padding(.vertical, 20)
                }.listRowBackground(AppTheme.surface)
                Section("Next 24 hours") {
                    ForEach(weather.hours) { hour in
                        HStack(alignment: .top, spacing: 12) {
                            VStack(alignment: .leading, spacing: 4) {
                                if let date = APIDate.parse(hour.time) {
                                    Text(date.formatted(date: .omitted, time: .shortened)).font(.body.weight(.medium))
                                }
                                Text("\(hour.condition) · Rain \(hour.rainProbability, specifier: "%.0f")% · Wind \(hour.windSpeed, specifier: "%.0f") km/h")
                                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                            }
                            Spacer()
                            Text("\(hour.temperature, specifier: "%.0f")°").font(.headline)
                        }.padding(.vertical, 4)
                    }
                }.listRowBackground(AppTheme.surface)
                if let fetched = APIDate.parse(weather.fetchedAt) {
                    Section { Text("Updated " + fetched.formatted(date: .omitted, time: .shortened)).font(.footnote).foregroundStyle(AppTheme.secondary) }
                        .listRowBackground(Color.clear)
                }
            } else { Text(model.weatherMessage) }
            Section {
                Link("Weather data: Open-Meteo", destination: URL(string: "https://open-meteo.com")!)
                Text("Advice uses the forecast for the next six hours.").font(.footnote).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle("Home weather")
        .refreshable { await model.loadWeather(force: true) }
        .onAppear { isVisible = true }
        .onDisappear { isVisible = false }
        .task(id: scenePhase == .active && isVisible) { [shouldRefresh = scenePhase == .active && isVisible] in
            guard shouldRefresh else { return }
            await model.loadWeather(force: true)
            while !Task.isCancelled {
                do { try await Task.sleep(for: .seconds(900)) } catch { return }
                await model.loadWeather(force: false)
            }
        }
    }
}

struct EventDetailsView: View {
    let event: CalendarEvent
    let apiClient: any APIClientProtocol
    @State private var departures: HouseholdDepartures?
    @State private var message = "Loading departure…"
    @Environment(\.scenePhase) private var scenePhase
    @State private var isVisible = false
    private var eventDate: String {
        let format = Date.FormatStyle.dateTime.weekday(.wide).day().month(.wide)
        let isCurrentYear = Calendar.current.isDate(event.startTime, equalTo: .now, toGranularity: .year)
        return event.startTime.formatted(isCurrentYear ? format : format.year())
    }

    private var eventTime: String {
        if event.allDay { return "All day" }
        let start = event.startTime.formatted(date: .omitted, time: .shortened)
        guard let end = event.endTime else { return start }
        let endText = end.formatted(date: Calendar.current.isDate(event.startTime, inSameDayAs: end) ? .omitted : .abbreviated, time: .shortened)
        return start + "–" + endText
    }

    private var locationURL: URL? {
        var components = URLComponents(string: "https://maps.apple.com/")
        components?.queryItems = [URLQueryItem(name: "q", value: event.location)]
        return components?.url
    }

    private var locationRow: some View {
        HStack(spacing: 12) {
            WarmIcon(symbol: "house")
            VStack(alignment: .leading, spacing: 4) {
                Text(event.location).foregroundStyle(AppTheme.ink)
                Text("View location").font(.subheadline).foregroundStyle(AppTheme.secondary)
            }
            Spacer()
            Image(systemName: "chevron.right").foregroundStyle(AppTheme.secondary)
        }.padding(16).frame(minHeight: 44)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Family calendar").font(.subheadline.weight(.semibold)).foregroundStyle(AppTheme.secondary)
                Text(event.title).font(.system(.largeTitle, design: .rounded, weight: .bold))
                VStack(alignment: .leading, spacing: 0) {
                    HStack(alignment: .top, spacing: 12) {
                        WarmIcon(symbol: "calendar")
                        VStack(alignment: .leading, spacing: 4) {
                            Text(eventDate)
                            Text(eventTime).font(.subheadline).foregroundStyle(AppTheme.secondary)
                        }
                    }.padding(16)
                    if !event.location.isEmpty {
                        Divider().padding(.leading, 16)
                        if let locationURL {
                            Link(destination: locationURL) { locationRow }
                                .buttonStyle(.plain)
                        } else { locationRow }
                    }
                }.groupedCard()
                if !event.allDay, !event.location.isEmpty {
                    TimelineView(.periodic(from: .now, by: 30)) { context in
                        if let departure = departures?.items.first(where: { $0.eventID == event.id }), departure.countdown(at: context.date) != nil {
                            VStack(alignment: .leading, spacing: 8) {
                                HStack(alignment: .top, spacing: 12) {
                                    WarmIcon(symbol: "clock")
                                    VStack(alignment: .leading, spacing: 4) {
                                        if let leave = APIDate.parse(departure.leaveAt) {
                                            Text("Leave at " + leave.formatted(date: .omitted, time: .shortened)).font(.headline)
                                        }
                                        Text("\(Int(ceil(Double(departure.drivingSeconds) / 60))) min drive")
                                            .font(.subheadline).foregroundStyle(AppTheme.secondary)
                                    }
                                }
                                Text("Current driving estimate, including traffic. · Google Maps")
                                    .font(.caption).foregroundStyle(AppTheme.secondary)
                            }.frame(maxWidth: .infinity, alignment: .leading).padding(16).groupedCard()
                        } else { Text(message).font(.subheadline).foregroundStyle(AppTheme.secondary) }
                    }
                }
                if !event.description.isEmpty {
                    Text("Details").font(.subheadline.weight(.semibold)).foregroundStyle(AppTheme.secondary).padding(.top, 8)
                    Text(event.description).frame(maxWidth: .infinity, alignment: .leading).padding(16).groupedCard()
                }
            }.frame(maxWidth: .infinity, alignment: .leading).padding(20)
        }
        .groupedBackground()
        .navigationTitle("Event")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear { isVisible = true }
        .onDisappear { isVisible = false }
        .task(id: scenePhase == .active && isVisible) { [shouldRefresh = scenePhase == .active && isVisible] in
            guard shouldRefresh else { return }
            while !Task.isCancelled {
                do {
                    departures = try await apiClient.fetchDepartures()
                    message = "No departure estimate for this event."
                } catch {
                    departures = nil
                    message = "Departure unavailable."
                }
                do { try await Task.sleep(for: .seconds(300)) } catch { return }
            }
        }
    }
}
