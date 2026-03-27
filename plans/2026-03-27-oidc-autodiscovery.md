# OIDC Auto-Discovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the iOS first-run setup form (4 manual fields) with a single base URL entry — the app fetches OIDC config from the server and completes setup automatically.

**Architecture:** The server exposes an unauthenticated `GET /api/client-config` endpoint that returns the OIDC `clientID` and `issuer` URL. The iOS app uses the issuer to fetch the standard OIDC discovery document (`/.well-known/openid-configuration`) and extracts the authorization and token endpoints. All derived values are persisted to UserDefaults so discovery only runs once.

**Tech Stack:** Go + chi (server); Swift + URLSession (iOS); OIDC Discovery spec (RFC 8414)

---

## File Map

**Server (modify):**
- `server/internal/handlers/api.go` — add `clientID` + `oidcIssuer` fields to `APIHandler`, add `ClientConfig` method
- `server/internal/server/server.go` — pass new fields to `NewAPIHandler`, register `GET /api/client-config` route
- `server/internal/handlers/api_test.go` — add test for `ClientConfig` handler

**iOS (modify):**
- `ios/FamilyHub/FamilyHub/Config/ConfigStore.swift` — add `applyDiscovery(_:)` method; `isConfigured` already checks all 4 fields, no change needed there
- `ios/FamilyHub/FamilyHub/Features/Settings/ConfigurationFormView.swift` — replace 4-field form with single baseURL field + async Connect button
- `ios/FamilyHub/FamilyHub/Features/Settings/SetupView.swift` — remove now-unused `onSave` callback pattern; pass discovery service

**iOS (create):**
- `ios/FamilyHub/FamilyHub/Config/OIDCDiscoveryService.swift` — protocol + URLSession implementation + result/error types
- `ios/FamilyHub/FamilyHubTests/Config/OIDCDiscoveryServiceTests.swift` — tests using a fake HTTP session

**iOS (test updates):**
- `ios/FamilyHub/FamilyHubTests/Auth/ConfigStoreTests.swift` — update to cover `applyDiscovery(_:)`
- `ios/FamilyHub/FamilyHubTests/Auth/OIDCConfigTests.swift` — no structural change needed (ConfigStore still has all 4 writable fields)

---

## Task 1: Server — `GET /api/client-config` endpoint

**Files:**
- Modify: `server/internal/handlers/api.go`
- Modify: `server/internal/server/server.go`
- Modify: `server/internal/handlers/api_test.go`

- [ ] **Step 1: Write the failing test**

Add to `server/internal/handlers/api_test.go`:

```go
func TestAPIHandler_ClientConfig(t *testing.T) {
    tests := []struct {
        name         string
        clientID     string
        oidcIssuer   string
        wantStatus   int
        wantClientID string
        wantIssuer   string
    }{
        {
            name:         "returns client config as JSON",
            clientID:     "familyhub-ios",
            oidcIssuer:   "https://auth.example.com/application/o/familyhub",
            wantStatus:   http.StatusOK,
            wantClientID: "familyhub-ios",
            wantIssuer:   "https://auth.example.com/application/o/familyhub",
        },
        {
            name:       "empty config still returns 200",
            clientID:   "",
            oidcIssuer: "",
            wantStatus: http.StatusOK,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, "", tt.clientID, tt.oidcIssuer)

            request := httptest.NewRequest(http.MethodGet, "/api/client-config", nil)
            recorder := httptest.NewRecorder()
            handler.ClientConfig(recorder, request)

            if recorder.Code != tt.wantStatus {
                t.Fatalf("want status %d, got %d", tt.wantStatus, recorder.Code)
            }

            var body struct {
                ClientID string `json:"clientID"`
                Issuer   string `json:"issuer"`
            }
            if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
                t.Fatalf("decoding response: %v", err)
            }
            if body.ClientID != tt.wantClientID {
                t.Errorf("want clientID %q, got %q", tt.wantClientID, body.ClientID)
            }
            if body.Issuer != tt.wantIssuer {
                t.Errorf("want issuer %q, got %q", tt.wantIssuer, body.Issuer)
            }
        })
    }
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd server && go test ./internal/handlers/ -run TestAPIHandler_ClientConfig -v
```

Expected: compile error — `NewAPIHandler` does not accept `clientID`/`oidcIssuer` parameters, and `ClientConfig` does not exist.

