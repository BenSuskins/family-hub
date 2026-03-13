import Foundation

struct OIDCConfig {
    let clientID: String
    let authorizationEndpoint: URL
    let tokenEndpoint: URL
    let baseURL: URL

    static func fromPlist() -> OIDCConfig {
        guard
            let url = Bundle.main.url(forResource: "Config", withExtension: "plist"),
            let dict = NSDictionary(contentsOf: url) as? [String: String],
            let clientID = dict["ClientID"],
            let authEndpoint = dict["AuthorizationEndpoint"].flatMap(URL.init),
            let tokenEndpoint = dict["TokenEndpoint"].flatMap(URL.init),
            let baseURL = dict["BaseURL"].flatMap(URL.init)
        else {
            fatalError("Config.plist missing or invalid — copy Config.example.plist to Config.plist and fill in values")
        }
        return OIDCConfig(
            clientID: clientID,
            authorizationEndpoint: authEndpoint,
            tokenEndpoint: tokenEndpoint,
            baseURL: baseURL
        )
    }
}
