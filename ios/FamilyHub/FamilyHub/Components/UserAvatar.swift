import SwiftUI

/// Circular avatar showing a user photo (async) or initials fallback.
///
/// Usage:
///   UserAvatar(user: user, size: 32)
///   UserAvatar(user: nil, size: 24)  // shows "?" placeholder
struct UserAvatar: View {
    let user: User?
    let size: CGFloat

    var body: some View {
        Circle()
            .fill(Color(.tertiarySystemFill))
            .frame(width: size, height: size)
            .overlay {
                if let user, !user.avatarURL.isEmpty, let url = URL(string: user.avatarURL) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable().scaledToFill()
                        default:
                            initialsLabel(for: user)
                        }
                    }
                    .clipShape(Circle())
                } else {
                    initialsLabel(for: user)
                }
            }
            .clipShape(Circle())
    }

    private func initialsLabel(for user: User?) -> some View {
        Text(user?.initials ?? "?")
            .font(.system(size: size * 0.38, weight: .semibold))
            .foregroundStyle(.white)
    }
}

#Preview {
    HStack(spacing: 12) {
        UserAvatar(user: User(id: "1", name: "Ben Suskins", email: "", avatarURL: ""), size: 32)
        UserAvatar(user: User(id: "2", name: "Megane Holl", email: "", avatarURL: ""), size: 32)
        UserAvatar(user: nil, size: 32)
        UserAvatar(user: User(id: "1", name: "Ben Suskins", email: "", avatarURL: ""), size: 24)
    }
    .padding()
    .background(Color(.systemBackground))
}
