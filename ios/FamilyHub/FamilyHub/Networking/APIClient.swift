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

    private func post<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        var request = try await buildRequest(path: path, method: "POST")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await perform(request)
        return try decode(T.self, from: data, response: response)
    }

    private func delete(_ path: String, queryItems: [URLQueryItem] = []) async throws {
        let request = try await buildRequest(path: path, method: "DELETE", queryItems: queryItems)
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

    func fetchRecipeImage(id: String) async throws -> Data {
        let request = try await buildRequest(path: "api/recipes/\(id)/image", method: "GET")
        let (data, response) = try await perform(request)
        try validate(response: response)
        return data
    }

    func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")

        var queryItems = [URLQueryItem(name: "view", value: view)]
        if view == "month" {
            formatter.dateFormat = "yyyy-MM"
            queryItems.append(URLQueryItem(name: "month", value: formatter.string(from: date)))
        } else {
            formatter.dateFormat = "yyyy-MM-dd"
            queryItems.append(URLQueryItem(name: "date", value: formatter.string(from: date)))
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601

        let request = try await buildRequest(path: "api/calendar", method: "GET", queryItems: queryItems)
        let (data, response) = try await perform(request)
        try validate(response: response)
        return try decoder.decode(CalendarResponse.self, from: data)
    }

    func fetchUsers() async throws -> [User] {
        try await get("/api/users")
    }

    func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan {
        struct SaveMealBody: Encodable {
            let date: String
            let mealType: String
            let name: String
            let recipeID: String?
        }
        return try await post("api/meals", body: SaveMealBody(date: date, mealType: mealType, name: name, recipeID: recipeID))
    }

    func deleteMeal(date: String, mealType: String) async throws {
        try await delete("api/meals", queryItems: [
            URLQueryItem(name: "date", value: date),
            URLQueryItem(name: "mealType", value: mealType)
        ])
    }

    func fetchMe() async throws -> User {
        try await get("api/me")
    }
}
