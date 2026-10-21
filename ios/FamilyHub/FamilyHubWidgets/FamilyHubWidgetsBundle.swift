import SwiftUI
import WidgetKit

@main
struct FamilyHubWidgetsBundle: WidgetBundle {
    var body: some Widget {
        TodayWidget()
        TonightWidget()
        MealPlanWidget()
        UpNextWidget()
        ChoresWidget()
    }
}
