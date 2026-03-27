import Foundation

struct OIDCDiscoveryResult {
    let clientID: String
    let authorizationEndpoint: URL
    let tokenEndpoint: URL
}

enum OIDCDiscoveryError: LocalizedError {
    case clientConfigFetchFailed(Int)
    case discoveryDocumentFetchFailed(Int)
    case missingField(String)
    case invalidURL(String)

    var errorDescription: String? {
        switch self {
        case .clientConfigFetchFailed(let code):
            return "Failed to fetch server config (HTTP \(code))"
        case .discoveryDocumentFetchFailed(let code):
            return "Failed to fetch OIDC discovery document (HTTP \(code))"
        case .missingField(let field):
            return "Server config missing required field: \(field)"
        case .invalidURL(let value):
            return "Invalid URL in server config: \(value)"
        }
    }
}

protocol URLSessionProtocol {
    func data(from url: URL) async throws -> (Data, URLResponse)
}

extension URLSession: URLSessionProtocol {}

protocol OIDCDiscoveryService {
    func discover(baseURL: URL) async throws -> OIDCDiscoveryResult
}

final class URLSessionOIDCDiscoveryService: OIDCDiscoveryService {
    private let session: URLSessionProtocol

    init(session: URLSessionProtocol = URLSession.shared) {
        self.session = session
    }

    func discover(baseURL: URL) async throws -> OIDCDiscoveryResult {
        let clientConfig = try await fetchClientConfig(baseURL: baseURL)
        return try await fetchDiscoveryDocument(clientConfig: clientConfig)
    }

    private func fetchClientConfig(baseURL: URL) async throws -> (clientID: String, issuer: URL) {
        let configURL = baseURL.appending(path: "api/client-config")
        let (data, response) = try await session.data(from: configURL)

        let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
        guard statusCode == 200 else {
            throw OIDCDiscoveryError.clientConfigFetchFailed(statusCode)
        }

        let json = try JSONDecoder().decode([String: String].self, from: data)

        guard let clientID = json["clientID"], !clientID.isEmpty else {
            throw OIDCDiscoveryError.missingField("clientID")
        }
        guard let issuerString = json["issuer"], !issuerString.isEmpty else {
            throw OIDCDiscoveryError.missingField("issuer")
        }
        guard let issuerURL = URL(string: issuerString) else {
            throw OIDCDiscoveryError.invalidURL(issuerString)
        }

        return (clientID: clientID, issuer: issuerURL)
    }

    private func fetchDiscoveryDocument(clientConfig: (clientID: String, issuer: URL)) async throws -> OIDCDiscoveryResult {
        let discoveryURL = clientConfig.issuer.appending(path: ".well-known/openid-configuration")
        let (data, response) = try await session.data(from: discoveryURL)

        let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
        guard statusCode == 200 else {
            throw OIDCDiscoveryError.discoveryDocumentFetchFailed(statusCode)
        }

        let json = try JSONDecoder().decode([String: String].self, from: data)

        guard let authString = json["authorization_endpoint"], !authString.isEmpty else {
            throw OIDCDiscoveryError.missingField("authorization_endpoint")
        }
        guard let tokenString = json["token_endpoint"], !tokenString.isEmpty else {
            throw OIDCDiscoveryError.missingField("token_endpoint")
        }
        guard let authURL = URL(string: authString) else {
            throw OIDCDiscoveryError.invalidURL(authString)
        }
        guard let tokenURL = URL(string: tokenString) else {
            throw OIDCDiscoveryError.invalidURL(tokenString)
        }

        return OIDCDiscoveryResult(
            clientID: clientConfig.clientID,
            authorizationEndpoint: authURL,
            tokenEndpoint: tokenURL
        )
    }
}
