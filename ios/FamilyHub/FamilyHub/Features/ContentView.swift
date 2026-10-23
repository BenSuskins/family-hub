import SwiftUI
import FamilyHubKit

enum AppTab: CaseIterable {
    case today, calendar, tasks, food, household
    var title: String {
        switch self {
        case .today: "Today"
        case .calendar: "Calendar"
        case .tasks: "Chores"
        case .food: "Food"
        case .household: "Household"
        }
    }
    var symbol: String {
        switch self {
        case .today: "house"
        case .calendar: "calendar"
        case .tasks: "checklist"
        case .food: "fork.knife"
        case .household: "cabinet"
        }
    }
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
            selectedTab = .tasks
        case "today":
            selectedTab = .today
        case "calendar": selectedTab = .calendar
        case "meals": openMealPlan()
        default: break
        }
    }
}

struct ContentView: View {
    let apiClient: any APIClientProtocol
    @State private var keyboardVisible = false
    @State private var shoppingModel: ShoppingViewModel
    @State private var navigation = AppNavigation()
    @State private var notifications = ChoreNotificationManager.shared
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _shoppingModel = State(wrappedValue: ShoppingViewModel(apiClient: apiClient))
    }

    private var showsFloatingTabBar: Bool {
        UIDevice.current.userInterfaceIdiom == .phone && !dynamicTypeSize.isAccessibilitySize && !keyboardVisible
    }

    var body: some View {
        TabView(selection: $navigation.selectedTab) {
            Tab("Today", systemImage: "house", value: AppTab.today) {
                HomeView(apiClient: apiClient, navigation: navigation)
                    .id(navigation.rootRefreshID(for: .today))
                    .modifier(PhoneTabVisibility())
            }
            Tab("Calendar", systemImage: "calendar", value: AppTab.calendar) {
                CalendarView(apiClient: apiClient)
                    .id(navigation.rootRefreshID(for: .calendar))
                    .modifier(PhoneTabVisibility())
            }
            Tab("Chores", systemImage: "checklist", value: AppTab.tasks) {
                NavigationStack {
                    ChoresListView(apiClient: apiClient)
                }
                .id(navigation.rootRefreshID(for: .tasks))
                .modifier(PhoneTabVisibility())
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
                .modifier(PhoneTabVisibility())
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
                .modifier(PhoneTabVisibility())
            }
        }
        .modifier(AdaptiveTabStyle())
        .contentMargins(.bottom, showsFloatingTabBar ? 90 : 0, for: .scrollContent)
        .overlay(alignment: .bottom) {
            if showsFloatingTabBar {
                FamilyTabBar(selection: $navigation.selectedTab, onSelect: navigation.select)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: UIResponder.keyboardWillShowNotification)) { _ in keyboardVisible = true }
        .onReceive(NotificationCenter.default.publisher(for: UIResponder.keyboardWillHideNotification)) { _ in keyboardVisible = false }
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

private struct FamilyTabBar: View {
    @Binding var selection: AppTab
    let onSelect: (AppTab) -> Void
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.colorSchemeContrast) private var contrast
    var body: some View {
        HStack(spacing: 0) {
            ForEach(AppTab.allCases, id: \.self) { tab in
                Button {
                    if reduceMotion {
                        onSelect(tab)
                    } else {
                        withAnimation(.snappy(duration: 0.28)) { onSelect(tab) }
                    }
                } label: {
                    VStack(spacing: 5) {
                        Image(systemName: tab.symbol).font(.system(size: 21, weight: .medium))
                            .symbolEffect(.bounce, value: selection == tab)
                        Text(tab.title).font(.caption.weight(.semibold))
                            .lineLimit(1).minimumScaleFactor(0.8)
                    }
                    .foregroundStyle(selection == tab ? AppTheme.accent : AppTheme.secondary)
                    .frame(maxWidth: .infinity, minHeight: 58)
                    .background {
                        if selection == tab {
                            Capsule().fill(AppTheme.sage).matchedGeometryEffect(id: "selected-tab", in: tabSelectionAnimation)
                        }
                    }
                    .contentShape(Capsule())
                }
                .buttonStyle(.plain)
                .accessibilityLabel(tab.title)
                .accessibilityIdentifier(tab.title)
                .accessibilityAddTraits(selection == tab ? [.isSelected] : [])
            }
        }
        .padding(6)
        .background {
            if reduceTransparency { Capsule().fill(AppTheme.surface) }
            else { Capsule().fill(.clear).glassEffect(.regular, in: Capsule()) }
        }
        .overlay { Capsule().stroke(contrast == .increased ? AppTheme.ink : AppTheme.border, lineWidth: 1) }
        .shadow(color: .black.opacity(0.05), radius: 12, y: 3)
        .padding(.horizontal, 14)
        .padding(.top, 8)
        .padding(.bottom, 6)
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Tab bar")
    }

    @Namespace private var tabSelectionAnimation
}

private struct PhoneTabVisibility: ViewModifier {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    func body(content: Content) -> some View {
        content.toolbar(UIDevice.current.userInterfaceIdiom == .phone && !dynamicTypeSize.isAccessibilitySize ? .hidden : .automatic, for: .tabBar)
    }
}
