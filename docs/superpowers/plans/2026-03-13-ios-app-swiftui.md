# iOS App (SwiftUI) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a native SwiftUI iOS app for Family Hub backed by the Go REST API, with OIDC/PKCE authentication via Authentik.

**Architecture:** Feature-by-feature delivery. Each feature has a ViewModel (tested against a FakeAPIClient) and a SwiftUI View. ViewModels are `@Observable` classes on `@MainActor`; the live `APIClient` uses `async/await` + `URLSession`. Auth is handled by `AuthManager` which gates the entire app.

**Tech Stack:** Swift 5.9+, SwiftUI (iOS 17+), `@Observable`, `ASWebAuthenticationSession` (OIDC/PKCE), `URLSession`, Keychain Services (`SecItem`), XCTest.

**Note:** Tasks 1–5 (backend API endpoints) are already complete on branch `feature/ios-app` (worktree at `.worktrees/feature/ios-app`). All iOS work in this plan must be done on that same branch — **not** on `main`. The extended `/api/dashboard` endpoint (with `chores_due_today_list` / `chores_overdue_list`), `/api/chores/{id}/complete`, `/api/meals`, `/api/recipes`, and `/api/calendar` all live in that branch's `internal/handlers/api.go`.

---

## File Map

### iOS (`ios/FamilyHub/`)

| File | Responsibility |
|------|----------------|
| `FamilyHubApp.swift` | `@main` App struct; auth-gate routing |
| `Auth/AuthManager.swift` | OIDC/PKCE flow, token refresh, `isAuthenticated` |
| `Auth/KeychainStore.swift` | Read/write tokens to Keychain via `SecItem` |
| `Auth/OIDCConfig.swift` | Load `ClientID`, endpoints, `BaseURL` from `Config.plist` |
| `Auth/LoginView.swift` | "Sign In" button that triggers `authManager.login()` |
| `Networking/APIError.swift` | Typed error enum: `.network`, `.unauthorized`, `.notFound`, `.conflict`, `.decoding`, `.server(Int)` |
| `Networking/APIClientProtocol.swift` | Protocol with `async throws` methods; `ViewState<T>` enum |
| `Networking/APIClient.swift` | Live implementation using `URLSession` |
| `Models/Chore.swift` | `Chore`, `ChoreStatus` Codable structs |
| `Models/MealPlan.swift` | `MealPlan` Codable struct |
| `Models/Recipe.swift` | `Recipe`, `IngredientGroup` Codable structs |
| `Models/CalendarData.swift` | `CalendarResponse` Codable struct wrapping `[Chore]` |
| `Models/DashboardStats.swift` | `DashboardStats` Codable struct |
| `Features/Dashboard/DashboardViewModel.swift` | Loads stats; `@Observable @MainActor` |
| `Features/Dashboard/DashboardView.swift` | Two stat cards + chore list |
| `Features/Chores/ChoresViewModel.swift` | Loads + filters chores; calls complete |
| `Features/Chores/ChoresView.swift` | Grouped list: Pending / Completed |
| `Features/Chores/ChoreDetailView.swift` | Chore detail + Mark Complete button |
| `Features/Meals/MealsViewModel.swift` | Loads week of meal plans |
| `Features/Meals/MealsView.swift` | Weekly table view, week navigation |
| `Features/Recipes/RecipesViewModel.swift` | Loads recipe list |
| `Features/Recipes/RecipesView.swift` | Two-column grid |
| `Features/Recipes/RecipeDetailView.swift` | Ingredients + steps + cook mode |
| `Features/Calendar/CalendarViewModel.swift` | Loads chores for a month |
| `Features/Calendar/CalendarView.swift` | Monthly grid + agenda list |

### Tests (`ios/FamilyHubTests/`)

| File | Responsibility |
|------|----------------|
| `FakeAPIClient.swift` | Test fake conforming to `APIClientProtocol` |
| `Auth/KeychainStoreTests.swift` | Keychain read/write/clear |
| `Dashboard/DashboardViewModelTests.swift` | ViewModel unit tests |
| `Chores/ChoresViewModelTests.swift` | ViewModel unit tests |
| `Meals/MealsViewModelTests.swift` | ViewModel unit tests |
| `Recipes/RecipesViewModelTests.swift` | ViewModel unit tests |
| `Calendar/CalendarViewModelTests.swift` | ViewModel unit tests |

---

## Chunk 1: Foundation

### Task 6: Create Xcode project (MANUAL)

This task is done in Xcode — it cannot be scripted.

**Files:**
- Create: `ios/FamilyHub.xcodeproj/`
- Create: `ios/FamilyHub/` (source root)
- Create: `ios/FamilyHubTests/` (test target)
- Modify: `.gitignore`

- [ ] **Step 1: Create the Xcode project**

  1. Open Xcode → File → New → Project → iOS → App
  2. Product Name: `FamilyHub`
  3. Team: your Apple ID
  4. Organization Identifier: `uk.co.suskins`  → Bundle ID becomes `uk.co.suskins.familyhub`
  5. Interface: **SwiftUI**
  6. Language: **Swift**
  7. Uncheck "Use Core Data" and "Include Tests" (we add tests manually next)
  8. Save to `ios/` inside the repo root

- [ ] **Step 2: Add unit test target**

  File → New → Target → Unit Testing Bundle
  - Product Name: `FamilyHubTests`
  - Target to be tested: `FamilyHub`

- [ ] **Step 3: Register `familyhub://` URL scheme**

  Select `FamilyHub` target → Info tab → URL Types → `+`
  - Identifier: `uk.co.suskins.familyhub`
  - URL Schemes: `familyhub`

- [ ] **Step 4: Create folder structure in Finder**

  Inside `ios/FamilyHub/` create these empty directories:
  ```
  Auth/
  Networking/
  Models/
  Features/Dashboard/
  Features/Chores/
  Features/Meals/
  Features/Recipes/
  Features/Calendar/
  ```

  In Xcode, drag each folder into the Project Navigator to add as groups (choose "Create groups", not references).

  In `ios/FamilyHubTests/` create:
  ```
  Auth/
  Dashboard/
  Chores/
  Meals/
  Recipes/
  Calendar/
  ```

