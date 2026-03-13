import Foundation

enum APIError: Error, LocalizedError {
    case network(Error)
    case unauthorized
    case notFound
    case conflict
    case decoding(Error)
    case server(Int)

    var errorDescription: String? {
        switch self {
        case .network(let e): return "Network error: \(e.localizedDescription)"
        case .unauthorized:   return "Unauthorized"
        case .notFound:       return "Not found"
        case .conflict:       return "Conflict"
        case .decoding(let e): return "Decoding error: \(e.localizedDescription)"
        case .server(let code): return "Server error (\(code))"
        }
    }
}
