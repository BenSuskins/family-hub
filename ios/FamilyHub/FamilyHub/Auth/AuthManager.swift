import CryptoKit
import Foundation
import AuthenticationServices

@Observable
@MainActor
final class AuthManager: NSObject {
    private(set) var isAuthenticated = false

    var displayName: String { "Family Member" }
    var email: String { "" }

    private let keychain: KeychainStore
    let config: OIDCConfig

    init(keychain: KeychainStore = .shared, config: OIDCConfig = .fromPlist()) {
        self.keychain = keychain
        self.config = config
        super.init()
        self.isAuthenticated = keychain.apiToken != nil
    }

    // MARK: - Login (OIDC/PKCE)

    func login() async throws {
        let (codeVerifier, codeChallenge) = generatePKCE()
        let state = UUID().uuidString

        var components = URLComponents(url: config.authorizationEndpoint, resolvingAgainstBaseURL: false)!
        components.queryItems = [
            URLQueryItem(name: "response_type", value: "code"),
            URLQueryItem(name: "client_id",     value: config.clientID),
            URLQueryItem(name: "redirect_uri",  value: "familyhub://callback"),
            URLQueryItem(name: "scope",         value: "openid profile email"),
            URLQueryItem(name: "state",         value: state),
            URLQueryItem(name: "code_challenge", value: codeChallenge),
            URLQueryItem(name: "code_challenge_method", value: "S256"),
        ]
        let authURL = components.url!

        let callbackURL: URL = try await withCheckedThrowingContinuation { continuation in
            let session = ASWebAuthenticationSession(
                url: authURL,
                callbackURLScheme: "familyhub"
            ) { url, error in
                if let error { continuation.resume(throwing: error) }
                else if let url { continuation.resume(returning: url) }
                else { continuation.resume(throwing: AuthError.cancelled) }
            }
            session.presentationContextProvider = self
            session.prefersEphemeralWebBrowserSession = false
            session.start()
        }

        guard
            let callbackComponents = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false),
            let code = callbackComponents.queryItems?.first(where: { $0.name == "code" })?.value
        else {
            throw AuthError.invalidCallback
        }

        try await exchangeCode(code, codeVerifier: codeVerifier)
    }

    // MARK: - Token management

    func validAPIToken() async throws -> String {
        guard let token = keychain.apiToken else {
            isAuthenticated = false
            throw APIError.unauthorized
        }
        return token
    }

    func logout() {
        keychain.clear()
        isAuthenticated = false
    }

    // MARK: - Private helpers

    private func exchangeCode(_ code: String, codeVerifier: String) async throws {
        var request = URLRequest(url: config.tokenEndpoint)
        request.httpMethod = "POST"
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")

        var params = URLComponents()
        params.queryItems = [
            URLQueryItem(name: "grant_type",    value: "authorization_code"),
            URLQueryItem(name: "client_id",     value: config.clientID),
            URLQueryItem(name: "code",          value: code),
            URLQueryItem(name: "redirect_uri",  value: "familyhub://callback"),
            URLQueryItem(name: "code_verifier", value: codeVerifier),
        ]
        request.httpBody = params.query?.data(using: .utf8)

        let (data, _) = try await URLSession.shared.data(for: request)
        let response = try JSONDecoder().decode(TokenResponse.self, from: data)
        keychain.save(accessToken: response.accessToken, refreshToken: response.refreshToken ?? "")

        try await exchangeForAPIToken(oidcAccessToken: response.accessToken)
    }

    private func exchangeForAPIToken(oidcAccessToken: String) async throws {
        let exchangeURL = config.baseURL.appendingPathComponent("api/auth/exchange")
        var request = URLRequest(url: exchangeURL)
        request.httpMethod = "POST"
        request.setValue("Bearer \(oidcAccessToken)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw APIError.unauthorized
        }

        struct ExchangeResponse: Decodable { let token: String }
        let exchangeResponse = try JSONDecoder().decode(ExchangeResponse.self, from: data)
        keychain.saveAPIToken(exchangeResponse.token)
        isAuthenticated = true
    }

    private func generatePKCE() -> (verifier: String, challenge: String) {
        var bytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        let verifier = Data(bytes).base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        let challenge = Data(SHA256.hash(data: Data(verifier.utf8)))
            .base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
        return (verifier, challenge)
    }
}

// MARK: - ASWebAuthenticationPresentationContextProviding
extension AuthManager: ASWebAuthenticationPresentationContextProviding {
    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap { $0.windows }
            .first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}

// MARK: - Supporting types
enum AuthError: Error {
    case cancelled
    case invalidCallback
}

private struct TokenResponse: Decodable {
    let accessToken: String
    let refreshToken: String?

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
    }
}
