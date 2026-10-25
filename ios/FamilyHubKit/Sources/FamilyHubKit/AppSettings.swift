import Foundation

public struct AppSettings: Codable {
    public let familyName: String
    public let mealWeekStart: MealWeekday

    public init(familyName: String, mealWeekStart: MealWeekday = .friday) {
        self.familyName = familyName
        self.mealWeekStart = mealWeekStart
    }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        familyName = try values.decode(String.self, forKey: .familyName)
        mealWeekStart = try values.decodeIfPresent(MealWeekday.self, forKey: .mealWeekStart) ?? .friday
    }

    enum CodingKeys: String, CodingKey {
        case familyName = "family_name"
        case mealWeekStart = "meal_week_start"
    }
}
