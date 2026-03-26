# iOS User-Configurable Settings

**Date:** 2026-03-17
**Status:** Approved

## Problem

The iOS app currently reads all configuration (server URL, OIDC client ID, auth endpoints) from a static `Config.plist` baked into the app bundle at build time. This requires developers to manually copy and edit the plist before building, and makes the app unusable for anyone who cannot modify the source. There is no way to change settings at runtime without rebuilding.

## Goal

Replace `Config.plist` with a runtime settings store backed by `UserDefaults`. Users configure the app on first launch via an onboarding screen and can edit settings later from the profile view.

## Scope

Four fields become user-configurable:

| Field | UserDefaults key |
|---|---|
| Server base URL | `baseURL` |
| OIDC client ID | `clientID` |
| Authorization endpoint | `authorizationEndpoint` |
| Token endpoint | `tokenEndpoint` |

`Config.plist` and `Config.example.plist` are deleted.

---

## Architecture

### Data Flow

```
FamilyHubApp
├── ConfigStore (@Observable, @MainActor)
│   └── UserDefaults suite: "uk.co.suskins.familyhub.config"
├── AuthManager (@Observable, @MainActor)
│   └── caches a copy of OIDCConfig at login time (does not hold a reference to ConfigStore)
└── Root view decision (synchronous — no loading state needed):
    ├── !ConfigStore.isConfigured → SetupView
    ├── !AuthManager.isAuthenticated → LoginView
    └── else → ContentView
```

`ConfigStore` and `AuthManager` are both injected into the SwiftUI environment at app startup. The root view reads both to decide which screen to show.

**No loading state is needed.** `ConfigStore` reads `UserDefaults` synchronously in `init`. `AuthManager` reads from the Keychain synchronously in `init`. Both are fully settled before the first view renders.

### `ConfigStore`

A new `@Observable @MainActor` class. Loads from and persists to a named `UserDefaults` suite so test suites can inject an isolated instance.

```swift
@Observable @MainActor
final class ConfigStore {
    var baseURL: String
    var clientID: String
    var authorizationEndpoint: String
    var tokenEndpoint: String

    var isConfigured: Bool {
        ![baseURL, clientID, authorizationEndpoint, tokenEndpoint]
            .contains(where: \.isEmpty)
    }

    // Default uses a named suite. Force-unwrap is intentional:
    // the suite name is a compile-time constant and nil would indicate
    // a programming error, not a runtime condition.
    init(defaults: UserDefaults = UserDefaults(suiteName: "uk.co.suskins.familyhub.config")!)

    // Writes the current property values to UserDefaults.
    // Does NOT re-assign the properties — avoids spurious @Observable change notifications.
    // UserDefaults writes are synchronous in memory; disk flush is handled by the OS.
    // synchronize() is not called (deprecated). Disk-flush failures are silently ignored;
    // values remain correct in memory for the current session.
    func save()
}
```

### `OIDCConfig`

The existing `fromPlist()` factory is replaced with:

```swift
static func from(configStore: ConfigStore) throws(ConfigurationError) -> OIDCConfig
```

**Error type:**

```swift
enum ConfigurationError: LocalizedError {
    case emptyField(String)          // field name
    case invalidURL(String, String)  // field name, raw value

    var errorDescription: String? { ... }
}
```

**URL validation rules:**
- All four fields must be non-empty (throws `.emptyField`)
- `authorizationEndpoint` and `tokenEndpoint` must parse as a `URL` with scheme `http` or `https` (throws `.invalidURL`)
- `baseURL` must parse as a `URL` with scheme `http` or `https` (throws `.invalidURL`)
- `clientID` is validated only for non-empty; no URL parsing

**`AuthManager` caches a copy** of the resulting `OIDCConfig` struct at login time. It does not hold a reference to `ConfigStore`. There is no stale-config risk during a session.

**`AuthManager` error property:**

```swift
var loginError: String?   // set to error.localizedDescription on failure
```

`loginError` is cleared to `nil` at the start of each login attempt. `LoginView` renders it inline when non-nil.

---

## UI

### `ConfigurationFormView`

