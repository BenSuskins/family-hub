import SwiftUI

/// Full-page failed state shown when a screen's initial load fails after the
/// networking layer has exhausted its automatic retries. Surfaces the friendly
/// `APIError` copy plus a manual **Try Again** action.
struct ErrorStateView: View {
    let error: APIError
    let retry: () async -> Void

    @State private var isRetrying = false

    private var systemImage: String {
        switch error {
        case .offline:      return "wifi.slash"
        case .timedOut:     return "clock.badge.exclamationmark"
        case .unauthorized: return "person.crop.circle.badge.exclamationmark"
        default:            return "exclamationmark.triangle"
        }
    }

    var body: some View {
        ContentUnavailableView {
            Label(error.errorDescription ?? "Something went wrong", systemImage: systemImage)
        } description: {
            if let suggestion = error.recoverySuggestion {
                Text(suggestion)
            }
        } actions: {
            Button {
                Task {
                    isRetrying = true
                    await retry()
                    isRetrying = false
                }
            } label: {
                if isRetrying {
                    ProgressView()
                } else {
                    Text("Try Again")
                }
            }
            .buttonStyle(.borderedProminent)
            .disabled(isRetrying)
        }
    }
}
