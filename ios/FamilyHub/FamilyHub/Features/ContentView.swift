import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol

    var body: some View {
        TabView {
            DashboardView(apiClient: apiClient)
                .tabItem { Label("Home", systemImage: "house.fill") }

            ChoresView(apiClient: apiClient)
                .tabItem { Label("Chores", systemImage: "checkmark.circle") }

            MealsView(apiClient: apiClient)
                .tabItem { Label("Meals", systemImage: "fork.knife") }

            RecipesView(apiClient: apiClient)
                .tabItem { Label("Recipes", systemImage: "book.closed") }

            CalendarView(apiClient: apiClient)
                .tabItem { Label("Calendar", systemImage: "calendar") }
        }
        .tint(Theme.accent)
    }
}
