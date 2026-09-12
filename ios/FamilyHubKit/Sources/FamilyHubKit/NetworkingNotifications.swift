import Foundation

extension Notification.Name {
    /// Posted by ``APIClient`` whenever any request returns 401. Observed by
    /// `AuthManager`, which signs the user out and routes back to login.
    public static let familyHubUnauthorized = Notification.Name("familyHubUnauthorized")
}