- [ ] **Step 3: Update `NewAPIHandler` and add `ClientConfig` in `api.go`**

Add two fields to `APIHandler`:

```go
type APIHandler struct {
    choreRepo       repository.ChoreRepository
    userRepo        repository.UserRepository
    categoryRepo    repository.CategoryRepository
    assignmentRepo  repository.ChoreAssignmentRepository
    tokenRepo       repository.APITokenRepository
    choreService    *services.ChoreService
    mealPlanRepo    repository.MealPlanRepository
    recipeRepo      repository.RecipeRepository
    oidcUserInfoURL string
    clientID        string
    oidcIssuer      string
}
```

Update `NewAPIHandler` signature (add the two new params at the end):

```go
func NewAPIHandler(
    choreRepo repository.ChoreRepository,
    userRepo repository.UserRepository,
    categoryRepo repository.CategoryRepository,
    assignmentRepo repository.ChoreAssignmentRepository,
    tokenRepo repository.APITokenRepository,
    choreService *services.ChoreService,
    mealPlanRepo repository.MealPlanRepository,
    recipeRepo repository.RecipeRepository,
    oidcUserInfoURL string,
    clientID string,
    oidcIssuer string,
) *APIHandler {
    return &APIHandler{
        choreRepo:       choreRepo,
        userRepo:        userRepo,
        categoryRepo:    categoryRepo,
        assignmentRepo:  assignmentRepo,
        tokenRepo:       tokenRepo,
        choreService:    choreService,
        mealPlanRepo:    mealPlanRepo,
        recipeRepo:      recipeRepo,
        oidcUserInfoURL: oidcUserInfoURL,
        clientID:        clientID,
        oidcIssuer:      oidcIssuer,
    }
}
```

Add the handler method (anywhere in `api.go`):

```go
func (handler *APIHandler) ClientConfig(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "clientID": handler.clientID,
        "issuer":   handler.oidcIssuer,
    })
}
```

- [ ] **Step 4: Update the existing call to `NewAPIHandler` in `server.go`**

In `server/internal/server/server.go`, change:

```go
apiHandler := handlers.NewAPIHandler(choreRepo, userRepo, categoryRepo, assignmentRepo, tokenRepo, choreService, mealPlanRepo, recipeRepo, cfg.OIDCUserInfoURL)
```

to:

```go
apiHandler := handlers.NewAPIHandler(choreRepo, userRepo, categoryRepo, assignmentRepo, tokenRepo, choreService, mealPlanRepo, recipeRepo, cfg.OIDCUserInfoURL, cfg.OIDCClientID, cfg.OIDCIssuer)
```

- [ ] **Step 5: Register the route in `server.go`**

Add alongside the other public routes (near `/health`):

```go
router.Get("/api/client-config", apiHandler.ClientConfig)
```

- [ ] **Step 6: Fix the existing test call to `NewAPIHandler` in `api_test.go`**

The existing test at the top of `api_test.go` calls `NewAPIHandler` with 9 args. Update it to pass two empty strings for the new params:

```go
apiHandler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil, "", "", "")
```

- [ ] **Step 7: Run the tests**

```bash
cd server && go test ./internal/handlers/ -run TestAPIHandler_ClientConfig -v
cd server && go test ./...
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add server/internal/handlers/api.go server/internal/server/server.go server/internal/handlers/api_test.go
git commit -m "feat(api): add GET /api/client-config endpoint for iOS OIDC discovery"
```

---

## Task 2: iOS — `OIDCDiscoveryService` protocol and implementation

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Config/OIDCDiscoveryService.swift`
- Create: `ios/FamilyHub/FamilyHubTests/Config/OIDCDiscoveryServiceTests.swift`

The service does two network calls:
1. `GET {baseURL}/api/client-config` → `{"clientID": "...", "issuer": "..."}`
2. `GET {issuer}/.well-known/openid-configuration` → standard OIDC discovery doc

- [ ] **Step 1: Create the test file with a fake URL session**

Create `ios/FamilyHub/FamilyHubTests/Config/OIDCDiscoveryServiceTests.swift`:

```swift
import XCTest
@testable import FamilyHub

final class OIDCDiscoveryServiceTests: XCTestCase {
    private func makeService(responses: [URL: (Data, Int)]) -> OIDCDiscoveryService {
        URLSessionOIDCDiscoveryService(session: FakeURLSession(responses: responses))
    }

