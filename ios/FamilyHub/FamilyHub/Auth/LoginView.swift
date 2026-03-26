import SwiftUI

struct LoginView: View {
    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @State private var isLoading = false

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            Text("Family Hub")
                .font(.largeTitle.bold())

            Text("Manage chores, meals, and more.")
                .foregroundStyle(.secondary)

            Spacer()

            if let error = authManager.loginError {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)
            }

            Button {
                Task {
                    isLoading = true
                    await authManager.login(configStore: configStore)
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