- [ ] **Step 5: Delete generated boilerplate**

  Delete the `ContentView.swift` file Xcode generated — we'll create our own in Task 18.

  After deleting it, open `FamilyHubApp.swift` and replace the generated `ContentView()` reference with a temporary placeholder so the project still compiles:

  ```swift
  // Temporary placeholder — replaced in Task 11
  @main
  struct FamilyHubApp: App {
      var body: some Scene {
          WindowGroup {
              Text("Loading…")
          }
      }
  }

  // Also remove the Xcode-generated `struct ContentView: View` if it's still referenced
  // in FamilyHubApp.swift — it was deleted from disk in Step 5.
  ```

  Keep `Assets.xcassets` (required by the build system). Delete only the placeholder color set entries inside it (AccentColor, etc.) if desired — the catalog itself must remain.

  Keep: `FamilyHubApp.swift` (we'll replace its contents in Task 11).

- [ ] **Step 6: Add `.gitignore` entries**

  Append to the root `.gitignore`:
  ```
  # iOS build artefacts
  ios/build/
  ios/DerivedData/
  ios/*.xcodeproj/xcuserdata/
  ios/*.xcodeproj/project.xcworkspace/xcuserdata/
  ios/*.xcworkspace/xcuserdata/
  ios/Config.plist
  ```

- [ ] **Step 7: Create `Config.example.plist`**

  Create `ios/FamilyHub/Config.example.plist`:
  ```xml
  <?xml version="1.0" encoding="UTF-8"?>
  <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
  <plist version="1.0">
  <dict>
      <key>ClientID</key>
      <string>YOUR_AUTHENTIK_CLIENT_ID</string>
      <key>AuthorizationEndpoint</key>
      <string>https://authentik.example.com/application/o/familyhub/authorize/</string>
      <key>TokenEndpoint</key>
      <string>https://authentik.example.com/application/o/token/</string>
      <key>BaseURL</key>
      <string>http://192.168.1.x:8080</string>
  </dict>
  </plist>
  ```

  Copy to `ios/FamilyHub/Config.plist` and fill in your real values. `Config.plist` is gitignored; `Config.example.plist` is committed.

- [ ] **Step 8: Commit**

  ```bash
  git add ios/ .gitignore
  git commit -m "feat: add Xcode project skeleton with SwiftUI + familyhub:// URL scheme"
  ```

---

### Task 7: KeychainStore

**Files:**
- Create: `ios/FamilyHub/Auth/KeychainStore.swift`
- Create: `ios/FamilyHubTests/Auth/KeychainStoreTests.swift`

- [ ] **Step 1: Write the failing test**

  Create `ios/FamilyHubTests/Auth/KeychainStoreTests.swift`:

  ```swift
  import XCTest
  @testable import FamilyHub

  final class KeychainStoreTests: XCTestCase {
      var store: KeychainStore!

      override func setUp() {
          super.setUp()
          // Unique service per test run to avoid pollution
          store = KeychainStore(service: "uk.co.suskins.familyhub.tests.\(UUID().uuidString)")
      }

      override func tearDown() {
          store.clear()
          super.tearDown()
      }

      func testSaveAndReadTokens() {
          store.save(accessToken: "access123", refreshToken: "refresh456")
          XCTAssertEqual(store.accessToken, "access123")
          XCTAssertEqual(store.refreshToken, "refresh456")
      }

      func testClearRemovesTokens() {
          store.save(accessToken: "access123", refreshToken: "refresh456")
          store.clear()
          XCTAssertNil(store.accessToken)
          XCTAssertNil(store.refreshToken)
      }

      func testReadBeforeSaveReturnsNil() {
          XCTAssertNil(store.accessToken)
          XCTAssertNil(store.refreshToken)
      }

      func testOverwriteToken() {
          store.save(accessToken: "old", refreshToken: "old-refresh")
          store.save(accessToken: "new", refreshToken: "new-refresh")
          XCTAssertEqual(store.accessToken, "new")
      }
  }
  ```

- [ ] **Step 2: Run to confirm failure**

  In Xcode: Product → Test (⌘U). Expected: compile error — `KeychainStore` not found.

- [ ] **Step 3: Implement `KeychainStore`**

  Create `ios/FamilyHub/Auth/KeychainStore.swift`:

  ```swift
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
  ```

- [ ] **Step 4: Run tests (⌘U)**

  Expected: 4 tests pass.

- [ ] **Step 5: Commit**

  ```bash
  git add ios/FamilyHub/Auth/KeychainStore.swift ios/FamilyHubTests/Auth/KeychainStoreTests.swift
  git commit -m "feat: add KeychainStore for token persistence"
  ```

---

### Task 8: APIError, ViewState, and APIClientProtocol

**Files:**
- Create: `ios/FamilyHub/Networking/APIError.swift`
- Create: `ios/FamilyHub/Networking/APIClientProtocol.swift`

No tests for these — they are protocols and enums with no logic. Tests come via ViewModel tests in later tasks.

- [ ] **Step 1: Create `APIError.swift`**

  ```swift
  // ios/FamilyHub/Networking/APIError.swift
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
  ```

- [ ] **Step 2: Create `APIClientProtocol.swift`**

  ```swift
  // ios/FamilyHub/Networking/APIClientProtocol.swift
  import Foundation

  enum ViewState<T> {
      case idle
      case loading
      case loaded(T)
      case failed(APIError)
  }

  protocol APIClientProtocol: AnyObject {
      func fetchDashboardStats() async throws -> DashboardStats
      func fetchChores() async throws -> [Chore]
      func completeChore(id: String) async throws
      func fetchMeals(week: Date) async throws -> [MealPlan]
      func fetchRecipes() async throws -> [Recipe]
      func fetchRecipe(id: String) async throws -> Recipe
      func fetchCalendar(month: Date) async throws -> [Chore]
  }
  ```

- [ ] **Step 3: Commit**

  ```bash
  git add ios/FamilyHub/Networking/APIError.swift ios/FamilyHub/Networking/APIClientProtocol.swift
  git commit -m "feat: add APIError, ViewState, and APIClientProtocol"
  ```

  **Note:** The project will not compile until the model types (`DashboardStats`, `Chore`, etc.) are added in Task 9. The build verification step is at the end of Task 9.

---

### Task 9: Codable models

**Files:**
- Create: `ios/FamilyHub/Models/Chore.swift`
- Create: `ios/FamilyHub/Models/MealPlan.swift`
- Create: `ios/FamilyHub/Models/Recipe.swift`
- Create: `ios/FamilyHub/Models/CalendarData.swift`
- Create: `ios/FamilyHub/Models/DashboardStats.swift`

No unit tests — these are data containers. Correctness is validated by ViewModel integration tests (Tasks 12–17) and end-to-end JSON decoding.

**JSON key mapping note:** The Go backend uses default `encoding/json` marshaling, so struct fields serialise as their Go names (PascalCase: `"ID"`, `"Name"`, `"Status"`, `"DueDate"`, etc.) **except** `IngredientGroup` which has explicit tags (`"name"`, `"items"`), and `DashboardStats` which uses a manual `map` with snake_case keys.

- [ ] **Step 1: Create `Chore.swift`**

  ```swift
  // ios/FamilyHub/Models/Chore.swift
  import Foundation

  enum ChoreStatus: String, Codable {
      case pending = "pending"
      case completed = "completed"
      case overdue = "overdue"
  }

  struct Chore: Codable, Identifiable {
      let id: String
      let name: String
      let description: String
      let status: ChoreStatus
      let dueDate: String?       // RFC3339 timestamp or nil
      let assignedToUserID: String?

      enum CodingKeys: String, CodingKey {
          case id = "ID"
          case name = "Name"
          case description = "Description"
          case status = "Status"
          case dueDate = "DueDate"
          case assignedToUserID = "AssignedToUserID"
      }
  }
  ```

- [ ] **Step 2: Create `MealPlan.swift`**

  `MealPlan` has no `ID` field in the backend — use a computed `id` for `Identifiable`.

  ```swift
  // ios/FamilyHub/Models/MealPlan.swift
  import Foundation

  struct MealPlan: Codable, Identifiable {
      let date: String     // "YYYY-MM-DD"
      let mealType: String // "breakfast" | "lunch" | "dinner"
      let name: String
      let notes: String
      let recipeID: String?

      var id: String { "\(date)-\(mealType)" }

      enum CodingKeys: String, CodingKey {
          case date = "Date"
          case mealType = "MealType"
          case name = "Name"
          case notes = "Notes"
          case recipeID = "RecipeID"
      }
  }
  ```

- [ ] **Step 3: Create `Recipe.swift`**

  ```swift
  // ios/FamilyHub/Models/Recipe.swift
  import Foundation

  struct IngredientGroup: Codable {
      let name: String
      let items: [String]
  }

  struct Recipe: Codable, Identifiable {
      let id: String
      let title: String
      let steps: [String]?           // Go nil slice marshals as null
      let ingredients: [IngredientGroup]?  // Go nil slice marshals as null
      let servings: Int?
      let prepTime: String?
      let cookTime: String?
      let hasImage: Bool

      enum CodingKeys: String, CodingKey {
          case id = "ID"
          case title = "Title"
          case steps = "Steps"
          case ingredients = "Ingredients"
          case servings = "Servings"
          case prepTime = "PrepTime"
          case cookTime = "CookTime"
          case hasImage = "HasImage"
      }
  }
  ```

- [ ] **Step 4: Create `CalendarData.swift`**

  The calendar endpoint returns `{"chores": [...]}`.

  ```swift
  // ios/FamilyHub/Models/CalendarData.swift
  import Foundation

  struct CalendarResponse: Decodable {
      let chores: [Chore]   // backend guarantees [] not null (nil guard in handler)
  }
  ```

- [ ] **Step 5: Create `DashboardStats.swift`**

  The dashboard endpoint uses snake_case manual JSON keys. **Task 1 (already complete) extended the backend to include `chores_due_today_list` and `chores_overdue_list` — these fields exist in the live API response.**

  ```swift
  // ios/FamilyHub/Models/DashboardStats.swift
  import Foundation

  struct DashboardStats: Decodable {
      let choresDueToday: Int
      let choresOverdue: Int
      let choresDueTodayList: [Chore]
      let choresOverdueList: [Chore]

      enum CodingKeys: String, CodingKey {
          case choresDueToday = "chores_due_today"
          case choresOverdue = "chores_overdue"
          case choresDueTodayList = "chores_due_today_list"
          case choresOverdueList = "chores_overdue_list"
      }
  }
  ```

- [ ] **Step 6: Build (⌘B)**

  This is the first successful build checkpoint — Tasks 8 and 9 together now provide the types `APIClientProtocol` depends on. Expected: success with no errors.

- [ ] **Step 7: Commit**

  ```bash
  git add ios/FamilyHub/Models/
  git commit -m "feat: add Codable models (Chore, MealPlan, Recipe, CalendarData, DashboardStats)"
  ```

---

## Chunk 2: Networking + Auth

### Task 10: Live APIClient

**Files:**
- Create: `ios/FamilyHub/Networking/APIClient.swift`

Testing: `APIClient` is tested end-to-end manually (point at real backend). ViewModel tests use `FakeAPIClient` (Task 11). No unit tests for `APIClient` itself in v1.

- [ ] **Step 1: Create `OIDCConfig.swift`**

  Create `ios/FamilyHub/Auth/OIDCConfig.swift`:

  ```swift
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
  ```

- [ ] **Step 2: Implement `APIClient.swift`**

  ```swift
  // ios/FamilyHub/Networking/APIClient.swift
  import Foundation

  final class APIClient: APIClientProtocol {
      private let baseURL: URL
      private let session: URLSession
      // Weak to avoid retain cycle: AuthManager → APIClient → AuthManager
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
          try validate(response: response, data: Data())
      }

      private func buildRequest(path: String, method: String, queryItems: [URLQueryItem] = []) async throws -> URLRequest {
          guard let authManager else { throw APIError.unauthorized }
          let token = try await authManager.validAccessToken()

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
          try validate(response: response, data: data)
          do {
              return try JSONDecoder().decode(type, from: data)
          } catch {
              throw APIError.decoding(error)
          }
      }

      @discardableResult
      private func validate(response: HTTPURLResponse, data: Data) throws -> Data {
          switch response.statusCode {
          case 200...299: return data
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
  }
  ```

- [ ] **Step 3: Build (⌘B)**

  Expected: success.

- [ ] **Step 4: Commit**

  ```bash
  git add ios/FamilyHub/Auth/OIDCConfig.swift ios/FamilyHub/Networking/APIClient.swift
  git commit -m "feat: add OIDCConfig and live APIClient"
  ```

---

### Task 11: FakeAPIClient, AuthManager, LoginView, FamilyHubApp

**Files:**
- Create: `ios/FamilyHubTests/FakeAPIClient.swift`
- Create: `ios/FamilyHub/Auth/AuthManager.swift`
- Create: `ios/FamilyHub/Auth/LoginView.swift`
- Modify: `ios/FamilyHub/FamilyHubApp.swift`

- [ ] **Step 1: Create `FakeAPIClient.swift`**

  ```swift
  // ios/FamilyHubTests/FakeAPIClient.swift
  import Foundation
  @testable import FamilyHub

  final class FakeAPIClient: APIClientProtocol {
      var dashboardResult: Result<DashboardStats, Error> = .success(
          DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [])
      )
      var choresResult: Result<[Chore], Error> = .success([])
      var completeChoreResult: Result<Void, Error> = .success(())
      var mealsResult: Result<[MealPlan], Error> = .success([])
      var recipesResult: Result<[Recipe], Error> = .success([])
      var recipeResult: Result<Recipe, Error> = .failure(APIError.notFound)
      var calendarResult: Result<[Chore], Error> = .success([])

      func fetchDashboardStats() async throws -> DashboardStats { try dashboardResult.get() }
      func fetchChores() async throws -> [Chore] { try choresResult.get() }
      func completeChore(id: String) async throws { try completeChoreResult.get() }
      func fetchMeals(week: Date) async throws -> [MealPlan] { try mealsResult.get() }
      func fetchRecipes() async throws -> [Recipe] { try recipesResult.get() }
      func fetchRecipe(id: String) async throws -> Recipe { try recipeResult.get() }
      func fetchCalendar(month: Date) async throws -> [Chore] { try calendarResult.get() }
  }
  ```

- [ ] **Step 2: Implement `AuthManager.swift`**

  ```swift
  // ios/FamilyHub/Auth/AuthManager.swift
  import CryptoKit
  import Foundation
  import AuthenticationServices

  @Observable
  @MainActor
  final class AuthManager: NSObject {
      private(set) var isAuthenticated = false

      private let keychain: KeychainStore
      let config: OIDCConfig

      init(keychain: KeychainStore = .shared, config: OIDCConfig = .fromPlist()) {
          self.keychain = keychain
          self.config = config
          super.init()
          self.isAuthenticated = keychain.accessToken != nil
      }

      // MARK: - Login (OIDC/PKCE)

      func login() async throws {
          let (codeVerifier, codeChallenge) = generatePKCE()
          let state = UUID().uuidString

          var components = URLComponents(url: config.authorizationEndpoint, resolvingAgainstBaseURL: false)!
          components.queryItems = [
              URLQueryItem(name: "response_type", value: "code"),
              URLQueryItem(name: "client_id",     value: config.clientID),
              URLQueryItem(name: "redirect_uri",  value: "familyhub://auth/callback"),
              URLQueryItem(name: "scope",         value: "openid profile email offline_access"),
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
              let components = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false),
              let code = components.queryItems?.first(where: { $0.name == "code" })?.value
          else {
              throw AuthError.invalidCallback
          }

          try await exchangeCode(code, codeVerifier: codeVerifier)
      }

      // MARK: - Token management

      func validAccessToken() async throws -> String {
          guard let token = keychain.accessToken else {
              isAuthenticated = false
              throw APIError.unauthorized
          }
          return token
          // TODO: check expiry and refresh — implement in v2
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
              URLQueryItem(name: "redirect_uri",  value: "familyhub://auth/callback"),
              URLQueryItem(name: "code_verifier", value: codeVerifier),
          ]
          request.httpBody = params.query?.data(using: .utf8)

          let (data, _) = try await URLSession.shared.data(for: request)
          let response = try JSONDecoder().decode(TokenResponse.self, from: data)
          keychain.save(accessToken: response.accessToken, refreshToken: response.refreshToken)
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
      let refreshToken: String

      enum CodingKeys: String, CodingKey {
          case accessToken = "access_token"
          case refreshToken = "refresh_token"
      }
  }
  ```

  `CryptoKit` is already imported at the top of the file.

- [ ] **Step 3: Create `LoginView.swift`**

  ```swift
  // ios/FamilyHub/Auth/LoginView.swift
  import SwiftUI

  struct LoginView: View {
      @Environment(AuthManager.self) private var authManager
      @State private var isLoading = false
      @State private var errorMessage: String?

      var body: some View {
          VStack(spacing: 24) {
              Spacer()

              Text("Family Hub")
                  .font(.largeTitle.bold())

              Text("Manage chores, meals, and more.")
                  .foregroundStyle(.secondary)

              Spacer()

              if let errorMessage {
                  Text(errorMessage)
                      .foregroundStyle(.red)
                      .font(.caption)
              }

              Button {
                  Task {
                      isLoading = true
                      errorMessage = nil
                      do {
                          try await authManager.login()
                      } catch {
                          errorMessage = error.localizedDescription
                      }
                      isLoading = false
                  }
              } label: {
                  if isLoading {
                      ProgressView()
                          .frame(maxWidth: .infinity)
                  } else {
                      Text("Sign In")
                          .frame(maxWidth: .infinity)
                  }
              }
              .buttonStyle(.borderedProminent)
              .padding(.horizontal)
              .disabled(isLoading)
          }
          .padding()
      }
  }
  ```

- [ ] **Step 4: Replace `FamilyHubApp.swift`**

  Replace the contents of `ios/FamilyHub/FamilyHubApp.swift`:

  ```swift
  import SwiftUI

  @main
  struct FamilyHubApp: App {
      @State private var authManager = AuthManager()

      var body: some Scene {
          WindowGroup {
              if authManager.isAuthenticated {
                  let config = authManager.config
                  let client = APIClient(baseURL: config.baseURL, authManager: authManager)
                  ContentView(apiClient: client)
                      .environment(authManager)
              } else {
                  LoginView()
                      .environment(authManager)
              }
          }
      }
  }
  ```

  `ContentView` will be created in Task 18. For now, add a temporary placeholder so it compiles:

  ```swift
  // Temporary — replaced in Task 18
  struct ContentView: View {
      let apiClient: any APIClientProtocol
      var body: some View { Text("Loading...") }
  }
  ```

- [ ] **Step 5: Build (⌘B)**

  Expected: success.

- [ ] **Step 6: Commit**

  ```bash
  git add ios/FamilyHubTests/FakeAPIClient.swift \
          ios/FamilyHub/Auth/AuthManager.swift \
          ios/FamilyHub/Auth/LoginView.swift \
          ios/FamilyHub/FamilyHubApp.swift
  git commit -m "feat: add FakeAPIClient, AuthManager, LoginView, and FamilyHubApp entry point"
  ```

---

## Chunk 3: Dashboard + Chores

### Task 12: DashboardViewModel + DashboardView

**Files:**
- Create: `ios/FamilyHub/Features/Dashboard/DashboardViewModel.swift`
- Create: `ios/FamilyHub/Features/Dashboard/DashboardView.swift`
- Create: `ios/FamilyHubTests/Dashboard/DashboardViewModelTests.swift`

- [ ] **Step 1: Write the failing tests**

  Create `ios/FamilyHubTests/Dashboard/DashboardViewModelTests.swift`:

  ```swift
  import XCTest
  @testable import FamilyHub

  @MainActor
  final class DashboardViewModelTests: XCTestCase {
      func testLoadSuccess() async {
          let fake = FakeAPIClient()
          fake.dashboardResult = .success(DashboardStats(
              choresDueToday: 2,
              choresOverdue: 1,
              choresDueTodayList: [],
              choresOverdueList: []
          ))
          let viewModel = DashboardViewModel(apiClient: fake)

          await viewModel.load()

          guard case .loaded(let stats) = viewModel.state else {
              XCTFail("expected loaded state, got \(viewModel.state)")
              return
          }
          XCTAssertEqual(stats.choresDueToday, 2)
          XCTAssertEqual(stats.choresOverdue, 1)
      }

      func testLoadFailure() async {
          let fake = FakeAPIClient()
          fake.dashboardResult = .failure(APIError.server(500))
          let viewModel = DashboardViewModel(apiClient: fake)

          await viewModel.load()

          guard case .failed(let error) = viewModel.state else {
              XCTFail("expected failed state")
              return
          }
          if case .server(let code) = error {
              XCTAssertEqual(code, 500)
          } else {
              XCTFail("expected server error, got \(error)")
          }
      }

      func testInitialStateIsIdle() {
          let viewModel = DashboardViewModel(apiClient: FakeAPIClient())
          guard case .idle = viewModel.state else {
              XCTFail("expected idle initial state")
              return
          }
      }
  }
  ```

- [ ] **Step 2: Run (⌘U) to confirm failure**

  Expected: compile error — `DashboardViewModel` not found.

- [ ] **Step 3: Implement `DashboardViewModel.swift`**

  ```swift
  // ios/FamilyHub/Features/Dashboard/DashboardViewModel.swift
  import Foundation
  import Observation

  @Observable
  @MainActor
  final class DashboardViewModel {
      var state: ViewState<DashboardStats> = .idle

      private let apiClient: any APIClientProtocol

      init(apiClient: any APIClientProtocol) {
          self.apiClient = apiClient
      }

      func load() async {
          state = .loading
          do {
              let stats = try await apiClient.fetchDashboardStats()
              state = .loaded(stats)
          } catch let error as APIError {
              state = .failed(error)
          } catch {
              state = .failed(.network(error))
          }
      }
  }
  ```

- [ ] **Step 4: Run (⌘U) — confirm tests pass**

- [ ] **Step 5: Implement `DashboardView.swift`**

  ```swift
  // ios/FamilyHub/Features/Dashboard/DashboardView.swift
  import SwiftUI

  struct DashboardView: View {
      @State private var viewModel: DashboardViewModel

      init(apiClient: any APIClientProtocol) {
          _viewModel = State(wrappedValue: DashboardViewModel(apiClient: apiClient))
      }

      var body: some View {
          NavigationStack {
              Group {
                  switch viewModel.state {
                  case .idle, .loading:
                      ProgressView()
                  case .loaded(let stats):
                      dashboardContent(stats)
                  case .failed(let error):
                      ContentUnavailableView(
                          "Failed to load",
                          systemImage: "exclamationmark.triangle",
                          description: Text(error.localizedDescription)
                      )
                  }
              }
              .navigationTitle("Dashboard")
              .toolbar {
                  ToolbarItem(placement: .navigationBarTrailing) {
                      Button("Refresh") {
                          Task { await viewModel.load() }
                      }
                  }
              }
          }
          .task { await viewModel.load() }
      }

      @ViewBuilder
      private func dashboardContent(_ stats: DashboardStats) -> some View {
          List {
              Section {
                  HStack(spacing: 16) {
                      StatCard(title: "Due Today", value: stats.choresDueToday, color: .blue)
                      StatCard(title: "Overdue", value: stats.choresOverdue, color: .red)
                  }
                  .listRowInsets(EdgeInsets())
                  .listRowBackground(Color.clear)
              }

              if !stats.choresDueTodayList.isEmpty {
                  Section("Due Today") {
                      ForEach(stats.choresDueTodayList) { chore in
                          Text(chore.name)
                      }
                  }
              }

              if !stats.choresOverdueList.isEmpty {
                  Section("Overdue") {
                      ForEach(stats.choresOverdueList) { chore in
                          Text(chore.name)
                              .foregroundStyle(.red)
                      }
                  }
              }
          }
      }
  }

  private struct StatCard: View {
      let title: String
      let value: Int
      let color: Color

      var body: some View {
          VStack {
              Text("\(value)")
                  .font(.largeTitle.bold())
                  .foregroundStyle(color)
              Text(title)
                  .font(.caption)
                  .foregroundStyle(.secondary)
          }
          .frame(maxWidth: .infinity)
          .padding()
          .background(color.opacity(0.1))
          .clipShape(RoundedRectangle(cornerRadius: 12))
      }
  }
  ```

- [ ] **Step 6: Build (⌘B)**

  Expected: success.

- [ ] **Step 7: Commit**

  ```bash
  git add ios/FamilyHub/Features/Dashboard/ ios/FamilyHubTests/Dashboard/
  git commit -m "feat: add DashboardViewModel and DashboardView"
  ```

---

### Task 13: ChoresViewModel + ChoresView

**Files:**
- Create: `ios/FamilyHub/Features/Chores/ChoresViewModel.swift`
- Create: `ios/FamilyHub/Features/Chores/ChoresView.swift`
- Create: `ios/FamilyHubTests/Chores/ChoresViewModelTests.swift`

- [ ] **Step 1: Write the failing tests**

  Create `ios/FamilyHubTests/Chores/ChoresViewModelTests.swift`:

  ```swift
  import XCTest
  @testable import FamilyHub

  @MainActor
  final class ChoresViewModelTests: XCTestCase {
      private func makeChore(id: String, status: ChoreStatus) -> Chore {
          Chore(id: id, name: "Chore \(id)", description: "", status: status, dueDate: nil, assignedToUserID: nil)
      }

      func testLoadFiltersChoresByStatus() async {
          let fake = FakeAPIClient()
          fake.choresResult = .success([
              makeChore(id: "1", status: .pending),
              makeChore(id: "2", status: .completed),
              makeChore(id: "3", status: .overdue),
          ])
          let viewModel = ChoresViewModel(apiClient: fake)

          await viewModel.load()

          XCTAssertEqual(viewModel.pendingChores.count, 2) // pending + overdue
          XCTAssertEqual(viewModel.completedChores.count, 1)
      }

      func testCompleteChoreUpdatesLocalState() async {
          let fake = FakeAPIClient()
          fake.choresResult = .success([makeChore(id: "1", status: .pending)])
          fake.completeChoreResult = .success(())
          let viewModel = ChoresViewModel(apiClient: fake)
          await viewModel.load()

          await viewModel.complete(choreID: "1")

          XCTAssertEqual(viewModel.pendingChores.count, 0)
          XCTAssertEqual(viewModel.completedChores.count, 1)
      }

      func testCompleteChoreFailureLeavesStateUnchanged() async {
          let fake = FakeAPIClient()
          fake.choresResult = .success([makeChore(id: "1", status: .pending)])
          fake.completeChoreResult = .failure(APIError.server(500))
          let viewModel = ChoresViewModel(apiClient: fake)
          await viewModel.load()

          await viewModel.complete(choreID: "1")

          XCTAssertEqual(viewModel.pendingChores.count, 1)
          XCTAssertNotNil(viewModel.errorMessage)
      }
  }
  ```

- [ ] **Step 2: Run (⌘U) to confirm failure**

- [ ] **Step 3: Implement `ChoresViewModel.swift`**

  ```swift
  // ios/FamilyHub/Features/Chores/ChoresViewModel.swift
  import Foundation
  import Observation

  @Observable
  @MainActor
  final class ChoresViewModel {
      var state: ViewState<[Chore]> = .idle
      var errorMessage: String?

      // Derived from loaded chores
      var pendingChores: [Chore] {
          guard case .loaded(let chores) = state else { return [] }
          return chores.filter { $0.status == .pending || $0.status == .overdue }
      }

      var completedChores: [Chore] {
          guard case .loaded(let chores) = state else { return [] }
          return chores.filter { $0.status == .completed }
      }

      private let apiClient: any APIClientProtocol

      init(apiClient: any APIClientProtocol) {
          self.apiClient = apiClient
      }

      func load() async {
          state = .loading
          do {
              let chores = try await apiClient.fetchChores()
              state = .loaded(chores)
          } catch let error as APIError {
              state = .failed(error)
          } catch {
              state = .failed(.network(error))
          }
      }

      func complete(choreID: String) async {
          do {
              try await apiClient.completeChore(id: choreID)
              // Update local state immediately without re-fetching
              guard case .loaded(var chores) = state else { return }
              if let index = chores.firstIndex(where: { $0.id == choreID }) {
                  chores[index] = Chore(
                      id: chores[index].id,
                      name: chores[index].name,
                      description: chores[index].description,
                      status: .completed,
                      dueDate: chores[index].dueDate,
                      assignedToUserID: chores[index].assignedToUserID
                  )
                  state = .loaded(chores)
              }
          } catch let error as APIError {
              errorMessage = error.localizedDescription
          } catch {
              errorMessage = error.localizedDescription
          }
      }
  }
  ```

- [ ] **Step 4: Run (⌘U) — confirm 3 tests pass**

- [ ] **Step 5: Implement `ChoresView.swift`**

  ```swift
  // ios/FamilyHub/Features/Chores/ChoresView.swift
  import SwiftUI

  struct ChoresView: View {
      @State private var viewModel: ChoresViewModel

      init(apiClient: any APIClientProtocol) {
          _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
      }

      var body: some View {
          NavigationStack {
              Group {
                  switch viewModel.state {
                  case .idle, .loading:
                      ProgressView()
                  case .loaded:
                      choresList
                  case .failed(let error):
                      ContentUnavailableView(
                          "Failed to load",
                          systemImage: "exclamationmark.triangle",
                          description: Text(error.localizedDescription)
                      )
                  }
              }
              .navigationTitle("Chores")
              .toolbar {
                  ToolbarItem(placement: .navigationBarTrailing) {
                      Button("Refresh") { Task { await viewModel.load() } }
                  }
              }
              .alert("Error", isPresented: .constant(viewModel.errorMessage != nil)) {
                  Button("OK") { viewModel.errorMessage = nil }
              } message: {
                  Text(viewModel.errorMessage ?? "")
              }
          }
          .task { await viewModel.load() }
      }

      private var choresList: some View {
          List {
              if !viewModel.pendingChores.isEmpty {
                  Section("Pending") {
                      ForEach(viewModel.pendingChores) { chore in
                          NavigationLink {
                              ChoreDetailView(chore: chore, viewModel: viewModel)
                          } label: {
                              ChoreRow(chore: chore)
                          }
                      }
                  }
              }
              if !viewModel.completedChores.isEmpty {
                  Section("Completed") {
                      ForEach(viewModel.completedChores) { chore in
                          ChoreRow(chore: chore)
                      }
                  }
              }
          }
      }
  }

  private struct ChoreRow: View {
      let chore: Chore

      var body: some View {
          VStack(alignment: .leading, spacing: 4) {
              Text(chore.name)
              if let dueDate = chore.dueDate {
                  Text(dueDate.prefix(10)) // Show YYYY-MM-DD portion
                      .font(.caption)
                      .foregroundStyle(chore.status == .overdue ? .red : .secondary)
              }
          }
      }
  }
  ```

- [ ] **Step 6: Build (⌘B)**

- [ ] **Step 7: Commit**

  ```bash
  git add ios/FamilyHub/Features/Chores/ChoresViewModel.swift \
          ios/FamilyHub/Features/Chores/ChoresView.swift \
          ios/FamilyHubTests/Chores/ChoresViewModelTests.swift
  git commit -m "feat: add ChoresViewModel and ChoresView"
  ```

---

### Task 14: ChoreDetailView

**Files:**
- Create: `ios/FamilyHub/Features/Chores/ChoreDetailView.swift`

No separate ViewModel — `ChoreDetailView` calls back into `ChoresViewModel` to complete a chore. Already tested via `ChoresViewModelTests`.

- [ ] **Step 1: Implement `ChoreDetailView.swift`**

  ```swift
  // ios/FamilyHub/Features/Chores/ChoreDetailView.swift
  import SwiftUI

  struct ChoreDetailView: View {
      let chore: Chore
      let viewModel: ChoresViewModel

      @Environment(\.dismiss) private var dismiss
      @State private var isCompleting = false

      var body: some View {
          List {
              Section {
                  LabeledContent("Status", value: chore.status.rawValue.capitalized)
                  if let dueDate = chore.dueDate {
                      LabeledContent("Due", value: String(dueDate.prefix(10)))
                  }
              }

              if !chore.description.isEmpty {
                  Section("Description") {
                      Text(chore.description)
                  }
              }

              if chore.status != .completed {
                  Section {
                      Button {
                          Task {
                              isCompleting = true
                              await viewModel.complete(choreID: chore.id)
                              isCompleting = false
                              dismiss()
                          }
                      } label: {
                          HStack {
                              Spacer()
                              if isCompleting {
                                  ProgressView()
                              } else {
                                  Text("Mark Complete")
                                      .bold()
                              }
                              Spacer()
                          }
                      }
                      .disabled(isCompleting)
                  }
              }
          }
          .navigationTitle(chore.name)
          .navigationBarTitleDisplayMode(.inline)
      }
  }
  ```

- [ ] **Step 2: Build (⌘B)**

- [ ] **Step 3: Commit**

  ```bash
  git add ios/FamilyHub/Features/Chores/ChoreDetailView.swift
  git commit -m "feat: add ChoreDetailView with Mark Complete"
  ```

---

## Chunk 4: Meals + Recipes + Calendar + Integration

### Task 15: MealsViewModel + MealsView

**Files:**
- Create: `ios/FamilyHub/Features/Meals/MealsViewModel.swift`
- Create: `ios/FamilyHub/Features/Meals/MealsView.swift`
- Create: `ios/FamilyHubTests/Meals/MealsViewModelTests.swift`

- [ ] **Step 1: Write the failing tests**

  Create `ios/FamilyHubTests/Meals/MealsViewModelTests.swift`:

  ```swift
  import XCTest
  @testable import FamilyHub

  @MainActor
  final class MealsViewModelTests: XCTestCase {
      private func makeMeal(date: String, mealType: String, name: String) -> MealPlan {
          MealPlan(date: date, mealType: mealType, name: name, notes: "", recipeID: nil)
      }

      func testLoadSuccess() async {
          let fake = FakeAPIClient()
          fake.mealsResult = .success([makeMeal(date: "2026-03-09", mealType: "dinner", name: "Pasta")])
          let viewModel = MealsViewModel(apiClient: fake)

          await viewModel.load()

          guard case .loaded(let meals) = viewModel.state else {
              XCTFail("expected loaded state")
              return
          }
          XCTAssertEqual(meals.count, 1)
          XCTAssertEqual(meals.first?.name, "Pasta")
      }

      func testNavigateWeekChangesCurrentWeek() async {
          let fake = FakeAPIClient()
          let viewModel = MealsViewModel(apiClient: fake)
          let original = viewModel.currentWeek

          viewModel.nextWeek()

          XCTAssertNotEqual(viewModel.currentWeek, original)
          // Next week should be 7 days later
          let diff = Calendar.current.dateComponents([.day], from: original, to: viewModel.currentWeek).day
          XCTAssertEqual(diff, 7)
      }
  }
  ```

- [ ] **Step 2: Run (⌘U) to confirm failure**

- [ ] **Step 3: Implement `MealsViewModel.swift`**

  ```swift
  // ios/FamilyHub/Features/Meals/MealsViewModel.swift
  import Foundation
  import Observation

  @Observable
  @MainActor
  final class MealsViewModel {
      var state: ViewState<[MealPlan]> = .idle
      var currentWeek: Date = Self.startOfCurrentWeek()

      private let apiClient: any APIClientProtocol

      init(apiClient: any APIClientProtocol) {
          self.apiClient = apiClient
      }

      func load() async {
          state = .loading
          do {
              let meals = try await apiClient.fetchMeals(week: currentWeek)
              state = .loaded(meals)
          } catch let error as APIError {
              state = .failed(error)
          } catch {
              state = .failed(.network(error))
          }
      }

      func nextWeek() {
          currentWeek = Calendar.current.date(byAdding: .day, value: 7, to: currentWeek)!
          Task { await load() }
      }

      func previousWeek() {
          currentWeek = Calendar.current.date(byAdding: .day, value: -7, to: currentWeek)!
          Task { await load() }
      }

      private static func startOfCurrentWeek() -> Date {
          let calendar = Calendar(identifier: .iso8601)
          return calendar.dateInterval(of: .weekOfYear, for: Date())!.start
      }
  }
  ```

- [ ] **Step 4: Run (⌘U) — confirm tests pass**

- [ ] **Step 5: Implement `MealsView.swift`**

  ```swift
  // ios/FamilyHub/Features/Meals/MealsView.swift
  import SwiftUI

  struct MealsView: View {
      @State private var viewModel: MealsViewModel

      private let days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
      private let mealTypes = ["breakfast", "lunch", "dinner"]

      init(apiClient: any APIClientProtocol) {
          _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
      }

      var body: some View {
          NavigationStack {
              Group {
                  switch viewModel.state {
                  case .idle, .loading:
                      ProgressView()
                  case .loaded(let meals):
                      mealsTable(meals)
                  case .failed(let error):
                      ContentUnavailableView(
                          "Failed to load",
                          systemImage: "exclamationmark.triangle",
                          description: Text(error.localizedDescription)
                      )
                  }
              }
              .navigationTitle(weekTitle)
              .toolbar {
                  ToolbarItem(placement: .navigationBarLeading) {
                      Button("< Prev") { viewModel.previousWeek() }
                  }
                  ToolbarItem(placement: .navigationBarTrailing) {
                      Button("Next >") { viewModel.nextWeek() }
                  }
              }
          }
          .task { await viewModel.load() }
      }

      private var weekTitle: String {
          let formatter = DateFormatter()
          formatter.dateFormat = "d MMM"
          let start = viewModel.currentWeek
          let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
          return "\(formatter.string(from: start)) – \(formatter.string(from: end))"
      }

      private func mealsTable(_ meals: [MealPlan]) -> some View {
          List {
              ForEach(Array(days.enumerated()), id: \.offset) { index, day in
                  let date = dayDate(offset: index)
                  Section(day + " " + dateLabel(date)) {
                      ForEach(mealTypes, id: \.self) { mealType in
                          let meal = meals.first(where: { $0.date == dateString(date) && $0.mealType == mealType })
                          HStack {
                              Text(mealType.capitalized)
                                  .foregroundStyle(.secondary)
                                  .frame(width: 90, alignment: .leading)
                              Text(meal?.name ?? "—")
                          }
                      }
                  }
              }
          }
      }

      private func dayDate(offset: Int) -> Date {
          Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
      }

      private func dateLabel(_ date: Date) -> String {
          let formatter = DateFormatter()
          formatter.dateFormat = "d"
          return formatter.string(from: date)
      }

      private func dateString(_ date: Date) -> String {
          let formatter = DateFormatter()
          formatter.dateFormat = "yyyy-MM-dd"
          return formatter.string(from: date)
      }
  }
  ```

- [ ] **Step 6: Build (⌘B)**

- [ ] **Step 7: Commit**

  ```bash
  git add ios/FamilyHub/Features/Meals/ ios/FamilyHubTests/Meals/
  git commit -m "feat: add MealsViewModel and MealsView"
  ```

---

### Task 16: RecipesViewModel + RecipesView + RecipeDetailView

**Files:**
- Create: `ios/FamilyHub/Features/Recipes/RecipesViewModel.swift`
- Create: `ios/FamilyHub/Features/Recipes/RecipesView.swift`
- Create: `ios/FamilyHub/Features/Recipes/RecipeDetailView.swift`
- Create: `ios/FamilyHubTests/Recipes/RecipesViewModelTests.swift`

- [ ] **Step 1: Write the failing tests**

  Create `ios/FamilyHubTests/Recipes/RecipesViewModelTests.swift`:

  ```swift
  import XCTest
  @testable import FamilyHub

  @MainActor
  final class RecipesViewModelTests: XCTestCase {
      private func makeRecipe(id: String, title: String) -> Recipe {
          Recipe(id: id, title: title, steps: [], ingredients: [], servings: nil, prepTime: nil, cookTime: nil, hasImage: false)
      }

      func testLoadSuccess() async {
          let fake = FakeAPIClient()
          fake.recipesResult = .success([makeRecipe(id: "1", title: "Pasta")])
          let viewModel = RecipesViewModel(apiClient: fake)

          await viewModel.load()

          guard case .loaded(let recipes) = viewModel.state else {
              XCTFail("expected loaded state")
              return
          }
          XCTAssertEqual(recipes.count, 1)
          XCTAssertEqual(recipes.first?.title, "Pasta")
      }

      func testLoadFailure() async {
          let fake = FakeAPIClient()
          fake.recipesResult = .failure(APIError.server(503))
          let viewModel = RecipesViewModel(apiClient: fake)

          await viewModel.load()

          guard case .failed = viewModel.state else {
              XCTFail("expected failed state")
              return
          }
      }
  }
  ```

- [ ] **Step 2: Run (⌘U) to confirm failure**

- [ ] **Step 3: Implement `RecipesViewModel.swift`**

  ```swift
  // ios/FamilyHub/Features/Recipes/RecipesViewModel.swift
  import Foundation
  import Observation

  @Observable
  @MainActor
  final class RecipesViewModel {
      var state: ViewState<[Recipe]> = .idle

      private let apiClient: any APIClientProtocol

      init(apiClient: any APIClientProtocol) {
          self.apiClient = apiClient
      }

      func load() async {
          state = .loading
          do {
              let recipes = try await apiClient.fetchRecipes()
              state = .loaded(recipes)
          } catch let error as APIError {
              state = .failed(error)
          } catch {
              state = .failed(.network(error))
          }
      }
  }
  ```

- [ ] **Step 4: Run (⌘U) — confirm tests pass**

- [ ] **Step 5: Implement `RecipesView.swift`**

  Two-column grid of recipe cards.

  ```swift
  // ios/FamilyHub/Features/Recipes/RecipesView.swift
  import SwiftUI

  struct RecipesView: View {
      @State private var viewModel: RecipesViewModel
      private let apiClient: any APIClientProtocol

      private let columns = [GridItem(.flexible()), GridItem(.flexible())]

      init(apiClient: any APIClientProtocol) {
          self.apiClient = apiClient
          _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
      }

      var body: some View {
          NavigationStack {
              Group {
                  switch viewModel.state {
                  case .idle, .loading:
                      ProgressView()
                  case .loaded(let recipes):
                      ScrollView {
                          LazyVGrid(columns: columns, spacing: 16) {
                              ForEach(recipes) { recipe in
                                  NavigationLink {
                                      RecipeDetailView(recipe: recipe, apiClient: apiClient)
                                  } label: {
                                      RecipeCard(recipe: recipe)
                                  }
                                  .buttonStyle(.plain)
                              }
                          }
                          .padding()
                      }
                  case .failed(let error):
                      ContentUnavailableView(
                          "Failed to load",
                          systemImage: "exclamationmark.triangle",
                          description: Text(error.localizedDescription)
                      )
                  }
              }
              .navigationTitle("Recipes")
          }
          .task { await viewModel.load() }
      }
  }

  private struct RecipeCard: View {
      let recipe: Recipe

      var body: some View {
          VStack(alignment: .leading, spacing: 8) {
              RoundedRectangle(cornerRadius: 8)
                  .fill(Color.secondary.opacity(0.2))
                  .aspectRatio(4/3, contentMode: .fit)
                  .overlay {
                      if !recipe.hasImage {
                          Image(systemName: "fork.knife")
                              .foregroundStyle(.secondary)
                      }
                  }

              Text(recipe.title)
                  .font(.subheadline.bold())
                  .lineLimit(2)

              if let servings = recipe.servings {
                  Text("\(servings) servings")
                      .font(.caption)
                      .foregroundStyle(.secondary)
              }
          }
          .padding(8)
          .background(Color(.secondarySystemBackground))
          .clipShape(RoundedRectangle(cornerRadius: 12))
      }
  }
  ```

- [ ] **Step 6: Implement `RecipeDetailView.swift`**

  ```swift
  // ios/FamilyHub/Features/Recipes/RecipeDetailView.swift
  import SwiftUI

  struct RecipeDetailView: View {
      let recipe: Recipe
      let apiClient: any APIClientProtocol

      @State private var cookMode = false
      @State private var fullRecipe: Recipe?
      @State private var isLoading = true

      var body: some View {
          let displayed = fullRecipe ?? recipe
          List {
              // Meta section
              Section {
                  if let servings = displayed.servings {
                      LabeledContent("Servings", value: "\(servings)")
                  }
                  if let prep = displayed.prepTime {
                      LabeledContent("Prep time", value: prep)
                  }
                  if let cook = displayed.cookTime {
                      LabeledContent("Cook time", value: cook)
                  }
              }

              // Ingredients
              if let ingredients = displayed.ingredients, !ingredients.isEmpty {
                  ForEach(ingredients, id: \.name) { group in
                      Section(group.name.isEmpty ? "Ingredients" : group.name) {
                          ForEach(group.items, id: \.self) { item in
                              Text(item)
                          }
                      }
                  }
              }

              // Steps
              if let steps = displayed.steps, !steps.isEmpty {
                  Section("Steps") {
                      ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                          HStack(alignment: .top, spacing: 12) {
                              Text("\(index + 1)")
                                  .font(.headline)
                                  .foregroundStyle(.secondary)
                                  .frame(width: 24)
                              Text(step)
                          }
                      }
                  }
              }
          }
          .navigationTitle(displayed.title)
          .navigationBarTitleDisplayMode(.large)
          .toolbar {
              ToolbarItem(placement: .navigationBarTrailing) {
                  Button(cookMode ? "Exit Cook Mode" : "Cook Mode") {
                      cookMode.toggle()
                      UIApplication.shared.isIdleTimerDisabled = cookMode
                  }
              }
          }
          .task {
              // Fetch full recipe details (list endpoint may omit steps/ingredients)
              if let full = try? await apiClient.fetchRecipe(id: recipe.id) {
                  fullRecipe = full
              }
              isLoading = false
          }
          .overlay {
              if isLoading { ProgressView() }
          }
      }
  }
  ```

- [ ] **Step 7: Build (⌘B)**

- [ ] **Step 8: Commit**

  ```bash
  git add ios/FamilyHub/Features/Recipes/ ios/FamilyHubTests/Recipes/
  git commit -m "feat: add RecipesViewModel, RecipesView, and RecipeDetailView"
  ```

---

### Task 17: CalendarViewModel + CalendarView

**Files:**
- Create: `ios/FamilyHub/Features/Calendar/CalendarViewModel.swift`
- Create: `ios/FamilyHub/Features/Calendar/CalendarView.swift`
- Create: `ios/FamilyHubTests/Calendar/CalendarViewModelTests.swift`

- [ ] **Step 1: Write the failing tests**

  Create `ios/FamilyHubTests/Calendar/CalendarViewModelTests.swift`:

  ```swift
  import XCTest
  @testable import FamilyHub

  @MainActor
  final class CalendarViewModelTests: XCTestCase {
      private func makeChore(id: String, name: String, dueDate: String) -> Chore {
          Chore(id: id, name: name, description: "", status: .pending,
                dueDate: dueDate + "T00:00:00Z", assignedToUserID: nil)
      }

      func testLoadSuccess() async {
          let fake = FakeAPIClient()
          fake.calendarResult = .success([makeChore(id: "1", name: "Clean", dueDate: "2026-03-15")])
          let viewModel = CalendarViewModel(apiClient: fake)

          await viewModel.load()

          guard case .loaded(let chores) = viewModel.state else {
              XCTFail("expected loaded state")
              return
          }
          XCTAssertEqual(chores.count, 1)
      }

      func testChoresForDayFiltersCorrectly() async {
          let fake = FakeAPIClient()
          fake.calendarResult = .success([
              makeChore(id: "1", name: "A", dueDate: "2026-03-15"),
              makeChore(id: "2", name: "B", dueDate: "2026-03-20"),
          ])
          let viewModel = CalendarViewModel(apiClient: fake)
          await viewModel.load()

          let march15 = DateComponents(calendar: .current, year: 2026, month: 3, day: 15).date!
          let chores = viewModel.chores(for: march15)

          XCTAssertEqual(chores.count, 1)
          XCTAssertEqual(chores.first?.name, "A")
      }
  }
  ```

- [ ] **Step 2: Run (⌘U) to confirm failure**

- [ ] **Step 3: Implement `CalendarViewModel.swift`**

  ```swift
  // ios/FamilyHub/Features/Calendar/CalendarViewModel.swift
  import Foundation
  import Observation

  @Observable
  @MainActor
  final class CalendarViewModel {
      var state: ViewState<[Chore]> = .idle
      var currentMonth: Date = {
          let components = Calendar.current.dateComponents([.year, .month], from: Date())
          return Calendar.current.date(from: components)!
      }()
      var selectedDay: Date?

      private let apiClient: any APIClientProtocol

      init(apiClient: any APIClientProtocol) {
          self.apiClient = apiClient
      }

      func load() async {
          state = .loading
          do {
              let chores = try await apiClient.fetchCalendar(month: currentMonth)
              state = .loaded(chores)
          } catch let error as APIError {
              state = .failed(error)
          } catch {
              state = .failed(.network(error))
          }
      }

      func nextMonth() {
          currentMonth = Calendar.current.date(byAdding: .month, value: 1, to: currentMonth)!
          Task { await load() }
      }

      func previousMonth() {
          currentMonth = Calendar.current.date(byAdding: .month, value: -1, to: currentMonth)!
          Task { await load() }
      }

      func chores(for day: Date) -> [Chore] {
          guard case .loaded(let chores) = state else { return [] }
          let dayString = iso8601DayString(day)
          return chores.filter { chore in
              guard let dueDate = chore.dueDate else { return false }
              return dueDate.hasPrefix(dayString)
          }
      }

      private func iso8601DayString(_ date: Date) -> String {
          let formatter = DateFormatter()
          formatter.dateFormat = "yyyy-MM-dd"
          return formatter.string(from: date)
      }
  }
  ```

- [ ] **Step 4: Run (⌘U) — confirm tests pass**

- [ ] **Step 5: Implement `CalendarView.swift`**

  Monthly grid using SwiftUI's date grid pattern, with agenda list below.

  ```swift
  // ios/FamilyHub/Features/Calendar/CalendarView.swift
  import SwiftUI

  struct CalendarView: View {
      @State private var viewModel: CalendarViewModel

      private let columns = Array(repeating: GridItem(.flexible()), count: 7)
      private let weekdaySymbols = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

      init(apiClient: any APIClientProtocol) {
          _viewModel = State(wrappedValue: CalendarViewModel(apiClient: apiClient))
      }

      var body: some View {
          NavigationStack {
              VStack(spacing: 0) {
                  calendarGrid
                  Divider()
                  agendaList
              }
              .navigationTitle(monthTitle)
              .toolbar {
                  ToolbarItem(placement: .navigationBarLeading) {
                      Button("< Prev") { viewModel.previousMonth() }
                  }
                  ToolbarItem(placement: .navigationBarTrailing) {
                      Button("Next >") { viewModel.nextMonth() }
                  }
              }
          }
          .task { await viewModel.load() }
      }

      private var monthTitle: String {
          let formatter = DateFormatter()
          formatter.dateFormat = "MMMM yyyy"
          return formatter.string(from: viewModel.currentMonth)
      }

      private var calendarGrid: some View {
          VStack(spacing: 4) {
              // Weekday headers
              LazyVGrid(columns: columns, spacing: 0) {
                  ForEach(Array(weekdaySymbols.enumerated()), id: \.offset) { _, symbol in
                      Text(symbol)
                          .font(.caption.bold())
                          .foregroundStyle(.secondary)
                          .frame(maxWidth: .infinity)
                  }
              }
              // Day cells
              LazyVGrid(columns: columns, spacing: 4) {
                  ForEach(daysInMonth, id: \.self) { day in
                      DayCell(
                          date: day,
                          isSelected: Calendar.current.isDate(day, inSameDayAs: viewModel.selectedDay ?? .distantPast),
                          hasChores: !viewModel.chores(for: day).isEmpty
                      )
                      .onTapGesture {
                          viewModel.selectedDay = day
                      }
                  }
              }
          }
          .padding()
      }

      private var agendaList: some View {
          Group {
              if let selectedDay = viewModel.selectedDay {
                  let chores = viewModel.chores(for: selectedDay)
                  if chores.isEmpty {
                      ContentUnavailableView("No chores", systemImage: "checkmark.circle")
                          .frame(maxHeight: .infinity)
                  } else {
                      List(chores) { chore in
                          Text(chore.name)
                      }
                  }
              } else {
                  ContentUnavailableView("Select a day", systemImage: "calendar")
                      .frame(maxHeight: .infinity)
              }
          }
      }

      private var daysInMonth: [Date] {
          let calendar = Calendar(identifier: .iso8601)
          guard let range = calendar.range(of: .day, in: .month, for: viewModel.currentMonth),
                let firstDay = calendar.date(from: calendar.dateComponents([.year, .month], from: viewModel.currentMonth))
          else { return [] }

          // Pad with leading empty days (offset from Monday)
          let weekdayOffset = (calendar.component(.weekday, from: firstDay) + 5) % 7
          var days: [Date] = Array(repeating: Date.distantPast, count: weekdayOffset)
          days += range.compactMap { day in
              calendar.date(byAdding: .day, value: day - 1, to: firstDay)
          }
          return days
      }
  }

  private struct DayCell: View {
      let date: Date
      let isSelected: Bool
      let hasChores: Bool

      var body: some View {
          let isPlaceholder = date == .distantPast
          VStack(spacing: 2) {
              Text(isPlaceholder ? "" : dayNumber)
                  .font(.callout)
                  .foregroundStyle(isSelected ? .white : .primary)
                  .frame(width: 32, height: 32)
                  .background(isSelected ? Color.blue : Color.clear)
                  .clipShape(Circle())

              Circle()
                  .fill(hasChores && !isPlaceholder ? Color.blue : Color.clear)
                  .frame(width: 5, height: 5)
          }
      }

      private var dayNumber: String {
          let formatter = DateFormatter()
          formatter.dateFormat = "d"
          return formatter.string(from: date)
      }
  }
  ```

- [ ] **Step 6: Build (⌘B)**

- [ ] **Step 7: Commit**

  ```bash
  git add ios/FamilyHub/Features/Calendar/ ios/FamilyHubTests/Calendar/
  git commit -m "feat: add CalendarViewModel and CalendarView"
  ```

---

### Task 18: Wire up TabView and final integration

**Files:**
- Modify: `ios/FamilyHub/FamilyHubApp.swift` (remove placeholder `ContentView` stub)
- Create: `ios/FamilyHub/Features/ContentView.swift`

- [ ] **Step 1: Create `ContentView.swift`**

  Replace the temporary `ContentView` stub with the real `TabView`:

  ```swift
  // ios/FamilyHub/Features/ContentView.swift
  import SwiftUI

  struct ContentView: View {
      let apiClient: any APIClientProtocol

      var body: some View {
          TabView {
              DashboardView(apiClient: apiClient)
                  .tabItem {
                      Label("Dashboard", systemImage: "house")
                  }

              ChoresView(apiClient: apiClient)
                  .tabItem {
                      Label("Chores", systemImage: "checklist")
                  }

              MealsView(apiClient: apiClient)
                  .tabItem {
                      Label("Meals", systemImage: "fork.knife")
                  }

              RecipesView(apiClient: apiClient)
                  .tabItem {
                      Label("Recipes", systemImage: "book")
                  }

              CalendarView(apiClient: apiClient)
                  .tabItem {
                      Label("Calendar", systemImage: "calendar")
                  }
          }
      }
  }
  ```

- [ ] **Step 2: Remove the stub from `FamilyHubApp.swift`**

  Delete the `// Temporary — replaced in Task 18` block from `FamilyHubApp.swift`. The file now only contains the `@main` struct (which imports `ContentView` from the new file).

- [ ] **Step 3: Build (⌘B) — ensure entire project compiles**

  Expected: success with no warnings.

- [ ] **Step 4: Run on simulator**

  Product → Run (⌘R). Select an iPhone 16 (or latest available) simulator.

  Verify:
  - Login screen appears
  - "Sign In" button is visible
  - Tapping Sign In opens the Authentik login page in an in-app browser
  - After logging in, the tab bar appears with all 5 tabs
  - Each tab loads without crashing

- [ ] **Step 5: Run all tests (⌘U)**

  Expected: all ViewModel tests pass (KeychainStore, Dashboard, Chores, Meals, Recipes, Calendar).

- [ ] **Step 6: Commit**

  ```bash
  git add ios/FamilyHub/Features/ContentView.swift ios/FamilyHub/FamilyHubApp.swift
  git commit -m "feat: wire up TabView with all five feature tabs — iOS app complete"
  ```

---

*Backend Tasks 1–5 are complete on branch `feature/ios-app`. This plan covers the SwiftUI iOS app (Tasks 6–18).*
