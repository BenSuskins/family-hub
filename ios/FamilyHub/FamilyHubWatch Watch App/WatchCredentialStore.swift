import FamilyHubKit
import Foundation
import Security

/// Keychain storage for the credentials the phone sends over.
///
/// Deliberately a separate implementation from the app's `KeychainStore` rather
/// than a shared one in `FamilyHubKit`: sharing it would mean putting `import
/// Security` in the package, which does not exist on Linux and would cost the
/// fast CI job that checks the shared layer in about a minute. A few duplicated
/// lines are a fair price for keeping the package Foundation-only.
///
/// The keychain does not sync between iPhone and Watch — they are separate
/// devices — which is exactly why the phone has to send these at all.
struct WatchCredentialStore {
    private let service = "uk.co.suskins.familyhub.watch"
    private let account = "credentials"

    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    // MARK: - Credentials

    func load() -> WatchCredentials? {
        guard let data = read() else { return nil }
        return try? JSONDecoder.watchLink.decode(WatchCredentials.self, from: data)
    }

    func save(_ credentials: WatchCredentials) {
        // An empty token means the phone signed out; drop it rather than
        // storing a useless value.
        guard !credentials.apiToken.isEmpty else {
            clear()
            return
        }
        guard let data = try? JSONEncoder.watchLink.encode(credentials) else { return }
        write(data)
    }

    func clear() {
        SecItemDelete(baseQuery() as CFDictionary)
    }

    // MARK: - Last recipe

    // Kept in UserDefaults rather than the keychain: a recipe is not a secret,
    // and this is what lets the watch offer "Resume" when opened cold.

    private static let lastRecipeKey = "lastCookHandoff"

    func loadLastHandoff() -> CookHandoff? {
        guard let data = defaults.data(forKey: Self.lastRecipeKey) else { return nil }
        return try? JSONDecoder.watchLink.decode(CookHandoff.self, from: data)
    }

    func saveLastHandoff(_ handoff: CookHandoff) {
        guard let data = try? JSONEncoder.watchLink.encode(handoff) else { return }
        defaults.set(data, forKey: Self.lastRecipeKey)
    }

    func clearLastHandoff() {
        defaults.removeObject(forKey: Self.lastRecipeKey)
    }

    // MARK: - Keychain plumbing

    private func baseQuery() -> [CFString: Any] {
        [
            kSecClass: kSecClassGenericPassword,
            kSecAttrService: service,
            kSecAttrAccount: account,
        ]
    }

    private func read() -> Data? {
        var query = baseQuery()
        query[kSecReturnData] = true
        query[kSecMatchLimit] = kSecMatchLimitOne

        var result: AnyObject?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess else { return nil }
        return result as? Data
    }

    private func write(_ data: Data) {
        let query = baseQuery()
        let attributes: [CFString: Any] = [kSecValueData: data]
        if SecItemUpdate(query as CFDictionary, attributes as CFDictionary) == errSecItemNotFound {
            var newItem = query
            newItem[kSecValueData] = data
            SecItemAdd(newItem as CFDictionary, nil)
        }
    }
}
