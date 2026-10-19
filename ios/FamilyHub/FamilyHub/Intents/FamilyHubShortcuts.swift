import AppIntents

struct FamilyHubShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(intent: ReadMealIntent(), phrases: ["What's for \(\.$meal) in \(.applicationName)", "Read a meal in \(.applicationName)"], shortTitle: "What's for Dinner?", systemImageName: "fork.knife")
        AppShortcut(intent: PlanMealIntent(), phrases: ["Set \(\.$meal) in \(.applicationName)", "Plan a meal in \(.applicationName)"], shortTitle: "Plan a Meal", systemImageName: "calendar.badge.plus")
        AppShortcut(intent: ReadMyChoresIntent(), phrases: ["What are my chores in \(.applicationName)", "Read my chores in \(.applicationName)"], shortTitle: "My Chores", systemImageName: "checklist")
        AppShortcut(intent: CompleteNamedChoreIntent(), phrases: ["Complete a chore in \(.applicationName)"], shortTitle: "Complete One Chore", systemImageName: "checkmark.circle")
        AppShortcut(intent: CompleteMyChoresIntent(), phrases: ["Complete my chores in \(.applicationName)"], shortTitle: "Complete My Chores", systemImageName: "checkmark.circle.fill")
        AppShortcut(intent: AddShoppingIntent(), phrases: ["Add to my shopping list in \(.applicationName)"], shortTitle: "Add Shopping", systemImageName: "cart.badge.plus")
        AppShortcut(intent: ReadInventoryIntent(), phrases: ["Read inventory in \(.applicationName)", "What's in \(\.$area) in \(.applicationName)"], shortTitle: "Read Inventory", systemImageName: "shippingbox")
        AppShortcut(intent: ReadLowStockIntent(), phrases: ["What's low in stock in \(.applicationName)", "Read low stock in \(.applicationName)"], shortTitle: "Low Stock", systemImageName: "exclamationmark.circle")
        AppShortcut(intent: ChangeStockIntent(), phrases: ["Update stock in \(.applicationName)"], shortTitle: "Update Stock", systemImageName: "plusminus")
        AppShortcut(intent: CreateInventoryItemIntent(), phrases: ["Add an inventory item in \(.applicationName)"], shortTitle: "Add Inventory Item", systemImageName: "plus.circle")
    }
}
