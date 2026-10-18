import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class HouseholdViewModel {
    var weather: HouseholdWeather?
    var departures: HouseholdDepartures?
    var weatherMessage = "Loading weather…"
    private let apiClient: any APIClientProtocol
    private let now: () -> Date
    private var weatherAttempt = Date.distantPast
    private var departureAttempt = Date.distantPast

    init(apiClient: any APIClientProtocol, now: @escaping () -> Date = Date.init) {
        self.apiClient = apiClient
        self.now = now
    }

    func load(force: Bool = false) async {
        async let weatherTask: Void = loadWeather(force: force)
        async let departureTask: Void = loadDepartures(force: force)
        _ = await (weatherTask, departureTask)
    }

    func loadWeather(force: Bool) async {
        guard force || now().timeIntervalSince(weatherAttempt) >= 900 else { return }
        weatherAttempt = now()
        if let weather, let captured = APIDate.parse(weather.fetchedAt), now().timeIntervalSince(captured) >= 900 {
            self.weather = nil
            weatherMessage = "Loading weather…"
        }
        do {
            let loaded = try await apiClient.fetchWeather()
            guard !Task.isCancelled else { return }
            weather = loaded.status == "ok" ? loaded : nil
            if loaded.status == "ok", let captured = APIDate.parse(loaded.fetchedAt) { weatherAttempt = captured }
            weatherMessage = loaded.status == "setup_required" ? "Set the home location in Admin Home Info." : "Unavailable"
        } catch {
            guard !Task.isCancelled else { return }
            weather = nil
            weatherMessage = "Unavailable"
        }
    }

    private func loadDepartures(force: Bool) async {
        guard force || now().timeIntervalSince(departureAttempt) >= 300 else { return }
        departureAttempt = now()
        do {
            let loaded = try await apiClient.fetchDepartures()
            guard !Task.isCancelled else { return }
            departures = loaded.status == "ok" ? loaded : nil
        } catch {
            guard !Task.isCancelled else { return }
            departures = nil
        }
    }
}
