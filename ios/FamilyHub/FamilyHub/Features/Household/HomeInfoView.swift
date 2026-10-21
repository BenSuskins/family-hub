import SwiftUI
import CoreLocation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class HomeLocationReader: NSObject, CLLocationManagerDelegate {
    var coordinate: CLLocationCoordinate2D?
    var message: String?
    var isLocating = false
    private let manager = CLLocationManager()

    override init() {
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyHundredMeters
    }

    func capture() {
        coordinate = nil
        message = nil
        isLocating = true
        switch manager.authorizationStatus {
        case .notDetermined: manager.requestWhenInUseAuthorization()
        case .authorizedAlways, .authorizedWhenInUse: manager.requestLocation()
        default: deny()
        }
    }
    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        guard isLocating else { return }
        switch manager.authorizationStatus {
        case .authorizedAlways, .authorizedWhenInUse: manager.requestLocation()
        case .denied, .restricted: deny()
        default: break
        }
    }
    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        isLocating = false
        guard let location = locations.last, location.horizontalAccuracy >= 0, location.horizontalAccuracy <= 1000, abs(location.timestamp.timeIntervalSinceNow) < 60 else {
            message = "No recent location is available. Try again."
            return
        }
        coordinate = location.coordinate
    }
    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        isLocating = false
        message = "The device location is unavailable. Try again."
    }
    private func deny() {
        isLocating = false
        message = "Allow location access in iOS Settings → Family Hub → Location, then try again."
    }
}

struct HomeInfoView: View {
    let apiClient: any APIClientProtocol
    @State private var home = HomeInfo()
    @State private var locationReader = HomeLocationReader()
    @State private var error: APIError?
    @State private var isLoading = true
    @State private var isSaving = false
    @State private var didLoad = false
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        Form {
            Section {
                Text("Used for local weather and departure times.").font(.subheadline).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)
            Section {
                if let latitude = home.latitude, let longitude = home.longitude {
                    HStack(spacing: 12) {
                        WarmIcon(symbol: "house")
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Saved location")
                            Text("\(latitude, specifier: "%.5f"), \(longitude, specifier: "%.5f")")
                                .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        }
                    }
                    LabeledContent("Time zone", value: home.timezone)
                } else { Text("No home location set.") }
            }.listRowBackground(AppTheme.surface)
            Section {
                Button(locationReader.isLocating ? "Locating…" : "Use this iPhone’s location") { locationReader.capture() }
                    .buttonStyle(AppSecondaryButtonStyle())
                    .accessibilityIdentifier("Use device location")
                    .disabled(locationReader.isLocating || !didLoad)
                if let coordinate = locationReader.coordinate {
                    Text("New location: \(coordinate.latitude, specifier: "%.5f"), \(coordinate.longitude, specifier: "%.5f")")
                    Text("Confirm this location as home. Capture it while you are at home.")
                    Button("Confirm home location") {
                        home.latitude = coordinate.latitude
                        home.longitude = coordinate.longitude
                        home.timezone = TimeZone.current.identifier
                        locationReader.coordinate = nil
                    }
                }
                if let message = locationReader.message { Text(message).foregroundStyle(AppTheme.secondary) }
            } footer: { Text("Capture the location while you are at home. Review it before you save.") }
            .listRowBackground(Color.clear).listRowInsets(EdgeInsets())
            Section("Arrival margin") {
                Stepper(value: $home.arrivalMarginMinutes, in: 0...120) {
                    LabeledContent("Arrive before the event", value: "\(home.arrivalMarginMinutes) min")
                }
                    .disabled(!didLoad)
                Text("Departure times include travel time plus this margin.").font(.footnote)
            }.listRowBackground(AppTheme.surface)
            if isLoading { ProgressView() }
            if !didLoad && !isLoading { Button("Retry") { Task { await load() } } }
        }
        .appListStyle()
        .navigationTitle("Home location")
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button(isSaving ? "Saving…" : "Save") {
                    Task {
                        isSaving = true
                        defer { isSaving = false }
                        do { try await apiClient.updateHomeInfo(home); dismiss() }
                        catch { self.error = .from(error) }
                    }
                }.buttonStyle(AppCommitButtonStyle()).disabled(!didLoad || home.latitude == nil || isSaving || locationReader.coordinate != nil)
            }
        }
        .errorAlert($error)
        .task { await load() }
    }
    private func load() async {
        isLoading = true
        defer { isLoading = false }
        do { home = try await apiClient.fetchHomeInfo(); didLoad = true }
        catch { self.error = .from(error) }
    }
}
