import Foundation
import Security

final class KeychainStore {
    static let shared = KeychainStore()

    private let service: String

    init(service: String = "uk.co.suskins.familyhub") {
        self.service = service
    }

    private enum Key: String {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
    }

    var accessToken: String? { read(.accessToken) }
    var refreshToken: String? { read(.refreshToken) }

    func save(accessToken: String, refreshToken: String) {
        write(accessToken, for: .accessToken)
        write(refreshToken, for: .refreshToken)
    }

    func clear() {
        delete(.accessToken)
        delete(.refreshToken)
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
