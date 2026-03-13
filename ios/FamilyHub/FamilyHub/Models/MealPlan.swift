import Foundation

struct MealPlan: Codable, Identifiable {
    let date: String     // "YYYY-MM-DD"
    let mealType: String // "breakfast" | "lunch" | "dinner"
    let name: String
    let notes: String
    let recipeID: String?

    var id: String { "\(date)-\(mealType)" }

    enum CodingKeys: String, CodingKey {
        case date = "Date"
        case mealType = "MealType"
        case name = "Name"
        case notes = "Notes"
        case recipeID = "RecipeID"
    }
}
