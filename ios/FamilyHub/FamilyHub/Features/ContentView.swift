import SwiftUI
import FamilyHubKit

enum AppTab { case home, meals, inventory, recipes, calendar }

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
            Tab("Inventory", systemImage: "shippingbox", value: AppTab.inventory) {
                InventoryHomeView(apiClient: apiClient)
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
        .onOpenURL { url in
            // Where the Today widget links to: the widget as a whole, and its
            // chore rows, land on Home — there is no Chores tab — while event
            // and meal rows open the tab they came from.
            //
            // The OIDC callback shares this scheme but never arrives here:
            // ASWebAuthenticationSession consumes it, and this view only
            // exists once the user is signed in.
            guard url.scheme == "familyhub" else { return }
            switch url.host {
            case "today":    selectedTab = .home
            case "calendar": selectedTab = .calendar
            case "meals":    selectedTab = .meals
            default:         break
            }
        }
    }
}