    func testSuccessfulDiscovery() async throws {
        let baseURL = URL(string: "https://hub.example.com")!
        let issuerURL = URL(string: "https://auth.example.com/application/o/familyhub")!

        let clientConfigData = try JSONEncoder().encode([
            "clientID": "familyhub-ios",
            "issuer": issuerURL.absoluteString
        ])

        let discoveryData = try JSONEncoder().encode([
            "authorization_endpoint": "https://auth.example.com/authorize",
            "token_endpoint": "https://auth.example.com/token"
        ])

        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (clientConfigData, 200),
            URL(string: "https://auth.example.com/application/o/familyhub/.well-known/openid-configuration")!: (discoveryData, 200),
        ])

        let result = try await service.discover(baseURL: baseURL)

        XCTAssertEqual(result.clientID, "familyhub-ios")
        XCTAssertEqual(result.authorizationEndpoint.absoluteString, "https://auth.example.com/authorize")
        XCTAssertEqual(result.tokenEndpoint.absoluteString, "https://auth.example.com/token")
    }

    func testClientConfigHTTPErrorThrows() async {
        let baseURL = URL(string: "https://hub.example.com")!
        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (Data(), 500),
        ])

        do {
            _ = try await service.discover(baseURL: baseURL)
            XCTFail("Expected error")
        } catch OIDCDiscoveryError.clientConfigFetchFailed(let statusCode) {
            XCTAssertEqual(statusCode, 500)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testDiscoveryDocumentHTTPErrorThrows() async throws {
        let baseURL = URL(string: "https://hub.example.com")!
        let issuerURL = URL(string: "https://auth.example.com/application/o/familyhub")!

        let clientConfigData = try JSONEncoder().encode([
            "clientID": "familyhub-ios",
            "issuer": issuerURL.absoluteString
        ])

        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (clientConfigData, 200),
            URL(string: "https://auth.example.com/application/o/familyhub/.well-known/openid-configuration")!: (Data(), 404),
        ])

        do {
            _ = try await service.discover(baseURL: baseURL)
            XCTFail("Expected error")
        } catch OIDCDiscoveryError.discoveryDocumentFetchFailed(let statusCode) {
            XCTAssertEqual(statusCode, 404)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testMissingIssuerThrows() async throws {
        let baseURL = URL(string: "https://hub.example.com")!
        let clientConfigData = try JSONEncoder().encode(["clientID": "familyhub-ios"])

        let service = makeService(responses: [
            URL(string: "https://hub.example.com/api/client-config")!: (clientConfigData, 200),
        ])

        do {
            _ = try await service.discover(baseURL: baseURL)
            XCTFail("Expected error")
        } catch OIDCDiscoveryError.missingField(let field) {
            XCTAssertEqual(field, "issuer")
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }
}

// MARK: - Fake

final class FakeURLSession: URLSessionProtocol {
    private let responses: [URL: (Data, Int)]

    init(responses: [URL: (Data, Int)]) {
        self.responses = responses
    }

    func data(from url: URL) async throws -> (Data, URLResponse) {
        guard let (data, statusCode) = responses[url] else {
            throw URLError(.badURL)
        }
        let response = HTTPURLResponse(url: url, statusCode: statusCode, httpVersion: nil, headerFields: nil)!
        return (data, response)
    }
}
```

- [ ] **Step 2: Run test to verify it fails (compile error expected)**

Open the Xcode project or run:
```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | grep -E "error:|FAILED|PASSED"
```

Expected: compile error — `OIDCDiscoveryService`, `URLSessionOIDCDiscoveryService`, `OIDCDiscoveryError`, `URLSessionProtocol` not found.

- [ ] **Step 3: Create `OIDCDiscoveryService.swift`**

Create `ios/FamilyHub/FamilyHub/Config/OIDCDiscoveryService.swift`:

```swift
import Foundation

struct OIDCDiscoveryResult {
    let clientID: String
    let authorizationEndpoint: URL
    let tokenEndpoint: URL
}

enum OIDCDiscoveryError: LocalizedError {
    case clientConfigFetchFailed(Int)
    case discoveryDocumentFetchFailed(Int)
    case missingField(String)
    case invalidURL(String)

    var errorDescription: String? {
        switch self {
        case .clientConfigFetchFailed(let code):
            return "Failed to fetch server config (HTTP \(code))"
        case .discoveryDocumentFetchFailed(let code):
            return "Failed to fetch OIDC discovery document (HTTP \(code))"
        case .missingField(let field):
            return "Server config missing required field: \(field)"
        case .invalidURL(let value):
            return "Invalid URL in server config: \(value)"
        }
    }
}

protocol URLSessionProtocol {
    func data(from url: URL) async throws -> (Data, URLResponse)
}

extension URLSession: URLSessionProtocol {
    func data(from url: URL) async throws -> (Data, URLResponse) {
        try await data(from: url, delegate: nil)
    }
}

protocol OIDCDiscoveryService {
    func discover(baseURL: URL) async throws -> OIDCDiscoveryResult
}

final class URLSessionOIDCDiscoveryService: OIDCDiscoveryService {
    private let session: URLSessionProtocol

    init(session: URLSessionProtocol = URLSession.shared) {
        self.session = session
    }

    func discover(baseURL: URL) async throws -> OIDCDiscoveryResult {
        let clientConfig = try await fetchClientConfig(baseURL: baseURL)
        return try await fetchDiscoveryDocument(clientConfig: clientConfig)
    }

    private func fetchClientConfig(baseURL: URL) async throws -> (clientID: String, issuer: URL) {
        let configURL = baseURL.appending(path: "api/client-config")
        let (data, response) = try await session.data(from: configURL)

        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
            throw OIDCDiscoveryError.clientConfigFetchFailed(statusCode)
        }

        let json = try JSONDecoder().decode([String: String].self, from: data)

        guard let clientID = json["clientID"], !clientID.isEmpty else {
            throw OIDCDiscoveryError.missingField("clientID")
        }
        guard let issuerString = json["issuer"], !issuerString.isEmpty else {
            throw OIDCDiscoveryError.missingField("issuer")
        }
        guard let issuerURL = URL(string: issuerString) else {
            throw OIDCDiscoveryError.invalidURL(issuerString)
        }

        return (clientID: clientID, issuer: issuerURL)
    }

    private func fetchDiscoveryDocument(clientConfig: (clientID: String, issuer: URL)) async throws -> OIDCDiscoveryResult {
        let discoveryURL = clientConfig.issuer.appending(path: ".well-known/openid-configuration")
        let (data, response) = try await session.data(from: discoveryURL)

        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
            throw OIDCDiscoveryError.discoveryDocumentFetchFailed(statusCode)
        }

        let json = try JSONDecoder().decode([String: String].self, from: data)

        guard let authString = json["authorization_endpoint"], !authString.isEmpty else {
            throw OIDCDiscoveryError.missingField("authorization_endpoint")
        }
        guard let tokenString = json["token_endpoint"], !tokenString.isEmpty else {
            throw OIDCDiscoveryError.missingField("token_endpoint")
        }
        guard let authURL = URL(string: authString) else {
            throw OIDCDiscoveryError.invalidURL(authString)
        }
        guard let tokenURL = URL(string: tokenString) else {
            throw OIDCDiscoveryError.invalidURL(tokenString)
        }

        return OIDCDiscoveryResult(
            clientID: clientConfig.clientID,
            authorizationEndpoint: authURL,
            tokenEndpoint: tokenURL
        )
    }
}
```

- [ ] **Step 4: Add both files to the Xcode target**

In Xcode: right-click `Config` group → Add Files → select `OIDCDiscoveryService.swift` (target: FamilyHub). Right-click `Config` group in the test target → Add Files → select `OIDCDiscoveryServiceTests.swift` (target: FamilyHubTests).

- [ ] **Step 5: Run the tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/OIDCDiscoveryServiceTests 2>&1 | grep -E "Test (Case|Suite)|error:|FAILED|passed|failed"
```

