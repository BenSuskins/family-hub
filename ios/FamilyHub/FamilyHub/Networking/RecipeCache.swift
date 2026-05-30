import Foundation

/// In-memory cache for recipe metadata, owned by the shared ``APIClient`` so
/// every screen reuses already-loaded recipes instead of refetching.
///
/// The list and per-id detail stores are kept **separate on purpose**: the
/// `GET /api/recipes` list omits `Steps` (summaries only), while
/// `GET /api/recipes/{id}` returns the full recipe. A list summary must never
/// satisfy a detail lookup, or the detail screen would render empty steps.
///
/// An `actor` provides thread-safety since recipe fetches originate from
/// several `@MainActor` view models and can overlap.
actor RecipeCache {
    private var list: [Recipe]?
    private var details: [String: Recipe] = [:]

    func cachedList() -> [Recipe]? { list }

    func cachedDetail(id: String) -> Recipe? { details[id] }

    func storeList(_ recipes: [Recipe]) {
        list = recipes
    }

    /// Store a full recipe and keep the corresponding list entry in sync.
    func storeDetail(_ recipe: Recipe) {
        details[recipe.id] = recipe
        if let index = list?.firstIndex(where: { $0.id == recipe.id }) {
            list?[index] = recipe
        }
    }

    /// Insert or replace a recipe in both stores (used after a create).
    func upsert(_ recipe: Recipe) {
        details[recipe.id] = recipe
        if list == nil {
            return
        }
        if let index = list?.firstIndex(where: { $0.id == recipe.id }) {
            list?[index] = recipe
        } else {
            list?.append(recipe)
        }
    }

    func remove(id: String) {
        details[id] = nil
        list?.removeAll { $0.id == id }
    }

    func invalidateAll() {
        list = nil
        details = [:]
    }
}
