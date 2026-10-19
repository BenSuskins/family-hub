import AppIntents

nonisolated enum SiriMealType: String, AppEnum {
    case breakfast, lunch, dinner
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Meal" }
    static var caseDisplayRepresentations: [Self: DisplayRepresentation] {
        [.breakfast: "Breakfast", .lunch: "Lunch", .dinner: "Dinner"]
    }
}

nonisolated enum SiriStockOperation: String, AppEnum {
    case setCount, addCount, subtractCount, setPercentage
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Stock change" }
    static var caseDisplayRepresentations: [Self: DisplayRepresentation] {
        [.setCount: "Set count", .addCount: "Add count", .subtractCount: "Subtract count", .setPercentage: "Set percentage"]
    }
}

nonisolated enum SiriTrackingMode: String, AppEnum {
    case count, level
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Stock measurement" }
    static var caseDisplayRepresentations: [Self: DisplayRepresentation] { [.count: "Whole-item count", .level: "Percentage"] }
}

nonisolated enum SiriInventoryRead: String, AppEnum {
    case item, area
    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Inventory query" }
    static var caseDisplayRepresentations: [Self: DisplayRepresentation] { [.item: "One item", .area: "A storage area"] }
}