Expected: all 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Config/OIDCDiscoveryService.swift ios/FamilyHub/FamilyHubTests/Config/OIDCDiscoveryServiceTests.swift
git commit -m "feat(ios): add OIDCDiscoveryService for auto-configuration from base URL"
```

---

## Task 3: iOS — Update `ConfigStore` to support `applyDiscovery`

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Config/ConfigStore.swift`
- Modify: `ios/FamilyHub/FamilyHubTests/Auth/ConfigStoreTests.swift`

`ConfigStore` already stores `clientID`, `authorizationEndpoint`, and `tokenEndpoint`. We only need to add a convenience method so callers don't manually set each field.

- [ ] **Step 1: Write the new test**

Add to `ConfigStoreTests.swift`:

```swift
func testApplyDiscoveryPopulatesFields() {
    let store = makeStore()
    store.baseURL = "https://hub.example.com"

    let result = OIDCDiscoveryResult(
        clientID: "familyhub-ios",
        authorizationEndpoint: URL(string: "https://auth.example.com/authorize")!,
        tokenEndpoint: URL(string: "https://auth.example.com/token")!
    )
    store.applyDiscovery(result)

    XCTAssertEqual(store.clientID, "familyhub-ios")
    XCTAssertEqual(store.authorizationEndpoint, "https://auth.example.com/authorize")
    XCTAssertEqual(store.tokenEndpoint, "https://auth.example.com/token")
    XCTAssertTrue(store.isConfigured)
}

func testApplyDiscoveryAndSaveRoundTrips() {
    let suiteName = "test.\(UUID().uuidString)"
    let defaults = UserDefaults(suiteName: suiteName)!

    let store = ConfigStore(defaults: defaults)
    store.baseURL = "https://hub.example.com"
    store.applyDiscovery(OIDCDiscoveryResult(
        clientID: "familyhub-ios",
        authorizationEndpoint: URL(string: "https://auth.example.com/authorize")!,
        tokenEndpoint: URL(string: "https://auth.example.com/token")!
    ))
    store.save()

    let reloaded = ConfigStore(defaults: defaults)
    XCTAssertEqual(reloaded.clientID, "familyhub-ios")
    XCTAssertTrue(reloaded.isConfigured)
}
```

