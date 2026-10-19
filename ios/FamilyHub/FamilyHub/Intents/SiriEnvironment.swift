import AppIntents
import Foundation
import FamilyHubKit

@MainActor
enum SiriEnvironment {
    static func service() throws -> SiriActionService {
        guard let baseURL = SharedContainer.baseURL, let token = SharedContainer.apiToken, !token.isEmpty else {
            throw SiriActionError.invalid("Open Family Hub and sign in first.")
        }
        let provider = StaticTokenProvider(read: {
            guard SharedContainer.baseURL == baseURL, SharedContainer.apiToken == token else { return nil }
            return token
        })
        return SiriActionService(client: APIClient(baseURL: baseURL, tokenProvider: provider), tokenProvider: provider)
    }

    static func scope(_ service: SiriActionService) async throws -> String {
        let user = try await service.client.fetchMe()
        return Data("\(service.client.serverIdentifier ?? "")|\(user.id)".utf8).base64EncodedString()
    }

    static func validate(scope: String, service: SiriActionService) async throws {
        guard try await self.scope(service) == scope else {
            throw SiriActionError.invalid("Your account or server changed. Select the item again.")
        }
    }

    static func changed() {
        WidgetPublisher.invalidate()
        ShoppingCache().clear()
        NotificationCenter.default.post(name: .familyHubIntentChanged, object: nil)
        FamilyHubShortcuts.updateAppShortcutParameters()
    }

    static func refreshSuggestions() {
        FamilyHubShortcuts.updateAppShortcutParameters()
    }
}

extension Notification.Name {
    static let familyHubIntentChanged = Notification.Name("familyHubIntentChanged")
}
