import Foundation

/// Shared CRUD plumbing for view models whose `state` holds a list of identifiable
/// items. Collapses the repeated "call the API, mutate the loaded array on success,
/// surface `actionError` on failure" dance into a few helpers so each view model
/// only describes *what* changes, not the bookkeeping around it.
@MainActor
protocol MutableListViewModel: AnyObject {
    associatedtype Element: Identifiable
    var state: ViewState<[Element]> { get set }
    var actionError: APIError? { get set }
}

extension MutableListViewModel {
    /// Mutate the loaded array in place. No-op unless `state` is `.loaded`.
    func mutateLoaded(_ change: (inout [Element]) -> Void) {
        guard case .loaded(var items) = state else { return }
        change(&items)
        state = .loaded(items)
    }

    /// Run a create/update call that returns the affected element. On success the
    /// element is folded into the loaded array via `apply`; on failure
    /// `actionError` is set. Returns the element, or `nil` on failure.
    func performMutation(
        _ operation: () async throws -> Element,
        applying apply: (Element, inout [Element]) -> Void
    ) async -> Element? {
        do {
            let element = try await operation()
            mutateLoaded { apply(element, &$0) }
            return element
        } catch {
            actionError = .from(error)
            return nil
        }
    }

    /// Run a delete call. On success the loaded array is updated via `apply`; on
    /// failure `actionError` is set. Returns whether it succeeded.
    func performDeletion(
        _ operation: () async throws -> Void,
        applying apply: (inout [Element]) -> Void
    ) async -> Bool {
        do {
            try await operation()
            mutateLoaded(apply)
            return true
        } catch {
            actionError = .from(error)
            return false
        }
    }
}
