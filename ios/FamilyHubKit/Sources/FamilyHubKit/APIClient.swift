import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif

public final class APIClient: APIClientProtocol {
    // Three deliberately distinct caches, each matched to its data's lifetime:
    //   • recipeCache — an actor for thread-safe recipe metadata reused across
    //     screens (list vs. detail kept separate; see RecipeCache).
    //   • imageCache — an NSCache so avatar/recipe image bytes are evicted under
    //     memory pressure rather than held forever.
    //   • the calendar view model keeps its own per-key response dictionary, as
    //     that caching is view-state specific (month/week/day) and short-lived.
    private let baseURL: URL
    private let session: URLSession
    private let retryPolicy: RetryPolicy
    private weak var tokenProvider: (any TokenProviding)?
    private let recipeCache = RecipeCache()
    private let snapshotCache = SnapshotCache()
    private let diskImageCache = DiskImageCache()
    private var lastCacheNamespace: String?
    private var cacheGeneration = 0
    private var inFlightGets: [String: (id: UUID, task: Task<Data, Error>)] = [:]
    private let freshness = CacheFreshness()
    private let imageCache: NSCache<NSString, NSData> = {
        let c = NSCache<NSString, NSData>()
        c.countLimit = 200
        c.totalCostLimit = 50 * 1024 * 1024 // 50 MB
        return c
    }()

    /// HTTP methods that are safe to retry automatically.
    private static let idempotentMethods: Set<String> = ["GET", "PUT", "DELETE"]

