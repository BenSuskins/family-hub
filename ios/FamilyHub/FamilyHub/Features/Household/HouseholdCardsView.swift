import SwiftUI
import FamilyHubKit

struct HouseholdCardsView: View {
    let apiClient: any APIClientProtocol
    @State private var model: HouseholdViewModel
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    @Environment(\.scenePhase) private var scenePhase
    @State private var isVisible = false
    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _model = State(wrappedValue: HouseholdViewModel(apiClient: apiClient))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            NavigationLink {
                WeatherDetailView(apiClient: apiClient)
            } label: {
                VStack(alignment: .leading, spacing: 6) {
                    if dynamicTypeSize.isAccessibilitySize {
                        Text("Weather at home").font(.subheadline.weight(.semibold))
                    } else {
                        Label("Weather at home", systemImage: "cloud.sun").font(.subheadline.weight(.semibold))
                    }
                    if let weather = model.weather {
                        if dynamicTypeSize.isAccessibilitySize {
                            VStack(alignment: .leading, spacing: 6) {
                                Text("\(weather.current.temperature, specifier: "%.0f")°C").font(.title2.bold())
                                Text(weather.current.condition).font(.footnote).foregroundStyle(.secondary)
                            }
                        } else {
                            HStack {
                                Image(systemName: weather.current.symbol).font(.body).accessibilityHidden(true)
                                Text("\(weather.current.temperature, specifier: "%.0f")°C").font(.title2.bold())
                                Text(weather.current.condition).font(.footnote).foregroundStyle(.secondary)
                                Spacer()
                                Image(systemName: "chevron.right").accessibilityHidden(true)
                            }
                        }
                        Text(weather.advice.joined(separator: " ")).font(.footnote)
                    } else { Text(model.weatherMessage).foregroundStyle(.secondary) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 16)
                .padding(.vertical, 12)
                .glassCard(radius: 16)
            }.buttonStyle(.plain)

            TimelineView(.periodic(from: .now, by: 30)) { context in
                if let departure = model.departures?.items.first(where: { $0.countdown(at: context.date) != nil }) {
                    DepartureCard(departure: departure, now: context.date)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.bottom, 16)
        .onAppear { isVisible = true }
        .onDisappear { isVisible = false }
        .task(id: scenePhase == .active && isVisible) { [shouldRefresh = scenePhase == .active && isVisible] in
            guard shouldRefresh else { return }
            await model.load(force: true)
            while !Task.isCancelled {
                do { try await Task.sleep(for: .seconds(30)) } catch { return }
                await model.load()
            }
        }
    }
}

struct DepartureCard: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let departure: Departure
    let now: Date
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if dynamicTypeSize.isAccessibilitySize {
                Text("Next departure · From home").font(.headline)
            } else {
                Label("Next departure · From home", systemImage: "car").font(.headline)
            }
            Text(departure.title).font(.title3.bold())
            Text(departure.destination).foregroundStyle(.secondary)
            if let countdown = departure.countdown(at: now) { Text(countdown).font(.title2.bold()) }
            if let date = APIDate.parse(departure.leaveAt) {
                Text("Leave at \(date.formatted(date: .omitted, time: .shortened)) · \(Int(ceil(Double(departure.drivingSeconds) / 60))) min drive")
                    .font(.subheadline)
            }
            Text("Current driving estimate, including traffic.").font(.caption).foregroundStyle(.secondary)
            Text("Google Maps").font(.system(size: 12)).foregroundStyle(.secondary)
        }.frame(maxWidth: .infinity, alignment: .leading).padding(16).glassCard(radius: 16)
    }
}

struct WeatherDetailView: View {
    let apiClient: any APIClientProtocol
    @State private var model: HouseholdViewModel
    @Environment(\.scenePhase) private var scenePhase
    @State private var isVisible = false
    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _model = State(wrappedValue: HouseholdViewModel(apiClient: apiClient))
    }
    var body: some View {
        List {
            if let weather = model.weather {
                Section("At home") {
                    Label("\(weather.current.temperature, specifier: "%.0f")°C · \(weather.current.condition)", systemImage: weather.current.symbol)
                    Text("High \(weather.high, specifier: "%.0f")°C · Low \(weather.low, specifier: "%.0f")°C")
                    ForEach(weather.advice, id: \.self) { Text($0) }
                }
                Section("Next 24 hours") {
                    ForEach(weather.hours) { hour in
                        VStack(alignment: .leading, spacing: 6) {
                            if let date = APIDate.parse(hour.time) {
                                Text(date.formatted(date: .abbreviated, time: .shortened)).font(.headline)
                            }
                            Label("\(hour.temperature, specifier: "%.0f")°C · \(hour.condition)", systemImage: hour.symbol)
                            Text("Rain \(hour.rainProbability, specifier: "%.0f")% · Wind \(hour.windSpeed, specifier: "%.0f") km/h")
                                .font(.subheadline).foregroundStyle(.secondary)
                        }.padding(.vertical, 4)
                    }
                }
            } else { Text(model.weatherMessage) }
            Section {
                Link("Weather data: Open-Meteo", destination: URL(string: "https://open-meteo.com")!)
                Text("Advice uses the forecast for the next six hours.").font(.footnote).foregroundStyle(.secondary)
            }
        }
        .navigationTitle("Weather")
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
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(event.title).font(.title.bold())
                Text(event.allDay ? "All day" : event.startTime.formatted(date: .abbreviated, time: .shortened))
                if let end = event.endTime, !event.allDay { Text("Ends \(end.formatted(date: .omitted, time: .shortened))") }
                if !event.location.isEmpty { Label(event.location, systemImage: "mappin") }
                if !event.description.isEmpty { Text(event.description) }
                if !event.allDay, !event.location.isEmpty {
                    TimelineView(.periodic(from: .now, by: 30)) { context in
                        if let departure = departures?.items.first(where: { $0.eventID == event.id }), departure.countdown(at: context.date) != nil {
                            DepartureCard(departure: departure, now: context.date)
                        } else { Text(message).foregroundStyle(.secondary) }
                    }
                }
            }.frame(maxWidth: .infinity, alignment: .leading).padding()
        }
        .navigationTitle("Event")
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
