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
            Section("Home location") {
                Text("Use your home location for weather and departure estimates.").font(.footnote).foregroundStyle(AppTheme.secondary)
                if let latitude = home.latitude, let longitude = home.longitude {
                    Text("\(latitude, specifier: "%.5f"), \(longitude, specifier: "%.5f")")
                    Text(home.timezone).foregroundStyle(AppTheme.secondary)
                } else { Text("No home location set.") }
                Button(locationReader.isLocating ? "Locating…" : "Use device location") { locationReader.capture() }
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
            }.listRowBackground(AppTheme.surface)
            Section("Departure arrival margin") {
                Stepper("\(home.arrivalMarginMinutes) minutes", value: $home.arrivalMarginMinutes, in: 0...120)
                    .disabled(!didLoad)
                Text("Allow this much time before each event starts.").font(.footnote)
            }.listRowBackground(AppTheme.surface)
            if isLoading { ProgressView() }
            if !didLoad && !isLoading { Button("Retry") { Task { await load() } } }
        }
        .appListStyle()
        .navigationTitle("Home location")
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