    // nonisolated because both are used as default arguments, and default
    // argument expressions are evaluated in the caller's nonisolated context.
    // Safe: each is immutable and only ever read.
    nonisolated private static let defaultDecoder = JSONDecoder()
    nonisolated private static let isoDecoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return decoder
    }()

    public init(baseURL: URL, session: URLSession = .shared, tokenProvider: any TokenProviding, retryPolicy: RetryPolicy = .default) {
        self.baseURL = baseURL
        self.session = session
        self.tokenProvider = tokenProvider
        self.retryPolicy = retryPolicy
    }

    // MARK: - Generic request helpers

    private func get<T: Decodable>(_ path: String, queryItems: [URLQueryItem] = [], decoder: JSONDecoder = APIClient.defaultDecoder) async throws -> T {
        try await send(path: path, method: "GET", queryItems: queryItems, decoder: decoder)
    }

    private func cached<T: Codable>(
        key: String,
        freshness: TimeInterval,
        forceRefresh: Bool = false,
        operation: @MainActor @escaping () async throws -> T
    ) async throws -> T {
        guard let namespace = await cacheNamespace() else { return try await operation() }
        let generation = cacheGeneration
        let snapshot = await snapshotCache.read(namespace: namespace, key: key)
        guard cacheGeneration == generation, lastCacheNamespace == namespace else { return try await operation() }
        if !forceRefresh, let snapshot, snapshot.metadata.isFresh,
           let value = try? JSONDecoder().decode(T.self, from: snapshot.data) { return value }
        do {
            let requestKey = "\(namespace)|\(key)"
            let requestID: UUID
            let request: Task<Data, Error>
            if let existing = inFlightGets[requestKey] {
                requestID = existing.id
                request = existing.task
            } else {
                requestID = UUID()
                request = Task { @MainActor in try JSONEncoder().encode(try await operation()) }
                inFlightGets[requestKey] = (requestID, request)
            }
            let data: Data
            do {
                data = try await request.value
            } catch {
                if inFlightGets[requestKey]?.id == requestID { inFlightGets[requestKey] = nil }
                throw error
            }
            if inFlightGets[requestKey]?.id == requestID { inFlightGets[requestKey] = nil }
            if cacheGeneration == generation, lastCacheNamespace == namespace {
                await snapshotCache.write(data, namespace: namespace, key: key, freshness: freshness, generation: generation)
            }
            return try JSONDecoder().decode(T.self, from: data)
        } catch is CancellationError {
            throw CancellationError()
        } catch {
            if let snapshot, let value = try? JSONDecoder().decode(T.self, from: snapshot.data) { return value }
            throw error
        }
    }

    private func cacheNamespace() async -> String? {
        guard let tokenProvider,
              let identity = try? await tokenProvider.cacheIdentity() else { return nil }
        let namespace = "\(baseURL.absoluteString)|\(Self.cacheScope(identity))"
        if let previous = lastCacheNamespace, previous != namespace {
            cacheGeneration += 1
            await snapshotCache.remove(namespace: previous, generation: cacheGeneration)
            await diskImageCache.remove(namespace: previous, generation: cacheGeneration)
            await recipeCache.invalidateAll()
            imageCache.removeAllObjects()
        }
        lastCacheNamespace = namespace
        await snapshotCache.activate(namespace: namespace, generation: cacheGeneration)
        await diskImageCache.activate(namespace: namespace, generation: cacheGeneration)
        return namespace
    }

    private static func cacheScope(_ identity: String) -> String {
        let hash = identity.utf8.reduce(into: UInt64(14_695_981_039_346_656_037)) { value, byte in
            value = (value ^ UInt64(byte)) &* 1_099_511_628_211
        }
        return String(hash, radix: 16)
    }

    private func loadImage(path: String, queryItems: [URLQueryItem] = [], namespace: String?, key: String) async throws -> Data {
        let requestKey = "\(namespace ?? baseURL.absoluteString)|image|\(key)"
        let generation = cacheGeneration
        let requestID: UUID
        let request: Task<Data, Error>
        if let existing = inFlightGets[requestKey] {
            requestID = existing.id
            request = existing.task
        } else {
            requestID = UUID()
            request = Task { @MainActor in
                let (data, _) = try await self.performValidated(path: path, method: "GET", queryItems: queryItems)
                return data
            }
            inFlightGets[requestKey] = (requestID, request)
        }
        do {
            let data = try await request.value
            if inFlightGets[requestKey]?.id == requestID { inFlightGets[requestKey] = nil }
            if let namespace, cacheGeneration == generation, lastCacheNamespace == namespace {
                await diskImageCache.write(data, namespace: namespace, key: key, generation: generation)
            }
            return data
        } catch {
            if inFlightGets[requestKey]?.id == requestID { inFlightGets[requestKey] = nil }
            throw error
        }
    }

    public func snapshotMetadata(for key: String) async -> SnapshotMetadata? {
        guard let namespace = await cacheNamespace() else { return nil }
        return await snapshotCache.read(namespace: namespace, key: key)?.metadata
    }

    private func updateSnapshot<Value: Codable>(
        key: String,
        freshness: TimeInterval,
        update: (inout Value) -> Void
    ) async {
        guard let namespace = await cacheNamespace(),
              let snapshot = await snapshotCache.read(namespace: namespace, key: key) else { return }
        let generation = cacheGeneration
        guard var value = try? JSONDecoder().decode(Value.self, from: snapshot.data) else { return }
        update(&value)
        guard let data = try? JSONEncoder().encode(value) else { return }
        guard cacheGeneration == generation, lastCacheNamespace == namespace else { return }
        await snapshotCache.write(data, namespace: namespace, key: key, freshness: freshness, generation: generation)
    }

    private func writeSnapshot<Value: Codable>(_ value: Value, key: String, freshness: TimeInterval) async {
        guard let namespace = await cacheNamespace() else { return }
        let generation = cacheGeneration
        guard let data = try? JSONEncoder().encode(value),
              cacheGeneration == generation, lastCacheNamespace == namespace else { return }
        await snapshotCache.write(data, namespace: namespace, key: key, freshness: freshness, generation: generation)
    }

    private func removeSnapshot(key: String) async {
        guard let namespace = await cacheNamespace() else { return }
        await snapshotCache.remove(namespace: namespace, key: key)
    }

    private func invalidateDashboardAndCalendar(on date: Date = Date()) async {
        await removeSnapshot(key: "dashboard-\(APIDate.dayString(date))")
        await removeSnapshot(key: "calendar-\(CalendarSchedule().monthKey(date))")
    }

    private func mealWeekKeys(_ dateString: String) -> [String] {
        guard let date = APIDate.day.date(from: dateString) else { return [] }
        return (0..<7).compactMap { offset in
            Calendar(identifier: .gregorian).date(byAdding: .day, value: -offset, to: date).map(APIDate.dayString)
        }
    }

    private func removeRecipeImages(id: String) async {
        guard let namespace = await cacheNamespace() else { return }
        for variant in RecipeImageVariant.allCases {
            let key = "recipe-\(variant.rawValue)-\(id)"
            await diskImageCache.remove(namespace: namespace, key: key)
            imageCache.removeObject(forKey: "\(namespace)-\(key)" as NSString)
        }
    }

    public func clearAccountCache() async {
        cacheGeneration += 1
        if let namespace = lastCacheNamespace {
            await snapshotCache.remove(namespace: namespace, generation: cacheGeneration)
            await diskImageCache.remove(namespace: namespace, generation: cacheGeneration)
        }
        await recipeCache.invalidateAll()
        imageCache.removeAllObjects()
        lastCacheNamespace = nil
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

    private func patch<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        try await send(path: path, method: "PATCH", body: body)
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

    private func uploadMultipartFile(_ path: String, field: String, fileURL: URL, mimeType: String) async throws {
        let boundary = UUID().uuidString
        let filename = fileURL.lastPathComponent.replacingOccurrences(of: "\"", with: "")
        let bodyURL = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: bodyURL) }

        FileManager.default.createFile(atPath: bodyURL.path, contents: nil)
        let output = try FileHandle(forWritingTo: bodyURL)
        let input = try FileHandle(forReadingFrom: fileURL)
        defer {
            try? input.close()
            try? output.close()
        }
        output.write(Data("--\(boundary)\r\nContent-Disposition: form-data; name=\"\(field)\"; filename=\"\(filename)\"\r\nContent-Type: \(mimeType)\r\n\r\n".utf8))
        while let chunk = try input.read(upToCount: 1024 * 1024), !chunk.isEmpty {
            output.write(chunk)
        }
        output.write(Data("\r\n--\(boundary)--\r\n".utf8))

        try await performFileUploadValidated(
            path: path,
            method: "POST",
            bodyFileURL: bodyURL,
            contentType: "multipart/form-data; boundary=\(boundary)"
        )
    }

    private func performFileUploadValidated(path: String, method: String, bodyFileURL: URL, contentType: String) async throws {
        _ = try await withRetry(
            policy: retryPolicy,
            shouldRetry: { Self.idempotentMethods.contains(method) && $0.isRetryable }
        ) {
            let request = try await self.buildRequest(path: path, method: method, contentType: contentType)
            do {
#if os(Linux)
                var uploadRequest = request
                uploadRequest.httpBody = try Data(contentsOf: bodyFileURL)
                let (data, response) = try await self.session.data(for: uploadRequest)
#else
                let (data, response) = try await self.session.upload(for: request, fromFile: bodyFileURL)
#endif
                guard let httpResponse = response as? HTTPURLResponse else {
                    throw APIError.network(URLError(.badServerResponse))
                }
                try self.validate(data: data, response: httpResponse)
                return (data, httpResponse)
            } catch {
                throw APIError.from(error)
            }
        }
    }

    private func downloadFile(_ path: String) async throws -> URL {
        try await withRetry(policy: retryPolicy, shouldRetry: { $0.isRetryable }) {
            let request = try await self.buildRequest(path: path, method: "GET")
            do {
#if os(Linux)
                let (data, response) = try await self.session.data(for: request)
                let temporaryURL = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
                try data.write(to: temporaryURL, options: .atomic)
#else
                let (temporaryURL, response) = try await self.session.download(for: request)
#endif
                defer { try? FileManager.default.removeItem(at: temporaryURL) }
                guard let httpResponse = response as? HTTPURLResponse else {
                    throw APIError.network(URLError(.badServerResponse))
                }
                guard (200...299).contains(httpResponse.statusCode) else {
                    try self.validate(data: Data(contentsOf: temporaryURL), response: httpResponse)
                    throw APIError.server(status: httpResponse.statusCode, serverMessage: nil)
                }

                let name = Self.downloadFilename(httpResponse) ?? "family-hub-backup.db.gz"
                let directory = FileManager.default.temporaryDirectory.appendingPathComponent("family-hub-\(UUID().uuidString)", isDirectory: true)
                try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
                let destination = directory.appendingPathComponent(name)
                try FileManager.default.moveItem(at: temporaryURL, to: destination)
                return destination
            } catch {
                throw APIError.from(error)
            }
        }
    }

    private static func downloadFilename(_ response: HTTPURLResponse) -> String? {
        guard let disposition = response.value(forHTTPHeaderField: "Content-Disposition"),
              let component = disposition.split(separator: ";").map(String.init).first(where: { $0.trimmingCharacters(in: .whitespaces).hasPrefix("filename=") }) else {
            return nil
        }
        let value = component.trimmingCharacters(in: .whitespaces).replacingOccurrences(of: "filename=", with: "")
        let filename = value.trimmingCharacters(in: CharacterSet(charactersIn: "\" "))
        let safeFilename = URL(fileURLWithPath: filename).lastPathComponent
        return safeFilename.isEmpty ? nil : safeFilename
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
        guard let tokenProvider else { throw APIError.unauthorized }
        let token = try await tokenProvider.validAPIToken()

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
        let trimmedValue = value.trimmingCharacters(in: .whitespaces)
        if let seconds = TimeInterval(trimmedValue) { return max(0, seconds) }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "EEE',' dd MMM yyyy HH':'mm':'ss z"
        return formatter.date(from: trimmedValue).map { max(0, $0.timeIntervalSinceNow) }
    }

    // MARK: - APIClientProtocol

    public func fetchDashboardStats() async throws -> DashboardStats {
        try await fetchDashboardStats(forceRefresh: false)
    }

    public func fetchDashboardStats(forceRefresh: Bool) async throws -> DashboardStats {
        let date = APIDate.dayString(Date())
        return try await cached(key: "dashboard-\(date)", freshness: freshness.dashboard, forceRefresh: forceRefresh) {
            try await self.get("api/dashboard")
        }
    }

    public func fetchChores() async throws -> [Chore] {
        try await cached(key: "chores", freshness: freshness.chores) { try await self.get("api/chores") }
    }

    public func completeChore(id: String) async throws {
        try await post("api/chores/\(id)/complete")
        await removeSnapshot(key: "chores")
        await invalidateDashboardAndCalendar()
    }

    public func fetchMeals(week: Date) async throws -> [MealPlan] {
        try await fetchMeals(week: week, forceRefresh: false)
    }

    public func fetchMeals(week: Date, forceRefresh: Bool) async throws -> [MealPlan] {
        let weekKey = APIDate.dayString(week)
        return try await cached(key: "meals-\(weekKey)", freshness: freshness.meals, forceRefresh: forceRefresh) {
            try await self.get("api/meals", queryItems: [URLQueryItem(name: "week", value: weekKey)])
        }
    }

    public func createChore(_ request: ChoreRequest) async throws -> Chore {
        let chore: Chore = try await post("api/chores", body: request)
        await removeSnapshot(key: "chores")
        await invalidateDashboardAndCalendar()
        return chore
    }

    public func updateChore(id: String, _ request: ChoreRequest) async throws -> Chore {
        let chore: Chore = try await put("api/chores/\(id)", body: request)
        await removeSnapshot(key: "chores")
        await invalidateDashboardAndCalendar()
        return chore
    }

    public func deleteChore(id: String) async throws {
        try await delete("api/chores/\(id)")
        await removeSnapshot(key: "chores")
        await invalidateDashboardAndCalendar()
    }

    public func fetchUserAvatar(id: String) async throws -> Data {
        let namespace = await cacheNamespace()
        let generation = cacheGeneration
        let key = "\(namespace ?? "")-avatar-\(id)" as NSString
        if let cached = imageCache.object(forKey: key) { return cached as Data }
        if let namespace, let cached = await diskImageCache.read(namespace: namespace, key: "avatar-\(id)") {
            guard cacheGeneration == generation, lastCacheNamespace == namespace else { return try await fetchUserAvatar(id: id) }
            imageCache.setObject(cached as NSData, forKey: key, cost: cached.count)
            return cached
        }
        let data = try await loadImage(path: "avatar/\(id)", namespace: namespace, key: "avatar-\(id)")
        imageCache.setObject(data as NSData, forKey: key, cost: data.count)
        return data
    }

    public func fetchRecipes(forceRefresh: Bool) async throws -> [Recipe] {
        let recipes: [Recipe] = try await cached(key: "recipes-summary", freshness: freshness.recipes, forceRefresh: forceRefresh) {
            try await self.get("api/recipes")
        }
        await recipeCache.storeList(recipes)
        return recipes
    }

    public func fetchRecipe(id: String) async throws -> Recipe {
        let recipe: Recipe = try await cached(key: "recipe-detail-\(id)", freshness: freshness.recipes) {
            try await self.get("api/recipes/\(id)")
        }
        await recipeCache.storeDetail(recipe)
        return recipe
    }

    public func fetchRecipeImage(id: String) async throws -> Data {
        try await fetchRecipeImage(id: id, variant: .original)
    }

    public func fetchRecipeImage(id: String, variant: RecipeImageVariant) async throws -> Data {
        let namespace = await cacheNamespace()
        let generation = cacheGeneration
        let imageKey = "recipe-\(variant.rawValue)-\(id)"
        let key = "\(namespace ?? "")-\(imageKey)" as NSString
        if let cached = imageCache.object(forKey: key) { return cached as Data }
        if let namespace, let cached = await diskImageCache.read(namespace: namespace, key: imageKey) {
            guard cacheGeneration == generation, lastCacheNamespace == namespace else { return try await fetchRecipeImage(id: id, variant: variant) }
            imageCache.setObject(cached as NSData, forKey: key, cost: cached.count)
            return cached
        }
        let queryItems = variant == .thumbnail ? [URLQueryItem(name: "size", value: "thumb")] : []
        let data = try await loadImage(path: "api/recipes/\(id)/image", queryItems: queryItems, namespace: namespace, key: imageKey)
        imageCache.setObject(data as NSData, forKey: key, cost: data.count)
        return data
    }

    public func createRecipe(_ request: RecipeRequest) async throws -> Recipe {
        let created: Recipe = try await post("api/recipes", body: request)
        await recipeCache.upsert(created)
        await updateSnapshot(key: "recipes-summary", freshness: freshness.recipes) { (recipes: inout [Recipe]) in
            if let index = recipes.firstIndex(where: { $0.id == created.id }) { recipes[index] = created }
            else { recipes.append(created) }
        }
        await writeSnapshot(created, key: "recipe-detail-\(created.id)", freshness: freshness.recipes)
        return created
    }

    public func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe {
        let updated: Recipe = try await put("api/recipes/\(id)", body: request)
        await recipeCache.storeDetail(updated)
        await updateSnapshot(key: "recipes-summary", freshness: freshness.recipes) { (recipes: inout [Recipe]) in
            if let index = recipes.firstIndex(where: { $0.id == id }) { recipes[index] = updated }
        }
        await writeSnapshot(updated, key: "recipe-detail-\(id)", freshness: freshness.recipes)
        await removeRecipeImages(id: id)
        return updated
    }

    public func deleteRecipe(id: String) async throws {
        try await delete("api/recipes/\(id)")
        await recipeCache.remove(id: id)
        await updateSnapshot(key: "recipes-summary", freshness: freshness.recipes) { (recipes: inout [Recipe]) in recipes.removeAll { $0.id == id } }
        await removeSnapshot(key: "recipe-detail-\(id)")
        await removeRecipeImages(id: id)
    }

    public func fetchCalendar(view: String, date: Date) async throws -> CalendarResponse {
        try await fetchCalendar(view: view, date: date, forceRefresh: false)
    }

    public func fetchCalendar(view: String, date: Date, forceRefresh: Bool) async throws -> CalendarResponse {
        var queryItems = [URLQueryItem(name: "view", value: view)]
        if view == "month" {
            queryItems.append(URLQueryItem(name: "month", value: CalendarSchedule().monthKey(date)))
        } else {
            queryItems.append(URLQueryItem(name: "date", value: CalendarSchedule().dayKey(date)))
        }
        if view == "month" {
            let monthKey = CalendarSchedule().monthKey(date)
            return try await cached(key: "calendar-\(monthKey)", freshness: freshness.calendar, forceRefresh: forceRefresh) {
                try await self.get("api/calendar", queryItems: queryItems, decoder: Self.isoDecoder)
            }
        }
        return try await get("api/calendar", queryItems: queryItems, decoder: Self.isoDecoder)
    }

    public func fetchCalendarSubscriptions() async throws -> [CalendarSubscription] {
        try await get("api/calendars", decoder: Self.isoDecoder)
    }

    public func createCalendarSubscription(name: String, url: String, color: String) async throws -> CalendarSubscription {
        struct Request: Encodable { let name: String; let url: String; let color: String }
        return try await send(path: "api/calendars", method: "POST", body: Request(name: name, url: url, color: color), decoder: Self.isoDecoder)
    }

    public func updateCalendarSubscriptionColor(id: String, color: String) async throws -> CalendarSubscription {
        struct Request: Encodable { let color: String }
        return try await send(path: "api/calendars/\(id)/color", method: "PATCH", body: Request(color: color), decoder: Self.isoDecoder)
    }

    public func refreshCalendarSubscription(id: String) async throws {
        try await post("api/calendars/\(id)/refresh")
    }

    public func deleteCalendarSubscription(id: String) async throws {
        try await delete("api/calendars/\(id)")
    }

    public func downloadDatabaseBackup() async throws -> URL {
        try await downloadFile("api/admin/backup")
    }

    public func restoreDatabaseBackup(from url: URL) async throws {
        try await uploadMultipartFile("api/admin/restore", field: "backup", fileURL: url, mimeType: "application/gzip")
    }

    public func fetchBinSchedule() async throws -> BinScheduleSettings {
        try await get("api/bin-collections")
    }

    public func uploadBinSchedule(from url: URL) async throws -> BinScheduleSettings {
        let body = try Data(contentsOf: url)
        return try await uploadMultipart("api/bin-collections", field: "schedule", data: body, mimeType: "text/calendar")
    }

    public func updateBinReminders(enabled: Bool) async throws -> BinScheduleSettings {
        struct Request: Encodable { let enabled: Bool }
        return try await patch("api/bin-collections", body: Request(enabled: enabled))
    }

    public func fetchUsers() async throws -> [User] {
        try await fetchUsers(forceRefresh: false)
    }

    public func fetchUsers(forceRefresh: Bool) async throws -> [User] {
        try await cached(key: "users", freshness: freshness.users, forceRefresh: forceRefresh) { try await self.get("/api/users") }
    }

    public func saveMeal(date: String, mealType: String, name: String, recipeID: String?) async throws -> MealPlan {
        struct SaveMealBody: Encodable {
            let date: String
            let mealType: String
            let name: String
            let recipeID: String?
        }
        let meal: MealPlan = try await post("api/meals", body: SaveMealBody(date: date, mealType: mealType, name: name, recipeID: recipeID))
        for weekKey in mealWeekKeys(date) {
            await updateSnapshot(key: "meals-\(weekKey)", freshness: freshness.meals) { (meals: inout [MealPlan]) in
                meals.removeAll { $0.date == date && $0.mealType == mealType }
                meals.append(meal)
            }
        }
        if let mealDate = APIDate.parse("\(date)T00:00:00Z") { await invalidateDashboardAndCalendar(on: mealDate) }
        return meal
    }

    public func deleteMeal(date: String, mealType: String) async throws {
        try await delete("api/meals", queryItems: [
            URLQueryItem(name: "date", value: date),
            URLQueryItem(name: "mealType", value: mealType)
        ])
        for weekKey in mealWeekKeys(date) {
            await updateSnapshot(key: "meals-\(weekKey)", freshness: freshness.meals) { (meals: inout [MealPlan]) in
                meals.removeAll { $0.date == date && $0.mealType == mealType }
            }
        }
        if let mealDate = APIDate.parse("\(date)T00:00:00Z") { await invalidateDashboardAndCalendar(on: mealDate) }
    }

    public func fetchMe() async throws -> User {
        try await fetchMe(forceRefresh: false)
    }

    public func fetchMe(forceRefresh: Bool) async throws -> User {
        try await cached(key: "current-user", freshness: freshness.users, forceRefresh: forceRefresh) { try await self.get("api/me") }
    }

    public func fetchCommuteTrains(from origin: String, to destination: String, count: Int) async throws -> CommuteBoard {
        let origin = CommutePreferences.normalizedStationCode(origin)
        let destination = CommutePreferences.normalizedStationCode(destination)
        guard CommutePreferences.isValidStationCode(origin), CommutePreferences.isValidStationCode(destination), (1...10).contains(count) else {
            throw APIError.badRequest(serverMessage: "Enter valid station codes and a train count from 1 to 10.")
        }
        return try await get("api/commute/trains/from/\(origin)/to/\(destination)", queryItems: [
            URLQueryItem(name: "count", value: String(count)),
        ])
    }

    // MARK: - Avatar

    public func uploadAvatar(imageData: Data, mimeType: String) async throws -> User {
        let user: User = try await uploadMultipart("api/profile/avatar", field: "avatar", data: imageData, mimeType: mimeType)
        let namespace = await cacheNamespace()
        if let namespace {
            let key = "avatar-\(user.id)"
            await diskImageCache.remove(namespace: namespace, key: key)
        }
        imageCache.removeObject(forKey: "\(namespace ?? "")-avatar-\(user.id)" as NSString)
        await writeSnapshot(user, key: "current-user", freshness: freshness.users)
        await updateSnapshot(key: "users", freshness: freshness.users) { (users: inout [User]) in
            if let index = users.firstIndex(where: { $0.id == user.id }) { users[index] = user }
        }
        return user
    }

    public func deleteAvatar() async throws {
        try await delete("api/profile/avatar")
        if let namespace = await cacheNamespace() {
            imageCache.removeAllObjects()
            await diskImageCache.remove(namespace: namespace)
        }
        await removeSnapshot(key: "current-user")
        await removeSnapshot(key: "users")
    }

    // MARK: - Settings

    public func fetchSettings() async throws -> AppSettings {
        try await cached(key: "settings", freshness: 0, forceRefresh: true) {
            try await self.get("api/settings")
        }
    }

    public func updateFamilyName(_ name: String) async throws {
        struct Body: Encodable { let family_name: String }
        try await patch("api/settings", body: Body(family_name: name))
        await updateSnapshot(key: "settings", freshness: 0) { (settings: inout AppSettings) in
            settings = AppSettings(familyName: name, mealWeekStart: settings.mealWeekStart)
        }
    }

    public func updateMealWeekStart(_ weekday: MealWeekday) async throws {
        struct Body: Encodable { let meal_week_start: MealWeekday }
        try await patch("api/settings", body: Body(meal_week_start: weekday))
        await updateSnapshot(key: "settings", freshness: 0) { (settings: inout AppSettings) in
            settings = AppSettings(familyName: settings.familyName, mealWeekStart: weekday)
        }
    }

    // MARK: - User management

    public func promoteUser(id: String) async throws -> User {
        let user: User = try await post("api/users/\(id)/promote")
        await updateSnapshot(key: "users", freshness: freshness.users) { (users: inout [User]) in
            if let index = users.firstIndex(where: { $0.id == id }) { users[index] = user }
        }
        return user
    }

    public func demoteUser(id: String) async throws -> User {
        let user: User = try await post("api/users/\(id)/demote")
        await updateSnapshot(key: "users", freshness: freshness.users) { (users: inout [User]) in
            if let index = users.firstIndex(where: { $0.id == id }) { users[index] = user }
        }
        return user
    }

    // MARK: - Categories

    public func fetchCategories() async throws -> [ChoreCategory] {
        try await fetchCategories(forceRefresh: false)
    }

    public func fetchCategories(forceRefresh: Bool) async throws -> [ChoreCategory] {
        try await cached(key: "categories", freshness: freshness.categories, forceRefresh: forceRefresh) { try await self.get("api/categories") }
    }

    public func createCategory(name: String) async throws -> ChoreCategory {
        struct Body: Encodable { let name: String }
        let category: ChoreCategory = try await post("api/categories", body: Body(name: name))
        await updateSnapshot(key: "categories", freshness: freshness.categories) { (categories: inout [ChoreCategory]) in
            categories.removeAll { $0.id == category.id }
            categories.append(category)
        }
        return category
    }

    public func updateCategory(id: String, name: String) async throws -> ChoreCategory {
        struct Body: Encodable { let name: String }
        let category: ChoreCategory = try await put("api/categories/\(id)", body: Body(name: name))
        await updateSnapshot(key: "categories", freshness: freshness.categories) { (categories: inout [ChoreCategory]) in
            if let index = categories.firstIndex(where: { $0.id == id }) { categories[index] = category }
        }
        return category
    }

    public func deleteCategory(id: String) async throws {
        try await delete("api/categories/\(id)")
        await updateSnapshot(key: "categories", freshness: freshness.categories) { (categories: inout [ChoreCategory]) in
            categories.removeAll { $0.id == id }
        }
    }

    // MARK: - API tokens

    public func fetchTokens() async throws -> [APIToken] {
        try await get("api/tokens", decoder: Self.isoDecoder)
    }

    public func createToken(name: String) async throws -> CreatedToken {
        struct Body: Encodable { let name: String }
        return try await post("api/tokens", body: Body(name: name))
    }

    public func deleteToken(id: String) async throws {
        try await delete("api/tokens/\(id)")
    }

    // MARK: - Inventory

    public func fetchInventory() async throws -> [InventoryArea] {
        try await get("api/inventory")
    }

    public func createArea(_ request: AreaRequest) async throws -> InventoryArea {
        try await post("api/inventory/areas", body: request)
    }

    public func updateArea(id: String, _ request: AreaRequest) async throws -> InventoryArea {
        try await put("api/inventory/areas/\(id)", body: request)
    }

    public func deleteArea(id: String) async throws {
        try await delete("api/inventory/areas/\(id)")
    }

    public func createItem(areaID: String, _ request: ItemRequest) async throws -> InventoryItem {
        try await post("api/inventory/areas/\(areaID)/items", body: request)
    }

    public func updateItem(id: String, _ request: ItemRequest) async throws -> InventoryItem {
        try await put("api/inventory/items/\(id)", body: request)
    }

    public func changeStock(id: String, change: InventoryStockChange) async throws -> InventoryItem {
        try await send(path: "api/inventory/items/\(id)/stock", method: "PATCH", body: change)
    }

    public func deleteItem(id: String) async throws {
        try await delete("api/inventory/items/\(id)")
    }
}

