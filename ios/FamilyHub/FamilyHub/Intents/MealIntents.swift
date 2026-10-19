import AppIntents
import Foundation
import FamilyHubKit

struct ReadMealIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "What's for a Meal?" }
    nonisolated static var description: IntentDescription { "Read breakfast, lunch, or dinner for a date. The date defaults to today." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }

    @Parameter(title: "Meal", requestValueDialog: "Which meal: breakfast, lunch, or dinner?")
    var meal: SiriMealType
    @Parameter(title: "Date", kind: .date)
    var date: DateComponents?

    static var parameterSummary: some ParameterSummary { Summary("Read \(\.$meal) for \(\.$date)") }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog & ReturnsValue<String> {
        let service = try SiriEnvironment.service()
        let day = try await service.day(date)
        let plan = try await service.meal(type: meal.rawValue, day: day)
        let answer = plan.map { "\(meal.rawValue.capitalized) on \(day.spoken) is \($0.name)." }
            ?? "No \(meal.rawValue) is planned for \(day.spoken)."
        return .result(value: answer, dialog: "\(answer)")
    }
}

struct PlanMealIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Plan a Meal" }
    nonisolated static var description: IntentDescription { "Set breakfast, lunch, or dinner for a date, replacing the existing meal. The date defaults to today." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }

    @Parameter(title: "Meal", requestValueDialog: "Which meal: breakfast, lunch, or dinner?")
    var meal: SiriMealType
    @Parameter(title: "Meal name", requestValueDialog: "What would you like to eat?")
    var name: String
    @Parameter(title: "Date", kind: .date)
    var date: DateComponents?
    @Parameter(title: "Recipe")
    var recipe: RecipeEntity?

    static var parameterSummary: some ParameterSummary { Summary("Plan \(\.$name) for \(\.$meal) on \(\.$date)") }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog {
        let service = try SiriEnvironment.service()
        let day = try await service.day(date)
        let matches = try await service.recipes(named: name)
        var recipeID: String?
        if let recipe {
            try await SiriEnvironment.validate(scope: recipe.scope, service: service)
            guard matches.contains(where: { $0.id == recipe.rawID }) else { throw SiriActionError.invalid("Select a recipe that matches the meal name.") }
            recipeID = recipe.rawID
        } else if matches.count == 1 {
            recipeID = matches.first?.id
        } else if matches.count > 1 {
            let scope = try await SiriEnvironment.scope(service)
            let options = matches.map { RecipeEntity(rawID: $0.id, scope: scope, title: $0.title, detail: $0.categoryID ?? $0.id) }
            recipeID = try await $recipe.requestDisambiguation(among: options, dialog: "Which recipe?").rawID
        }
        _ = try await service.saveMeal(type: meal.rawValue, name: name, day: day, recipeID: recipeID)
        SiriEnvironment.changed()
        return .result(dialog: "Set \(meal.rawValue) on \(day.spoken) to \(name).")
    }
}
