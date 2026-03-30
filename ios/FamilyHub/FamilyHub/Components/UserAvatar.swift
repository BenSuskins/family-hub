import SwiftUI
import UIKit

/// Circular avatar showing a user photo or initials fallback.
///
/// Handles two avatar URL formats:
///  - Relative ("/avatar/{id}"): fetched via the authenticated apiClient
///  - Absolute (OIDC provider URL): loaded by AsyncImage
///
/// Usage:
///   UserAvatar(user: user, size: 32, apiClient: apiClient)
///   UserAvatar(user: nil, size: 24)  // shows "?" placeholder
struct UserAvatar: View {
    let user: User?
    let size: CGFloat
    var apiClient: (any APIClientProtocol)?

    @State private var loadedImageData: Data?

    var body: some View {
        Circle()
            .fill(Color(.tertiarySystemFill))
            .frame(width: size, height: size)
            .overlay {
                if let data = loadedImageData, let uiImage = UIImage(data: data) {
                    Image(uiImage: uiImage)
                        .resizable()
                        .scaledToFill()
                        .clipShape(Circle())
                } else if let url = absoluteAvatarURL {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable().scaledToFill()
                        default:
                            initialsLabel
                        }
                    }
                    .clipShape(Circle())
                } else {
                    initialsLabel
                }
            }
            .clipShape(Circle())
            .task(id: user?.id) {
                loadedImageData = nil
                guard let user, user.avatarURL.hasPrefix("/avatar/") else { return }
                let avatarID = String(user.avatarURL.dropFirst("/avatar/".count))
                loadedImageData = try? await apiClient?.fetchUserAvatar(id: avatarID)
            }
    }

    private var absoluteAvatarURL: URL? {
        guard let user, !user.avatarURL.isEmpty, !user.avatarURL.hasPrefix("/") else { return nil }
        return URL(string: user.avatarURL)
    }

    private var initialsLabel: some View {
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