extension APIClient {
    public var serverIdentifier: String? { baseURL.absoluteString }
    public func sendFamilyBroadcast(message: String, requestID: String) async throws -> BroadcastResult {
        struct Request: Encodable { let message: String; let requestID: String }
        return try await post("api/notifications/broadcast", body: Request(message: message, requestID: requestID))
    }

    public func fetchChoreReminders() async throws -> ChoreReminders {
        try await get("api/chore-reminders")
    }

    public func updateChoreReminders(_ settings: ChoreReminders) async throws {
        struct Request: Encodable { let enabled: Bool; let times: [String] }
        try await patch("api/chore-reminders", body: Request(enabled: settings.enabled, times: settings.times))
    }

    public func registerNotificationDevice(_ device: NotificationDevice) async throws {
        try await sendVoid(path: "api/notification-devices/\(device.id)", method: "PUT", body: device)
    }

    public func removeNotificationDevice(id: String) async throws {
        try await delete("api/notification-devices/\(id)")
    }

    public func fetchHomeInfo() async throws -> HomeInfo { try await get("api/home-info") }
    public func updateHomeInfo(_ home: HomeInfo) async throws { try await patch("api/home-info", body: home) }
    public func fetchWeather() async throws -> HouseholdWeather { try await get("api/weather") }
    public func fetchDepartures() async throws -> HouseholdDepartures { try await get("api/departures") }
    public func fetchShopping() async throws -> [ShoppingItem] { try await get("api/shopping") }
    public func addShopping(name: String, inventoryItemID: String?) async throws -> ShoppingItem {
        struct Body: Encodable { let name: String; let inventoryItemID: String? }
        return try await post("api/shopping/items", body: Body(name: name, inventoryItemID: inventoryItemID))
    }
    public func updateShopping(id: String, change: ShoppingChange) async throws -> ShoppingItem {
        try await send(path: "api/shopping/items/\(id)", method: "PATCH", body: change)
    }
    public func deleteShopping(id: String) async throws { try await delete("api/shopping/items/\(id)") }
    public func clearPurchasedShopping() async throws { try await delete("api/shopping/purchased") }
    public func restockShopping(id: String, request: ShoppingRestock) async throws -> ShoppingItem {
        try await post("api/shopping/items/\(id)/restock", body: request)
    }
}

