import SwiftUI

struct LoginView: View {
    @Environment(AuthManager.self) private var authManager
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            Text("Family Hub")
                .font(.largeTitle.bold())

            Text("Manage chores, meals, and more.")
                .foregroundStyle(.secondary)

            Spacer()

            if let errorMessage {
                Text(errorMessage)
                    .foregroundStyle(.red)
                    .font(.caption)
            }

            Button {
                Task {
                    isLoading = true
                    errorMessage = nil
                    do {
                        try await authManager.login()
                    } catch {
                        errorMessage = error.localizedDescription
                    }
                    isLoading = false
                }
            } label: {
                if isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                } else {
                    Text("Sign In")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .padding(.horizontal)
            .disabled(isLoading)
        }
        .padding()
    }
}