A form component that edits configuration fields. It maintains `@State` copies of the four field values, initialised from the current `ConfigStore` values when the view appears.

```swift
struct ConfigurationFormView: View {
    let configStore: ConfigStore   // read-only reference; edits are local until saved
    let onSave: () -> Void

    @State private var baseURL: String
    @State private var clientID: String
    @State private var authorizationEndpoint: String
    @State private var tokenEndpoint: String

    // On save: writes @State values back to configStore properties, then calls onSave()
    // On cancel/dismiss: @State is discarded — configStore is never mutated
}
```

**Editing is isolated.** The four `@State` fields are local copies. `configStore` is only written to when the user explicitly taps Save. Dismissing the view (cancel, swipe-down) discards the local state without touching `configStore`. This prevents dirty state.

The form accepts any string without validation. Validation (URL format, scheme) is the sole responsibility of `OIDCConfig.from(configStore:)` at login time. This is a deliberate trade-off: invalid URLs entered at setup are caught on the first login attempt, where `LoginView` displays the error inline. This avoids duplicating validation logic in the UI.

### `SetupView`

Shown when `!configStore.isConfigured`. Acts as the app root until setup is complete.

- Hosts `ConfigurationFormView` with `onSave: { configStore.save() }`
- No back navigation (this is the entry point)
- On save, `configStore.isConfigured` becomes true, and the root view transitions to `LoginView`

### `ProfileView` — Settings Entry

An "Edit Configuration" navigation row is added. Tapping it shows a **confirmation alert** first:

> "Editing your configuration will sign you out. Continue?"

The form is presented **only after the user confirms**. On cancellation, nothing changes and the form is never shown.

After confirmation, `ConfigurationFormView` is presented with:

```swift
onSave: {
    configStore.save()
    authManager.signOut()
}
```

On save, `authManager.signOut()` clears the cached token and sets `isAuthenticated = false`. The root view observes this and replaces the entire view hierarchy with `LoginView`. This is the same mechanism as a normal sign-out — no manual `NavigationStack` path manipulation is required.

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| App launched, no config stored | `SetupView` shown — no crash |
| `OIDCConfig.from(configStore:)` throws | `AuthManager` sets `loginError`; `LoginView` displays it inline |
| User cancels config edit (swipe-down or back) | `@State` discarded; `configStore` unchanged |
| User taps "Edit Configuration" in profile | Confirmation alert shown; form only opens on confirm |
| User confirms config edit, taps Save | Save → sign out → root view replaces hierarchy with `LoginView` |
| UserDefaults disk-flush fails | Values correct in memory; no user-visible error (acceptable) |

---

## Testing

- **`ConfigStoreTests`**
  - `isConfigured` returns false when any field is empty; true when all are non-empty
  - `save()` / `init()` round-trip: set values, call `save()`, construct a new instance with the same suite, confirm values are loaded
  - `save()` does not mutate properties (observable change count stays at zero after save)
  - Use a unique isolated suite per test (e.g. `"test.\(UUID().uuidString)"`) — never `.standard`

- **`OIDCConfigTests`**
  - `from(configStore:)` with all valid `http` and `https` URLs → succeeds
  - `from(configStore:)` with one empty field → throws `.emptyField`
  - `from(configStore:)` with a non-URL string → throws `.invalidURL`
  - `from(configStore:)` with a `file://` URL → throws `.invalidURL` (scheme validation)
  - `from(configStore:)` with a relative path string → throws `.invalidURL`

- **`AuthManager` tests** — unchanged; they construct `OIDCConfig` directly

- **`ConfigurationFormView` cancel path** — no automated tests; covered by manual smoke test: open form, edit a field, dismiss without saving, confirm original values are unchanged

---

## Migration

No data migration needed. On first launch after update, `ConfigStore` finds no stored values, `isConfigured` returns false, and `SetupView` is shown. The user re-enters their configuration once.

---

## Out of Scope

- Build-scheme-based configuration (no xcconfig changes)
- Secure storage of config values in Keychain (not needed; these are not secrets)
- Feature flags or environment switching beyond the four fields
