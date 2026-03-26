import Foundation

struct OIDCConfig {
    let clientID: String
    let authorizationEndpoint: URL
    let tokenEndpoint: URL
    let baseURL: URL

    static func from(configStore: ConfigStore) throws(ConfigurationError) -> OIDCConfig {
        guard !configStore.clientID.isEmpty else { throw .emptyField("Client ID") }
        guard !configStore.authorizationEndpoint.isEmpty else { throw .emptyField("Authorization Endpoint") }
        guard !configStore.tokenEndpoint.isEmpty else { throw .emptyField("Token Endpoint") }
        guard !configStore.baseURL.isEmpty else { throw .emptyField("Server URL") }

        guard let baseURL = URL(string: configStore.baseURL),
              baseURL.scheme == "http" || baseURL.scheme == "https"
        else { throw .invalidURL("Server URL", configStore.baseURL) }

        guard let authEndpoint = URL(string: configStore.authorizationEndpoint),
              authEndpoint.scheme == "http" || authEndpoint.scheme == "https"
        else { throw .invalidURL("Authorization Endpoint", configStore.authorizationEndpoint) }

        guard let tokenEndpoint = URL(string: configStore.tokenEndpoint),
              tokenEndpoint.scheme == "http" || tokenEndpoint.scheme == "https"
        else { throw .invalidURL("Token Endpoint", configStore.tokenEndpoint) }

        return OIDCConfig(
            clientID: configStore.clientID,
            authorizationEndpoint: authEndpoint,
            tokenEndpoint: tokenEndpoint,
            baseURL: baseURL
        )
    }
}

enum ConfigurationError: LocalizedError {
    case emptyField(String)
    case invalidURL(String, String)

    var errorDescription: String? {
        switch self {
        case .emptyField(let field):
            return "\(field) must not be empty"
        case .invalidURL(let field, let value):
            return "\(field) \"\(value)\" is not a valid http or https URL"
        }
    }
}
