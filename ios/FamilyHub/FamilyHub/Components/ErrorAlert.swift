import SwiftUI

/// Presents a dismissible alert whenever a bound `APIError` becomes non-nil.
/// Used for mutation failures (create/update/delete) across the app so every
/// action surfaces the same friendly copy.
private struct ErrorAlertModifier: ViewModifier {
    @Binding var error: APIError?

    func body(content: Content) -> some View {
        content.alert(
            "Something went wrong",
            isPresented: Binding(
                get: { error != nil },
                set: { if !$0 { error = nil } }
            ),
            presenting: error
        ) { _ in
            Button("OK", role: .cancel) { error = nil }
        } message: { error in
            Text(error.errorDescription ?? "Please try again.")
        }
    }
}

extension View {
    /// Show a standard error alert driven by a bound optional `APIError`.
    func errorAlert(_ error: Binding<APIError?>) -> some View {
        modifier(ErrorAlertModifier(error: error))
    }
}
