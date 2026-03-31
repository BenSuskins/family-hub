import Foundation

@Observable @MainActor
final class ConfigStore {
    var baseURL: String
    var clientID: String
    var authorizationEndpoint: String
    var tokenEndpoint: String

    var isConfigured: Bool {
        ![baseURL, clientID, authorizationEndpoint, tokenEndpoint]
            .contains(where: \.isEmpty)
    }

    private let defaults: UserDefaults

    init(defaults: UserDefaults = UserDefaults(suiteName: "group.uk.co.suskins.familyhub")!) {
        self.defaults = defaults
        self.baseURL = defaults.string(forKey: "baseURL") ?? ""
        self.clientID = defaults.string(forKey: "clientID") ?? ""
        self.authorizationEndpoint = defaults.string(forKey: "authorizationEndpoint") ?? ""
        self.tokenEndpoint = defaults.string(forKey: "tokenEndpoint") ?? ""
    }

    func applyDiscovery(_ result: OIDCDiscoveryResult) {
        clientID = result.clientID
        authorizationEndpoint = result.authorizationEndpoint.absoluteString
        tokenEndpoint = result.tokenEndpoint.absoluteString
    }

    func save() {
        defaults.set(baseURL, forKey: "baseURL")
        defaults.set(clientID, forKey: "clientID")
        defaults.set(authorizationEndpoint, forKey: "authorizationEndpoint")
        defaults.set(tokenEndpoint, forKey: "tokenEndpoint")
    }
}
