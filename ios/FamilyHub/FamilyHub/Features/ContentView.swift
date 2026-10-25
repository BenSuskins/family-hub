import SwiftUI
import FamilyHubKit

enum AppTab: CaseIterable {
    case today, calendar, tasks, food, household
}
enum FoodDestination: Hashable { case mealPlan, recipes }
enum HouseholdDestination: Hashable { case shopping, inventory }

@Observable
final class AppNavigation {
    var selectedTab: AppTab = .today
    var foodPath: [FoodDestination] = []
    var householdPath: [HouseholdDestination] = []
    private(set) var rootRefreshIDs = Dictionary(uniqueKeysWithValues: AppTab.allCases.map { ($0, UUID()) })

    func rootRefreshID(for tab: AppTab) -> UUID { rootRefreshIDs[tab]! }

    func select(_ tab: AppTab) {
        if selectedTab == tab {
            rootRefreshIDs[tab] = UUID()
            switch tab {
            case .food: foodPath = []
            case .household: householdPath = []
            default: break
            }
            return
        }
        selectedTab = tab
    }

    func openMealPlan() {
        foodPath = [.mealPlan]
        selectedTab = .food
    }

    func open(_ url: URL) {
        guard url.scheme == "familyhub" else { return }
        switch url.host {
        case "chores":
            rootRefreshIDs[.tasks] = UUID()
            selectedTab = .tasks
        case "today":
            rootRefreshIDs[.today] = UUID()
            selectedTab = .today
        case "calendar": selectedTab = .calendar
        case "meals": openMealPlan()
        default: break
        }
    }
}
struct ContentView: View {
    let apiClient: any APIClientProtocol
    @State private var shoppingModel: ShoppingViewModel
    @State private var navigation = AppNavigation()
    @State private var notifications = ChoreNotificationManager.shared
    @Environment(\.scenePhase) private var scenePhase

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _shoppingModel = State(wrappedValue: ShoppingViewModel(apiClient: apiClient))
    }

    var body: some View {
        TabView(selection: $navigation.selectedTab) {
            Tab("Today", systemImage: "house", value: AppTab.today) {
                HomeView(apiClient: apiClient, navigation: navigation)
                    .id(navigation.rootRefreshID(for: .today))
            }
            Tab("Calendar", systemImage: "calendar", value: AppTab.calendar) {
                CalendarView(apiClient: apiClient)
                    .id(navigation.rootRefreshID(for: .calendar))
            }
            Tab("Food", systemImage: "fork.knife", value: AppTab.food) {
                NavigationStack(path: $navigation.foodPath) {
                    MealsView(apiClient: apiClient, isFoodHome: true)
                    .navigationDestination(for: FoodDestination.self) { destination in
                        switch destination {
                        case .mealPlan: MealsView(apiClient: apiClient)
                        case .recipes: RecipesView(apiClient: apiClient)
                        }
                    }
                }
                .id(navigation.rootRefreshID(for: .food))
            }
            Tab("Household", systemImage: "cabinet", value: AppTab.household) {
                NavigationStack(path: $navigation.householdPath) {
                    ShoppingListView(apiClient: apiClient, isHouseholdHome: true, model: shoppingModel)
                    .navigationDestination(for: HouseholdDestination.self) { destination in
                        switch destination {
                        case .shopping: ShoppingListView(apiClient: apiClient, model: shoppingModel)
                        case .inventory: InventoryHomeView(apiClient: apiClient)
                        }
                    }
                }
                .id(navigation.rootRefreshID(for: .household))
            }
            Tab("Chores", systemImage: "checklist", value: AppTab.tasks) {
                NavigationStack {
                    ChoresListView(apiClient: apiClient)
                }
                .id(navigation.rootRefreshID(for: .tasks))
            }
        }
        .modifier(AdaptiveTabStyle())
        .modifier(AccessibleMotionModifier())
        .onOpenURL { navigation.open($0) }
        .task {
            if apiClient.serverIdentifier != nil { try? await notifications.configure(apiClient: apiClient) }
            openNotification()
        }
        .onChange(of: notifications.pendingNotificationURL) { _, _ in openNotification() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active && apiClient.serverIdentifier != nil {
                Task { try? await notifications.configure(apiClient: apiClient) }
            }
        }
    }
    private func openNotification() {
        guard let url = notifications.pendingNotificationURL else { return }
        notifications.pendingNotificationURL = nil
        navigation.open(url)
    }
}