- [ ] **Step 2: Run test to verify it fails**

Expected: compile error — `applyDiscovery` not found on `ConfigStore`.

- [ ] **Step 3: Add `applyDiscovery` to `ConfigStore.swift`**

```swift
func applyDiscovery(_ result: OIDCDiscoveryResult) {
    clientID = result.clientID
    authorizationEndpoint = result.authorizationEndpoint.absoluteString
    tokenEndpoint = result.tokenEndpoint.absoluteString
}
```

- [ ] **Step 4: Run the tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/ConfigStoreTests 2>&1 | grep -E "Test (Case|Suite)|error:|FAILED|passed|failed"
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Config/ConfigStore.swift ios/FamilyHub/FamilyHubTests/Auth/ConfigStoreTests.swift
git commit -m "feat(ios): add applyDiscovery to ConfigStore"
```

---

## Task 4: iOS — Simplify `ConfigurationFormView` and `SetupView`

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Settings/ConfigurationFormView.swift`
- Modify: `ios/FamilyHub/FamilyHub/Features/Settings/SetupView.swift`

Replace the 4-field form with a single URL field and an async "Connect" button that runs discovery. `SetupView` injects a `URLSessionOIDCDiscoveryService` in production; tests can inject a fake.

- [ ] **Step 1: Rewrite `ConfigurationFormView.swift`**

```swift
import SwiftUI

struct ConfigurationFormView: View {
    let configStore: ConfigStore
    let discoveryService: OIDCDiscoveryService
    let onSave: () -> Void

    @State private var baseURL: String
    @State private var isDiscovering = false
    @State private var discoveryError: String?

    @Environment(\.dismiss) private var dismiss

    init(configStore: ConfigStore, discoveryService: OIDCDiscoveryService, onSave: @escaping () -> Void) {
        self.configStore = configStore
        self.discoveryService = discoveryService
        self.onSave = onSave
        _baseURL = State(initialValue: configStore.baseURL)
    }

    var body: some View {
        List {
            Section("Server") {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Base URL")
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.textSecondary)
                    TextField("https://hub.example.com", text: $baseURL)
                        .font(.system(size: 14))
                        .foregroundStyle(Theme.textPrimary)
                        .keyboardType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }
                .padding(.vertical, 2)

                if let error = discoveryError {
                    Text(error)
                        .font(.system(size: 12))
                        .foregroundStyle(.red)
                }
            }
            .listRowBackground(Theme.surface)
        }
        .scrollContentBackground(.hidden)
        .listStyle(.insetGrouped)
        .background(Theme.background)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                if isDiscovering {
                    ProgressView()
                        .tint(Theme.accent)
                } else {
                    Button("Connect") {
                        Task { await connect() }
                    }
                    .foregroundStyle(Theme.accent)
                    .disabled(baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
                    .foregroundStyle(Theme.accent)
            }
        }
    }

    private func connect() async {
        discoveryError = nil
        isDiscovering = true
        defer { isDiscovering = false }

        guard let url = URL(string: baseURL.trimmingCharacters(in: .whitespaces)),
              url.scheme == "http" || url.scheme == "https" else {
            discoveryError = "Enter a valid http or https URL"
            return
        }

        do {
            let result = try await discoveryService.discover(baseURL: url)
            configStore.baseURL = url.absoluteString
            configStore.applyDiscovery(result)
            configStore.save()
            onSave()
            dismiss()
        } catch {
            discoveryError = error.localizedDescription
        }
    }
}
```