extension APIClient {
    public func fetchChoreDefinitions(includeInactive: Bool) async throws -> [ChoreDefinition] {
        try await fetchChoreDefinitions(includeInactive: includeInactive, forceRefresh: false)
    }

    public func fetchChoreDefinitions(includeInactive: Bool, forceRefresh: Bool) async throws -> [ChoreDefinition] {
        let query = [URLQueryItem(name: "include_inactive", value: String(includeInactive))]
        return try await cached(key: "chore-definitions-\(includeInactive)", freshness: freshness.chores, forceRefresh: forceRefresh) {
            try await self.get("api/chore-definitions", queryItems: query)
        }
    }

    public func createChoreDefinition(_ request: ChoreRequest) async throws -> ChoreDefinition {
        let definition: ChoreDefinition = try await post("api/chore-definitions", body: request)
        await updateChoreDefinitionSnapshots(definition, id: definition.id)
        await invalidateDashboardAndCalendar()
        return definition
    }

    public func updateChoreDefinition(id: String, _ request: ChoreRequest) async throws -> ChoreDefinition {
        let definition: ChoreDefinition = try await put("api/chore-definitions/\(id)", body: request)
        await updateChoreDefinitionSnapshots(definition, id: id)
        await invalidateDashboardAndCalendar()
        return definition
    }

