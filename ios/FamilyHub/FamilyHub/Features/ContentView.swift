import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol

    var body: some View {
        TabView {
            Tab("Home", systemImage: "house") {
                DashboardView(apiClient: apiClient)
            }
            Tab("Meals", systemImage: "fork.knife") {
                MealsView(apiClient: apiClient)
            }
            Tab("Calendar", systemImage: "calendar") {
                CalendarView(apiClient: apiClient)
            }
        }
    }
}
