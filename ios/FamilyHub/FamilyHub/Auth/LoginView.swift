import SwiftUI

struct LoginView: View {
    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @State private var isLoading = false
    @State private var appeared = false

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            VStack(spacing: 16) {
                Image(systemName: "house.fill")
                    .font(.system(size: 56))
                    .foregroundStyle(.tint)
                    .symbolEffect(.pulse, options: .repeating.speed(0.5), isActive: isLoading)

                Text("Family Hub")
                    .font(.largeTitle.bold())

                Text("Manage chores, meals, and more.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .opacity(appeared ? 1 : 0)
            .offset(y: appeared ? 0 : 20)

            Spacer()

            VStack(spacing: 12) {
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
                .controlSize(.large)
                .disabled(isLoading)
            }
            .padding(.horizontal)
            .opacity(appeared ? 1 : 0)
        }
        .padding()
        .background(.regularMaterial)
        .onAppear {
            withAnimation(.spring(duration: 0.6)) {
                appeared = true
            }
        }
    }
}
