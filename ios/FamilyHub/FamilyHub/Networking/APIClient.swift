import Foundation

final class APIClient: APIClientProtocol {
    private let baseURL: URL
    private let session: URLSession
    private let retryPolicy: RetryPolicy
    private weak var authManager: AuthManager?
    private let imageCache: NSCache<NSString, NSData> = {
        let c = NSCache<NSString, NSData>()
        c.countLimit = 200
        c.totalCostLimit = 50 * 1024 * 1024 // 50 MB
        return c
    }()

    /// HTTP methods that are safe to retry automatically.
    private static let idempotentMethods: Set<String> = ["GET", "PUT", "DELETE"]

    private static let defaultDecoder = JSONDecoder()
    private static let isoDecoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return decoder
    }()

    init(baseURL: URL, session: URLSession = .shared, authManager: AuthManager, retryPolicy: RetryPolicy = .default) {
        self.baseURL = baseURL
        self.session = session
        self.authManager = authManager
        self.retryPolicy = retryPolicy
    }

    // MARK: - Generic request helpers

    private func get<T: Decodable>(_ path: String, queryItems: [URLQueryItem] = [], decoder: JSONDecoder = APIClient.defaultDecoder) async throws -> T {
        try await send(path: path, method: "GET", queryItems: queryItems, decoder: decoder)
    }

    private func post(_ path: String) async throws {
        try await sendVoid(path: path, method: "POST")
    }

    private func post<T: Decodable>(_ path: String) async throws -> T {
        try await send(path: path, method: "POST")
    }

    private func post<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        try await send(path: path, method: "POST", body: body)
    }

    private func put<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        try await send(path: path, method: "PUT", body: body)
    }

    private func patch(_ path: String, body: some Encodable) async throws {
        try await sendVoid(path: path, method: "PATCH", body: body)
    }

    private func delete(_ path: String, queryItems: [URLQueryItem] = []) async throws {
        try await sendVoid(path: path, method: "DELETE", queryItems: queryItems)
    }

    private func uploadMultipart<T: Decodable>(_ path: String, field: String, data: Data, mimeType: String) async throws -> T {
        let boundary = UUID().uuidString
        var body = Data()
        body.append(Data("--\(boundary)\r\n".utf8))
        body.append(Data("Content-Disposition: form-data; name=\"\(field)\"; filename=\"upload\"\r\n".utf8))
        body.append(Data("Content-Type: \(mimeType)\r\n\r\n".utf8))
        body.append(data)
        body.append(Data("\r\n--\(boundary)--\r\n".utf8))

        let (responseData, _) = try await performValidated(
            path: path,
            method: "POST",
            body: body,
            contentType: "multipart/form-data; boundary=\(boundary)"
        )
        return try decode(T.self, from: responseData)
    }

    // MARK: - Unified request core

    /// Build, send (with retries for idempotent methods), validate, and decode.
    private func send<T: Decodable>(
        path: String,
        method: String,
        body: (any Encodable)? = nil,
        queryItems: [URLQueryItem] = [],
        decoder: JSONDecoder = APIClient.defaultDecoder
    ) async throws -> T {
        let encodedBody = try body.map { try JSONEncoder().encode($0) }
        let (data, _) = try await performValidated(
            path: path,
            method: method,
            queryItems: queryItems,
            body: encodedBody,
            contentType: body == nil ? nil : "application/json"
        )
        return try decode(T.self, from: data, decoder: decoder)
    }

    /// Same as `send` but for endpoints that return no body to decode.
    private func sendVoid(
        path: String,
        method: String,
        body: (any Encodable)? = nil,
        queryItems: [URLQueryItem] = []
    ) async throws {
        let encodedBody = try body.map { try JSONEncoder().encode($0) }
        _ = try await performValidated(
            path: path,
            method: method,
            queryItems: queryItems,
            body: encodedBody,
            contentType: body == nil ? nil : "application/json"
        )
    }

    /// Perform a request, retrying transient failures for idempotent methods, and
    /// map non-2xx responses to `APIError` (signaling 401 globally).
    private func performValidated(
        path: String,
        method: String,
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        contentType: String? = nil
    ) async throws -> (Data, HTTPURLResponse) {
        let isIdempotent = Self.idempotentMethods.contains(method)
        return try await withRetry(
            policy: retryPolicy,
            shouldRetry: { isIdempotent && $0.isRetryable }
        ) {
            let request = try await self.buildRequest(
                path: path, method: method, queryItems: queryItems, body: body, contentType: contentType
            )
            let (data, response) = try await self.perform(request)
            try self.validate(data: data, response: response)
            return (data, response)
        }
    }

    private func buildRequest(
        path: String,
        method: String,
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        contentType: String? = nil
    ) async throws -> URLRequest {
        guard let authManager else { throw APIError.unauthorized }
        let token = try await authManager.validAPIToken()

        guard var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: false) else {
            throw APIError.network(URLError(.badURL))
        }
        if !queryItems.isEmpty { components.queryItems = queryItems }
        guard let requestURL = components.url else {
            throw APIError.network(URLError(.badURL))
        }

        var request = URLRequest(url: requestURL)
        request.httpMethod = method
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        if let contentType { request.setValue(contentType, forHTTPHeaderField: "Content-Type") }
        request.httpBody = body
        return request
    }

    private func perform(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
        do {
            let (data, response) = try await session.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse else {
                throw APIError.network(URLError(.badServerResponse))
            }
            return (data, httpResponse)
        } catch {
            throw APIError.from(error)
        }
    }

    private func decode<T: Decodable>(_ type: T.Type, from data: Data, decoder: JSONDecoder = APIClient.defaultDecoder) throws -> T {
        do {
            return try decoder.decode(type, from: data)
        } catch {
            throw APIError.decoding
        }
    }

    private func validate(data: Data, response: HTTPURLResponse) throws {
        guard let error = Self.mapStatus(data, response) else { return }
        if case .unauthorized = error {
            NotificationCenter.default.post(name: .familyHubUnauthorized, object: nil)
        }
        throw error
    }

    /// Map an HTTP status to an `APIError`, capturing the plain-text server body
    /// for 4xx/5xx responses. Returns `nil` for 2xx (success).
    private static func mapStatus(_ data: Data, _ response: HTTPURLResponse) -> APIError? {
        switch response.statusCode {
        case 200...299: return nil
        case 400, 422:  return .badRequest(serverMessage: bodyText(data))
        case 401:       return .unauthorized
        case 403:       return .forbidden
        case 404:       return .notFound
        case 409:       return .conflict
        case 429:       return .rateLimited(retryAfter: retryAfter(response))
        default:        return .server(status: response.statusCode, serverMessage: bodyText(data))
        }
    }

    private static func bodyText(_ data: Data) -> String? {
        guard let text = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines),
              !text.isEmpty else { return nil }
        return text
    }

    private static func retryAfter(_ response: HTTPURLResponse) -> TimeInterval? {
        guard let value = response.value(forHTTPHeaderField: "Retry-After") else { return nil }
        return TimeInterval(value.trimmingCharacters(in: .whitespaces))
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

    func createChore(_ request: ChoreRequest) async throws -> Chore {
        try await post("api/chores", body: request)
    }

    func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore {
        try await put("api/chores/\(id)", body: request)
    }

    func deleteChore(id: String) async throws {
        try await delete("api/chores/\(id)")
    }

    func fetchUserAvatar(id: String) async throws -> Data {
        let key = "avatar-\(id)" as NSString
        if let cached = imageCache.object(forKey: key) { return cached as Data }
        let (data, _) = try await performValidated(path: "avatar/\(id)", method: "GET")
        imageCache.setObject(data as NSData, forKey: key, cost: data.count)
        return data
    }

    func fetchRecipes() async throws -> [Recipe] {
        try await get("api/recipes")
    }

    func fetchRecipe(id: String) async throws -> Recipe {
        try await get("api/recipes/\(id)")
    }

    func fetchRecipeImage(id: String) async throws -> Data {
        let key = "recipe-\(id)" as NSString
        if let cached = imageCache.object(forKey: key) { return cached as Data }
        let (data, _) = try await performValidated(path: "api/recipes/\(id)/image", method: "GET")
        imageCache.setObject(data as NSData, forKey: key, cost: data.count)
        return data
    }

    func createRecipe(_ request: RecipeRequest) async throws -> Recipe {
        try await post("api/recipes", body: request)
    }

    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe {
        try await put("api/recipes/\(id)", body: request)
    }

    func deleteRecipe(id: String) async throws {
        try await delete("api/recipes/\(id)")
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

        return try await get("api/calendar", queryItems: queryItems, decoder: Self.isoDecoder)
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

    // MARK: - Avatar

    func uploadAvatar(imageData: Data, mimeType: String) async throws -> User {
        try await uploadMultipart("api/profile/avatar", field: "avatar", data: imageData, mimeType: mimeType)
    }

    func deleteAvatar() async throws {
        try await delete("api/profile/avatar")
    }

    // MARK: - Settings

    func fetchSettings() async throws -> AppSettings {
        try await get("api/settings")
    }

    func updateFamilyName(_ name: String) async throws {
        struct Body: Encodable { let family_name: String }
        try await patch("api/settings", body: Body(family_name: name))
    }

    // MARK: - User management

    func promoteUser(id: String) async throws -> User {
        try await post("api/users/\(id)/promote")
    }

    func demoteUser(id: String) async throws -> User {
        try await post("api/users/\(id)/demote")
    }

    // MARK: - Categories

    func fetchCategories() async throws -> [Category] {
        try await get("api/categories")
    }

    func createCategory(name: String) async throws -> Category {
        struct Body: Encodable { let name: String }
        return try await post("api/categories", body: Body(name: name))
    }

    func updateCategory(id: String, name: String) async throws -> Category {
        struct Body: Encodable { let name: String }
        return try await put("api/categories/\(id)", body: Body(name: name))
    }

    func deleteCategory(id: String) async throws {
        try await delete("api/categories/\(id)")
    }

    // MARK: - API tokens

    func fetchTokens() async throws -> [APIToken] {
        try await get("api/tokens", decoder: Self.isoDecoder)
    }

    func createToken(name: String) async throws -> CreatedToken {
        struct Body: Encodable { let name: String }
        return try await post("api/tokens", body: Body(name: name))
    }

    func deleteToken(id: String) async throws {
        try await delete("api/tokens/\(id)")
    }
}
