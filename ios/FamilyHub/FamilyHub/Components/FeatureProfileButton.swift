import SwiftUI
import FamilyHubKit

struct FeatureProfileButton: View {
    let apiClient: any APIClientProtocol
    @State private var user: User?
    @State private var showsProfile = false

    var body: some View {
        Button { showsProfile = true } label: {
            Image(systemName: "gearshape").font(.title3)
                .frame(width: 44, height: 44)
        }
        .accessibilityLabel("Settings")
                    .accessibilityIdentifier("Open profile")
        .sheet(isPresented: $showsProfile, onDismiss: {
            Task { user = try? await apiClient.fetchMe() }
        }) {
            ProfileView(apiClient: apiClient)
        }
        .task { user = try? await apiClient.fetchMe() }
    }
}