    public func deleteChoreDefinition(id: String) async throws {
        try await delete("api/chore-definitions/\(id)")
        for includeInactive in [false, true] {
            await updateSnapshot(key: "chore-definitions-\(includeInactive)", freshness: freshness.chores) { (definitions: inout [ChoreDefinition]) in
                definitions.removeAll { $0.id == id }
            }
        }
        await invalidateDashboardAndCalendar()
    }

    public func fetchChoreHistory(definitionID: String, cursor: String?) async throws -> ChoreHistoryPage {
        var components = URLComponents()
        components.queryItems = [URLQueryItem(name: "definition_id", value: definitionID)]
        if let cursor { components.queryItems?.append(URLQueryItem(name: "cursor", value: cursor)) }
        return try await get("api/chore-history", queryItems: components.queryItems ?? [])
    }

    public func skipChore(id: String) async throws {
        try await post("api/chores/\(id)/skip")
        await removeSnapshot(key: "chores")
        await invalidateDashboardAndCalendar()
    }

    private func updateChoreDefinitionSnapshots(_ definition: ChoreDefinition, id: String) async {
        for includeInactive in [false, true] {
            await updateSnapshot(key: "chore-definitions-\(includeInactive)", freshness: freshness.chores) { (definitions: inout [ChoreDefinition]) in
                definitions.removeAll { $0.id == id }
                if includeInactive || definition.active { definitions.append(definition) }
            }
        }
    }
}
