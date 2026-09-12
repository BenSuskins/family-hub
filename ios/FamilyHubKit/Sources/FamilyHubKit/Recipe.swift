import Foundation

public struct IngredientGroup: Codable, Hashable, Sendable {
    public let name: String
    public let items: [String]

    public init(name: String, items: [String]) {
        self.name = name
        self.items = items
    }
}

public struct Recipe: Codable, Identifiable, Hashable, Sendable {
    public let id: String
    public let title: String
    public let steps: [String]?           // Go nil slice marshals as null
    public let ingredients: [IngredientGroup]?  // Go nil slice marshals as null
    public let mealType: String?
    public let servings: Int?
    public let prepTime: String?
    public let cookTime: String?
    public let sourceURL: String?
    public let categoryID: String?
    public let hasImage: Bool
    /// Timer hints in seconds, aligned with `steps`, `nil` where a step has no
    /// recognisable timing. Computed server-side and only sent by the
    /// single-recipe endpoint — the list endpoint omits `steps`, so there is
    /// nothing to align against. Optional so older responses still decode.
    public let stepDurations: [Int?]?

    public init(
        id: String,
        title: String,
        steps: [String]? = nil,
        ingredients: [IngredientGroup]? = nil,
        mealType: String? = nil,
        servings: Int? = nil,
        prepTime: String? = nil,
        cookTime: String? = nil,
        sourceURL: String? = nil,
        categoryID: String? = nil,
        hasImage: Bool = false,
        stepDurations: [Int?]? = nil
    ) {
        self.id = id
        self.title = title
        self.steps = steps
        self.ingredients = ingredients
        self.mealType = mealType
        self.servings = servings
        self.prepTime = prepTime
        self.cookTime = cookTime
        self.sourceURL = sourceURL
        self.categoryID = categoryID
        self.hasImage = hasImage
        self.stepDurations = stepDurations
    }

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case title = "Title"
        case steps = "Steps"
        case ingredients = "Ingredients"
        case mealType = "MealType"
        case servings = "Servings"
        case prepTime = "PrepTime"
        case cookTime = "CookTime"
        case sourceURL = "SourceURL"
        case categoryID = "CategoryID"
        case hasImage = "HasImage"
        case stepDurations = "StepDurations"
    }
}

public struct RecipeRequest: Encodable {
    public var title: String
    public var steps: [String]
    public var ingredients: [IngredientGroup]
    public var mealType: String?
    public var servings: Int?
    public var prepTime: String?
    public var cookTime: String?
    public var sourceURL: String?
    // nil = leave image unchanged (update), "" = clear image, "data:..." = new image
    public var imageData: String?

    public init(
        title: String,
        steps: [String],
        ingredients: [IngredientGroup],
        mealType: String? = nil,
        servings: Int? = nil,
        prepTime: String? = nil,
        cookTime: String? = nil,
        sourceURL: String? = nil,
        imageData: String? = nil
    ) {
        self.title = title
        self.steps = steps
        self.ingredients = ingredients
        self.mealType = mealType
        self.servings = servings
        self.prepTime = prepTime
        self.cookTime = cookTime
        self.sourceURL = sourceURL
        self.imageData = imageData
    }
}
