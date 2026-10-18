import SwiftUI
import FamilyHubKit

struct FeatureProfileButton: View {
    let apiClient: any APIClientProtocol
    @State private var user: User?
    @State private var showsProfile = false

    var body: some View {
        Button { showsProfile = true } label: {
            UserAvatar(user: user, size: 32, apiClient: apiClient)
                .frame(width: 44, height: 44)
        }
        .accessibilityLabel("Open profile")
        .sheet(isPresented: $showsProfile, onDismiss: {
            Task { user = try? await apiClient.fetchMe() }
        }) {
            ProfileView(apiClient: apiClient)
        }
        .task { user = try? await apiClient.fetchMe() }
    }
}
