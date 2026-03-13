# iOS App Design — Family Hub

**Date:** 2026-03-12
**Status:** Approved

## Overview

A native UIKit iOS app for Family Hub, living in `ios/` inside the monorepo. Connects to the existing Go backend via a REST API. Covers all core features: Dashboard, Chores, Meals, Recipes, and Calendar.

## Approach

Feature-by-feature, end-to-end. Each iteration delivers a working backend endpoint and a working iOS screen. Order: Dashboard → Chores → Meals → Recipes → Calendar.

## Architecture

```
ios/
├── FamilyHub.xcodeproj
└── FamilyHub/
    ├── App/               # AppDelegate, SceneDelegate, entry point
    ├── Auth/              # OIDC/PKCE flow, token storage (Keychain)
    ├── Networking/        # APIClient, request builder, response decoder
    ├── Models/            # Codable structs mirroring Go models
    ├── Features/
    │   ├── Dashboard/     # ViewController + ViewModel
    │   ├── Chores/
    │   ├── Meals/
    │   ├── Recipes/
    │   └── Calendar/
    └── UI/                # Shared components, extensions
```

Navigation: `UITabBarController` with five tabs — Dashboard, Chores, Meals, Recipes, Calendar.

## Authentication

- OIDC + OAuth2 + PKCE via Authentik
- `ASWebAuthenticationSession` opens Authentik's authorize endpoint
- Custom URL scheme `familyhub://` handles the redirect callback (to be registered in Xcode)
- Auth code exchanged for access + refresh tokens at Authentik's token endpoint
- Tokens stored in Keychain
- `APIClient` attaches `Authorization: Bearer <access_token>` to all requests
- On 401: attempt silent refresh; on failure, re-prompt login
- Backend unchanged — iOS is a new OIDC client registered in Authentik

## Data Flow

```
ViewController → ViewModel.load() → APIClient.request() → URLSession
                                                         ↓
ViewController ← ViewModel publishes state ← Decoded Codable model
```

ViewModel exposes a state enum:

```swift
enum ViewState<T> {
    case idle
    case loading
    case loaded(T)
    case failed(APIError)
}
```

State changes delivered via closure callback on ViewModel (no Combine). `APIError` is a typed enum: `.network`, `.unauthorized`, `.decoding`, `.server(Int)`.

## Features & Backend Scope

| Feature   | Existing API                          | New endpoints needed                        | Write in v1? |
|-----------|---------------------------------------|---------------------------------------------|--------------|
| Dashboard | `GET /api/dashboard` (counts only)    | Extend to return today's + overdue chore list | No         |
| Chores    | `GET /api/chores`, `GET /api/chores/{id}` | `POST /api/chores/{id}/complete`        | Yes (complete only) |
| Meals     | None                                  | `GET /api/meals?week=YYYY-MM-DD`            | No           |
| Recipes   | None                                  | `GET /api/recipes`, `GET /api/recipes/{id}` | No           |
| Calendar  | None                                  | `GET /api/calendar?month=YYYY-MM`           | No           |

Create/edit for meals, recipes, and calendar stays in the web UI for v1.

## Screens

### Tab Bar
Five tabs: Dashboard · Chores · Meals · Recipes · Calendar.

### Dashboard
Two stat cards (Due Today, Overdue). List of today's and overdue chores below.

### Chores
Grouped list (Pending / Completed) with filter. Tap → Chore Detail with Mark Complete button.

### Meals
Weekly view, navigable by week. Each day shows Breakfast / Lunch / Dinner entries.

### Recipes
Two-column grid with image thumbnails. Tap → Recipe Detail with ingredients and steps. Cook Mode available.

### Calendar
Monthly grid with event dots. Tap a day → agenda list below showing events and chores for that day, colour-coded by source.

## Testing

- `APIClient` accepts a `URLSession` — `URLProtocol` stub used in tests (fake, not mock)
- ViewModels tested against a fake `APIClient` conforming to a protocol
- No UI tests in v1
