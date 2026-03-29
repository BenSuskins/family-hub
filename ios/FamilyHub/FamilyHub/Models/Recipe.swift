import Foundation

struct IngredientGroup: Codable {
    let name: String
    let items: [String]
}

struct Recipe: Codable, Identifiable {
    let id: String
    let title: String
    let steps: [String]?           // Go nil slice marshals as null
    let ingredients: [IngredientGroup]?  // Go nil slice marshals as null
    let mealType: String?
    let servings: Int?
    let prepTime: String?
    let cookTime: String?
    let hasImage: Bool

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case title = "Title"
        case steps = "Steps"
        case ingredients = "Ingredients"
        case mealType = "MealType"
        case servings = "Servings"
        case prepTime = "PrepTime"
        case cookTime = "CookTime"
        case hasImage = "HasImage"
    }
}
