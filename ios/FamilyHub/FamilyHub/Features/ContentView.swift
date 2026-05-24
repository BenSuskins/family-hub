import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol
    var isDemoMode: Bool = false

    var body: some View {
        TabView {
            Tab("Home", systemImage: "house") {
                HomeView(apiClient: apiClient)
            }
            Tab("Meals", systemImage: "fork.knife") {
                MealsView(apiClient: apiClient)
            }
            Tab("Recipes", systemImage: "book") {
                RecipesView(apiClient: apiClient)
            }
            Tab("Calendar", systemImage: "calendar") {
                CalendarView(apiClient: apiClient)
            }
        }
        .toolbarBackground(.ultraThinMaterial, for: .tabBar)
        .toolbarBackground(.visible, for: .tabBar)
        .safeAreaInset(edge: .top) {
            if isDemoMode {
                HStack(spacing: 6) {
                    Image(systemName: "theatermasks")
                    Text("Demo Mode — changes are not saved")
                        .font(.caption.weight(.medium))
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
                .background(.orange.opacity(0.15))
                .foregroundStyle(.orange)
            }
        }
    }
}
