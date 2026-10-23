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
                    VStack(alignment: .leading, spacing: 14) {
                        HStack(spacing: 12) {
                            WarmIcon(symbol: "house")
                            VStack(alignment: .leading, spacing: 4) {
                                Text("Saved location").font(.headline)
                                Text("\(latitude, specifier: "%.5f"), \(longitude, specifier: "%.5f")")
                                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                            }
                        }
                        Divider()
                        LabeledContent("Time zone", value: home.timezone)
                    }
                } else {
                    Label("No home location set", systemImage: "mappin.and.ellipse")
                        .foregroundStyle(AppTheme.secondary)
                }
            }.listRowBackground(AppTheme.surface)
            Section {
                Button(locationReader.isLocating ? "Locating…" : "Use this iPhone’s location") { locationReader.capture() }
                    .buttonStyle(AppSecondaryButtonStyle())
                    .accessibilityIdentifier("Use device location")
                    .disabled(locationReader.isLocating || !didLoad)
                if let coordinate = locationReader.coordinate {
                    VStack(alignment: .leading, spacing: 12) {
                        Label("New location", systemImage: "location.fill")
                            .font(.headline)
                            .foregroundStyle(AppTheme.accent)
                        Text("\(coordinate.latitude, specifier: "%.5f"), \(coordinate.longitude, specifier: "%.5f")")
                            .font(.subheadline.monospacedDigit())
                        Text("Confirm this location while you are at home.")
                            .font(.subheadline).foregroundStyle(AppTheme.secondary)
                        Button("Set as home location") {
                            home.latitude = coordinate.latitude
                            home.longitude = coordinate.longitude
                            home.timezone = TimeZone.current.identifier
                            locationReader.coordinate = nil
                        }
                        .buttonStyle(AppSecondaryButtonStyle())
                    }
                    .padding(14)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(AppTheme.sage.opacity(0.35), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
                if let message = locationReader.message { Text(message).foregroundStyle(AppTheme.secondary) }
            } footer: { Text("Capture the location while you are at home. Review it before you save.") }
            .listRowBackground(Color.clear)
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
