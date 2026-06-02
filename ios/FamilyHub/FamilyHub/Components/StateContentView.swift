import SwiftUI

/// Renders a `ViewState<T>` with consistent loading, error and loaded handling
/// so every screen treats async content the same way: a centered spinner while
/// loading, the shared `ErrorStateView` (with retry) on failure, and the
/// caller's content once loaded.
struct StateContentView<T, Content: View>: View {
    let state: ViewState<T>
    let retry: () async -> Void
    @ViewBuilder let content: (T) -> Content

    var body: some View {
        switch state {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity)
                .padding(.vertical, 60)
        case .failed(let error):
            ErrorStateView(error: error, retry: retry)
        case .loaded(let value):
            content(value)
        }
    }
}
