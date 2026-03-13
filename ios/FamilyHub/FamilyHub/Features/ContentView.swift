// ios/FamilyHub/Features/ContentView.swift
import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol

    var body: some View {
        TabView {
            DashboardView(apiClient: apiClient)
                .tabItem {
                    Label("Dashboard", systemImage: "house")
                }

            ChoresView(apiClient: apiClient)
                .tabItem {
                    Label("Chores", systemImage: "checklist")
                }

            MealsView(apiClient: apiClient)
                .tabItem {
                    Label("Meals", systemImage: "fork.knife")
                }

            RecipesView(apiClient: apiClient)
                .tabItem {
                    Label("Recipes", systemImage: "book")
                }

            CalendarView(apiClient: apiClient)
                .tabItem {
                    Label("Calendar", systemImage: "calendar")
                }
        }
    }
}
