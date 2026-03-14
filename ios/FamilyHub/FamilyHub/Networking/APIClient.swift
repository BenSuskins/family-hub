import Foundation

final class APIClient: APIClientProtocol {
    private let baseURL: URL
    private let session: URLSession
    private weak var authManager: AuthManager?

    init(baseURL: URL, session: URLSession = .shared, authManager: AuthManager) {
        self.baseURL = baseURL
        self.session = session
        self.authManager = authManager
    }

    // MARK: - Generic request helpers

    private func get<T: Decodable>(_ path: String, queryItems: [URLQueryItem] = []) async throws -> T {
        let request = try await buildRequest(path: path, method: "GET", queryItems: queryItems)
        let (data, response) = try await perform(request)
        return try decode(T.self, from: data, response: response)
    }

    private func post(_ path: String) async throws {
        let request = try await buildRequest(path: path, method: "POST")
        let (_, response) = try await perform(request)
        try validate(response: response)
    }

    private func buildRequest(path: String, method: String, queryItems: [URLQueryItem] = []) async throws -> URLRequest {
        guard let authManager else { throw APIError.unauthorized }
        let token = try await authManager.validAPIToken()

        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false)!
        if !queryItems.isEmpty { components.queryItems = queryItems }

        var request = URLRequest(url: components.url!)
        request.httpMethod = method
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        return request
    }

    private func perform(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
        do {
            let (data, response) = try await session.data(for: request)
            return (data, response as! HTTPURLResponse)
        } catch {
            throw APIError.network(error)
        }
    }

    private func decode<T: Decodable>(_ type: T.Type, from data: Data, response: HTTPURLResponse) throws -> T {
        try validate(response: response)
        do {
            return try JSONDecoder().decode(type, from: data)
        } catch {
            throw APIError.decoding(error)
        }
    }

    private func validate(response: HTTPURLResponse) throws {
        switch response.statusCode {
        case 200...299: return
        case 401: throw APIError.unauthorized
        case 404: throw APIError.notFound
        case 409: throw APIError.conflict
        default: throw APIError.server(response.statusCode)
        }
    }

    // MARK: - APIClientProtocol

    func fetchDashboardStats() async throws -> DashboardStats {
        try await get("api/dashboard")
    }

    func fetchChores() async throws -> [Chore] {
        try await get("api/chores")
    }

    func completeChore(id: String) async throws {
        try await post("api/chores/\(id)/complete")
    }

    func fetchMeals(week: Date) async throws -> [MealPlan] {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return try await get("api/meals", queryItems: [
            URLQueryItem(name: "week", value: formatter.string(from: week))
        ])
    }

    func fetchRecipes() async throws -> [Recipe] {
        try await get("api/recipes")
    }

    func fetchRecipe(id: String) async throws -> Recipe {
        try await get("api/recipes/\(id)")
    }

    func fetchCalendar(month: Date) async throws -> [Chore] {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM"
        let response: CalendarResponse = try await get("api/calendar", queryItems: [
            URLQueryItem(name: "month", value: formatter.string(from: month))
        ])
        return response.chores
    }

    func fetchUsers() async throws -> [User] {
        try await get("/api/users")
    }
}
