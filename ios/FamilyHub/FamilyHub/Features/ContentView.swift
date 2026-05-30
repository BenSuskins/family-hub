import SwiftUI

enum AppTab { case home, meals, recipes, calendar }

struct ContentView: View {
    let apiClient: any APIClientProtocol
    @State private var selectedTab: AppTab = .home

    var body: some View {
        TabView(selection: $selectedTab) {
            Tab("Home", systemImage: "house", value: AppTab.home) {
                HomeView(apiClient: apiClient, selectedTab: $selectedTab)
            }
            Tab("Meals", systemImage: "fork.knife", value: AppTab.meals) {
                MealsView(apiClient: apiClient)
            }
            Tab("Recipes", systemImage: "book", value: AppTab.recipes) {
                RecipesView(apiClient: apiClient)
            }
            Tab("Calendar", systemImage: "calendar", value: AppTab.calendar) {
                CalendarView(apiClient: apiClient)
            }
        }
        .toolbarBackground(.ultraThinMaterial, for: .tabBar)
        .toolbarBackground(.visible, for: .tabBar)
    }
}
