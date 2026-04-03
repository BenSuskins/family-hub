import Foundation
import Security

final class KeychainStore {
    nonisolated(unsafe) static let shared = KeychainStore()

    private let service: String

    init(service: String = "uk.co.suskins.familyhub") {
        self.service = service
    }

    private enum Key: String {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case apiToken = "api_token"
    }

    var accessToken: String? { read(.accessToken) }
    var refreshToken: String? { read(.refreshToken) }
    var apiToken: String? { read(.apiToken) }

    func save(accessToken: String, refreshToken: String) {
        write(accessToken, for: .accessToken)
        write(refreshToken, for: .refreshToken)
    }

    func saveAPIToken(_ token: String) {
        write(token, for: .apiToken)
        UserDefaults(suiteName: "group.uk.co.suskins.familyhub")?.set(token, forKey: "api_token")
    }

    func clear() {
        delete(.accessToken)
        delete(.refreshToken)
        delete(.apiToken)
        UserDefaults(suiteName: "group.uk.co.suskins.familyhub")?.removeObject(forKey: "api_token")
    }

    private func read(_ key: Key) -> String? {
        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrService: service,
            kSecAttrAccount: key.rawValue,
            kSecReturnData: true,
            kSecMatchLimit: kSecMatchLimitOne
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func write(_ value: String, for key: Key) {
        guard let data = value.data(using: .utf8) else { return }
        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrService: service,
            kSecAttrAccount: key.rawValue
        ]
        let attributes: [CFString: Any] = [kSecValueData: data]
        if SecItemUpdate(query as CFDictionary, attributes as CFDictionary) == errSecItemNotFound {
            var newItem = query
            newItem[kSecValueData] = data
            SecItemAdd(newItem as CFDictionary, nil)
        }
    }

    private func delete(_ key: Key) {
        let query: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrService: service,
            kSecAttrAccount: key.rawValue
        ]
        SecItemDelete(query as CFDictionary)
    }
}
