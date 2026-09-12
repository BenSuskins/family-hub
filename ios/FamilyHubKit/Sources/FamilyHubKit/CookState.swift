import Foundation

/// Identifies one ingredient by position.
///
/// Deliberately positional rather than by text. The iOS cook mode keys its
/// checklist on `"\(group.name)-\(item)"` (`CookModeView.swift`), which collides
/// whenever a group repeats an item — "1 tbsp olive oil" listed twice ticks both
/// boxes at once. Indices cannot collide.
public struct IngredientIndex: Hashable, Sendable {
    public let group: Int
    public let item: Int

    public init(group: Int, item: Int) {
        self.group = group
        self.item = item
    }
}

/// The whole state of a cooking session: where you are and what you have ticked.
///
/// A value type with no UI or framework dependencies, so the awkward parts —
/// page maths, the checklist, which step has a timer — are unit tested on the
/// Linux CI job in seconds rather than needing a watch. The watch app wraps this
/// in an `@Observable` holder; this type knows nothing about that.
///
/// Page 0 is the ingredient list; pages 1…n are the steps.
public struct CookState: Equatable, Sendable {
    public let recipe: Recipe
    public private(set) var currentPage: Int
    private var checked: Set<IngredientIndex>

    public init(recipe: Recipe, currentPage: Int = 0, checked: Set<IngredientIndex> = []) {
        self.recipe = recipe
        self.checked = checked
        self.currentPage = 0
        go(to: currentPage)
    }

    // MARK: - Content

    public var steps: [String] { recipe.steps ?? [] }
    public var ingredientGroups: [IngredientGroup] { recipe.ingredients ?? [] }

    /// Ingredients page plus one page per step.
    public var pageCount: Int { 1 + steps.count }

    public var isOnIngredients: Bool { currentPage == 0 }

    /// A recipe with no steps cannot be cooked — the caller should say so rather
    /// than presenting an empty carousel.
    public var isCookable: Bool { !steps.isEmpty }

    // MARK: - Navigation

    /// The step shown on `page`, or nil for the ingredients page or out of range.
    public func stepIndex(forPage page: Int) -> Int? {
        guard page >= 1, page <= steps.count else { return nil }
        return page - 1
    }

    public var currentStepIndex: Int? { stepIndex(forPage: currentPage) }

    public var currentStepText: String? {
        currentStepIndex.map { steps[$0] }
    }

    /// Timer length for a step, in seconds, when the server recognised one.
    /// Positional with `steps`, and tolerant of a short or absent array.
    public func duration(forStep index: Int) -> Int? {
        guard let durations = recipe.stepDurations, durations.indices.contains(index) else { return nil }
        return durations[index]
    }

    public var currentStepDuration: Int? {
        currentStepIndex.flatMap { duration(forStep: $0) }
    }

    public var isOnLastStep: Bool {
        currentStepIndex.map { $0 == steps.count - 1 } ?? false
    }

    /// Clamped so a swipe past either end is a no-op rather than a crash.
    public mutating func go(to page: Int) {
        currentPage = min(max(page, 0), pageCount - 1)
    }

    public mutating func advance() { go(to: currentPage + 1) }
    public mutating func goBack() { go(to: currentPage - 1) }

    // MARK: - Ingredient checklist

    public func isChecked(_ index: IngredientIndex) -> Bool { checked.contains(index) }

    public mutating func toggle(_ index: IngredientIndex) {
        if checked.contains(index) {
            checked.remove(index)
        } else {
            checked.insert(index)
        }
    }

    public var totalIngredientCount: Int {
        ingredientGroups.reduce(0) { $0 + $1.items.count }
    }

    public var checkedIngredientCount: Int { checked.count }

    public var allIngredientsChecked: Bool {
        totalIngredientCount > 0 && checkedIngredientCount == totalIngredientCount
    }
}