- [ ] **Step 2: Update `SetupView.swift` to inject the discovery service**

```swift
import SwiftUI

struct SetupView: View {
    @Environment(ConfigStore.self) private var configStore

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                VStack(spacing: 0) {
                    VStack(spacing: 8) {
                        Text("Welcome to Family Hub")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundStyle(Theme.textPrimary)
                        Text("Enter your server URL to get started.")
                            .font(.system(size: 15))
                            .foregroundStyle(Theme.textSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding(.top, 48)
                    .padding(.horizontal)

                    ConfigurationFormView(
                        configStore: configStore,
                        discoveryService: URLSessionOIDCDiscoveryService(),
                        onSave: {}
                    )
                }
            }
            .navigationTitle("Setup")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
        }
    }
}
```

Note: `onSave` is now a no-op on `SetupView` — `ConfigurationFormView` handles saving and dismissal internally. The callback is kept for callers that present `ConfigurationFormView` from a settings screen.

- [ ] **Step 3: Build the app to confirm it compiles**

In Xcode: Product → Build (⌘B), or:
```bash
cd ios/FamilyHub && xcodebuild build -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | grep -E "error:|BUILD SUCCEEDED|BUILD FAILED"
```

Expected: `BUILD SUCCEEDED`

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Settings/ConfigurationFormView.swift ios/FamilyHub/FamilyHub/Features/Settings/SetupView.swift
git commit -m "feat(ios): simplify setup to single base URL field with OIDC auto-discovery"
```

---

## Task 5: iOS — Run full test suite and fix any breakage

**Files:**
- Modify as needed (expect `OIDCConfigTests.swift` to still pass unchanged; check for any other callers of the old `ConfigurationFormView` init)

- [ ] **Step 1: Check for other callers of `ConfigurationFormView`**

```bash
grep -r "ConfigurationFormView" ios/FamilyHub --include="*.swift" -l
```

If any file other than `SetupView.swift` instantiates `ConfigurationFormView`, update those call sites to pass a `discoveryService` argument (inject `URLSessionOIDCDiscoveryService()` for production code).

- [ ] **Step 2: Run the full iOS test suite**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | grep -E "Test (Case|Suite)|error:|FAILED|passed|failed"
```

Expected: all tests pass.

- [ ] **Step 3: Run the server test suite**

```bash
cd server && make test
```

Expected: all tests pass.

- [ ] **Step 4: Commit if any fixes were needed**

```bash
git add -p
git commit -m "fix(ios): update remaining callers after ConfigurationFormView refactor"
```

---

## Self-Review

**Spec coverage:**
- ✅ User enters only base URL on first setup
- ✅ Server exposes `GET /api/client-config` (clientID + issuer, no auth)
- ✅ iOS uses OIDC discovery to get authorization + token endpoints
- ✅ Derived values persisted to UserDefaults (via `applyDiscovery` + `save`)
- ✅ Error states surfaced to user in the form
- ✅ `isConfigured` still gates app access (requires all 4 fields set)
- ✅ TDD throughout (failing test → implement → pass → commit)

**Placeholder check:** No TBD, TODO, or vague steps. All code blocks are complete.

**Type consistency:**
- `OIDCDiscoveryResult` defined in Task 2, used in Task 3 — ✅
- `OIDCDiscoveryService` protocol defined in Task 2, injected in Task 4 — ✅
- `URLSessionOIDCDiscoveryService` defined in Task 2, used in Task 4 — ✅
- `applyDiscovery(_:)` defined in Task 3, called in Task 4 — ✅
- `FakeURLSession` defined in test file in Task 2 — ✅
