import Foundation

public struct MealPlan: Codable, Identifiable {
    public let date: String     // "YYYY-MM-DD"
    public let mealType: String // "breakfast" | "lunch" | "dinner"
    public let name: String
    public let notes: String
    public let recipeID: String?

    public var id: String { "\(date)-\(mealType)" }

    public init(date: String, mealType: String, name: String, notes: String, recipeID: String?) {
        self.date = date
        self.mealType = mealType
        self.name = name
        self.notes = notes
        self.recipeID = recipeID
    }

    enum CodingKeys: String, CodingKey {
        case date = "Date"
        case mealType = "MealType"
        case name = "Name"
        case notes = "Notes"
        case recipeID = "RecipeID"
    }
}
