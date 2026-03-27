# iOS Native Redesign — Design Spec

## Overview

Redesign the Family Hub iOS app to feel like a native iOS 26 application instead of a web app in an iPhone wrapper. The current app uses a custom dark-navy design system (`Theme.swift`) that overrides all native SwiftUI styling. This redesign strips that away entirely and leans into stock SwiftUI components, Apple's semantic color system, and iOS 26's Liquid Glass chrome.

## Goals

- App feels like a first-party Apple app (Reminders, Notes, Health)
- Automatically inherits iOS 26 Liquid Glass treatment
- Supports Dark Mode, Dynamic Type, and future OS updates for free
- Reduces custom UI code — less to maintain
- Restructures navigation from 5 tabs to 3, reflecting how features naturally group

## Constraints

- iOS 26+ deployment target (no backward compatibility needed)
- Keep the existing networking layer, auth flow, models, and view model patterns untouched
- Keep the same API contract with the Go backend — no backend changes required
- Color palette becomes fully Apple system colors (no custom branding)

## Tab Structure

Three tabs in a floating Liquid Glass tab bar:

| Tab | Icon | Content |
|-----|------|---------|
| Home | `house` | Today's chores, today's meals, weekly stats, profile access |
| Meals | `fork.knife` | Weekly meal planner (Plan) + recipe browser (Recipes) via segmented picker |
| Calendar | `calendar` | Month grid + agenda for selected day |

### Home Tab

The Home tab is the landing screen. It shows what matters today.

**Sections (top to bottom):**

1. **Chores** — overdue and due-today chores in a native grouped `List` section. Each row shows chore name, assignee, and due status. Swipe-to-complete action on each row. A "See All Chores" link at the bottom pushes a full chores list view onto the navigation stack.

2. **Today's Meals** — lunch and dinner for today in a grouped section. Tapping a meal with a linked recipe navigates to the recipe detail.

3. **This Week** — three stat cards in an `HStack` below the `List`, inside a `Section`. Each card is a simple `VStack` with the count and label. Uses native styling (no custom `StatCard` component).

**Navigation bar:**
- Large title: "Home"
- Trailing toolbar item: user avatar button that presents a profile sheet

### Chores List (Pushed from Home)

Full chores list pushed via `NavigationLink` from the "See All Chores" row. Not a tab — it's a child view in Home's `NavigationStack`.

- Segmented `Picker` at top: Pending / Completed
- Pending section groups chores by status: Overdue (red secondary label), Due Today (amber), Upcoming
- Completed section shows recently completed chores with green checkmark
- Swipe actions: swipe to complete (leading, green), swipe to view detail (trailing)
- Pull-to-refresh

### Meals Tab

Segmented `Picker` (`.pickerStyle(.segmented)`) at the top toggles between two views:

**Plan view:**
- Week navigation bar (previous/next buttons, centered date range label)
- Native `List` with one `Section` per day (Monday–Sunday)
- Each section has rows for meal types (Lunch, Dinner)
- Rows show meal name or "—" placeholder for empty slots
- Tap empty slot → presents recipe picker sheet
- Swipe to remove a meal assignment
- Pull-to-refresh

**Recipes view:**
- `.searchable()` modifier for filtering
- 2-column `LazyVGrid` displaying recipe cards
- Each card: image placeholder area, title, prep time, servings
- Tap pushes `RecipeDetailView` onto the navigation stack

### Recipe Detail (Pushed from Meals)

Pushed within the Meals tab's `NavigationStack`. Same content as current but with native styling:

- Header section: prep time, cook time, servings in a native grid
- Ingredients section: grouped by category in `List` sections
- Steps section: numbered list
- Cook Mode toggle (disables idle timer via `UIApplication.shared.isIdleTimerDisabled`)

### Calendar Tab

- Large title: "Calendar"
- Month navigation (previous/next)
- Calendar month grid with weekday headers
- Today highlighted with system accent circle
- Days with events show small dot indicators
- Tapping a day updates the agenda section below
- Agenda: native grouped `List` with color-coded leading sidebar indicators:
  - Red: overdue chore
  - Blue (system accent): calendar event
  - Green: meal
- Tapping an agenda item navigates to its detail (chore detail, recipe detail, etc.)

### Profile (Sheet from Home)

Presented as a `.sheet` from the avatar toolbar button on Home.

- User avatar, name, email
- "Edit Configuration" button (navigates to config form, warns about sign-out)
- "Sign Out" button (red, destructive role)
- Uses native `Form` / `List` styling

## Theming Strategy

### Delete entirely

The `Theme.swift` file and all references to it are removed. Every `Theme.x` color reference is replaced with the appropriate Apple semantic color.

### Color mapping

| Current (`Theme.x`) | Replacement |
|---------------------|-------------|
| `Theme.background` | `Color(.systemBackground)` or implicit from `List` |
| `Theme.surface` | `Color(.secondarySystemGroupedBackground)` |
| `Theme.surfaceElevated` | `Color(.tertiarySystemGroupedBackground)` |
| `Theme.textPrimary` | `Color.primary` |
| `Theme.textSecondary` | `Color.secondary` |
| `Theme.textMuted` | `Color(.tertiaryLabel)` |
| `Theme.accent` | `Color.accentColor` (system blue by default) |
| `Theme.statusRed` | `Color.red` |
| `Theme.statusAmber` | `Color.orange` |
| `Theme.statusGreen` | `Color.green` |
| `Theme.doneButtonBackground` | Remove — use `.tint(.green)` on button |
| `Theme.doneButtonBorder` | Remove — not needed with native buttons |
| `Theme.avatarFallback` | `Color(.tertiarySystemFill)` |

### Pattern replacements

| Current pattern | Native replacement |
|----------------|-------------------|
| `ZStack { Theme.background.ignoresSafeArea(); content }` | Remove wrapper. Use `List` or `ScrollView` which provide their own background. |
| Custom segment buttons with animation | `Picker("", selection: $x) { }.pickerStyle(.segmented)` |
| Manual `.toolbarBackground(Theme.background, for: .navigationBar)` | Remove. Let iOS 26 Liquid Glass handle toolbar chrome. |
| `.scrollContentBackground(.hidden)` + manual background | Remove. Let system provide scroll content background. |
| `.listRowBackground(Theme.surface)` | Remove or use `.listRowBackground(Color(.secondarySystemGroupedBackground))` only if needed. |

## Component Changes

### Files to delete

| File | Reason |
|------|--------|
| `DesignSystem/Theme.swift` | Replaced entirely by system colors |
| `Components/StatCard.swift` | Replaced by inline native layout |
| `Components/SectionCard.swift` | Replaced by native `Section` in `List` |
| `Components/StatusBadge.swift` | Replaced by colored secondary text labels |

### Files to keep (with simplification)

| File | Changes |
|------|---------|
| `Components/UserAvatar.swift` | Remove `Theme` color references, use `Color(.tertiarySystemFill)` for fallback background |

### Files to heavily rewrite

| File | Changes |
|------|---------|
| `Features/ContentView.swift` | 5-tab → 3-tab `TabView` using iOS 26 `Tab` API |
| `Features/Dashboard/DashboardView.swift` | Rename to `HomeView.swift`. Rebuild as native `List` with sections for chores, meals, stats. Add "See All Chores" navigation link. |
| `Features/Dashboard/DashboardViewModel.swift` | Rename to `HomeViewModel.swift`. May need to fetch chores + meals + stats in one load. |
| `Features/Chores/ChoresView.swift` | No longer a tab root. Becomes a pushed detail list within Home's `NavigationStack`. |
| `Features/Meals/MealsView.swift` | Add segmented picker for Plan/Recipes scope. Integrate recipe grid as an alternate view. |
| `Features/Recipes/RecipesView.swift` | Moves under Meals tab's navigation. Add `.searchable()`. Strip all Theme references. |
| `Features/Calendar/CalendarView.swift` | Rebuild with native patterns. Add color-coded agenda indicators. Strip Theme references. |

### Files with minor changes (strip Theme colors)

| File | Changes |
|------|---------|
| `Features/Chores/ChoreDetailView.swift` | Replace `Theme.x` with system colors |
| `Features/Recipes/RecipeDetailView.swift` | Replace `Theme.x` with system colors |
| `Features/Profile/ProfileView.swift` | Replace `Theme.x` with system colors |
| `Auth/LoginView.swift` | Replace `Theme.x` with system colors |
| `Features/Settings/SetupView.swift` | Replace `Theme.x` with system colors |
| `Features/Settings/ConfigurationFormView.swift` | Replace `Theme.x` with system colors |

## Directory Structure (after redesign)

```
FamilyHub/
├── Auth/                          (unchanged)
├── Components/
│   └── UserAvatar.swift           (simplified)
├── Config/                        (unchanged)
├── Features/
│   ├── Home/
│   │   ├── HomeView.swift         (renamed from DashboardView)
│   │   └── HomeViewModel.swift    (renamed from DashboardViewModel)
│   ├── Chores/
│   │   ├── ChoresListView.swift   (renamed from ChoresView, no longer tab root)
│   │   ├── ChoresViewModel.swift
│   │   └── ChoreDetailView.swift
│   ├── Meals/
│   │   ├── MealsView.swift        (rewritten with scope bar)
│   │   ├── MealsViewModel.swift
│   │   ├── RecipesView.swift      (moved here from Recipes/)
│   │   ├── RecipesViewModel.swift (moved here from Recipes/)
│   │   └── RecipeDetailView.swift (moved here from Recipes/)
│   ├── Calendar/
│   │   ├── CalendarView.swift     (rewritten)
│   │   └── CalendarViewModel.swift
│   ├── Profile/
│   │   └── ProfileView.swift
│   ├── Settings/
│   │   ├── SetupView.swift
│   │   └── ConfigurationFormView.swift
│   └── ContentView.swift          (3-tab rewrite)
├── Models/                        (unchanged)
├── Networking/                    (unchanged)
└── FamilyHubApp.swift             (unchanged)
```

The `DesignSystem/` directory is deleted entirely. The `Features/Recipes/` directory is merged into `Features/Meals/`.

## Native iOS Interactions

All standard iOS interactions are used where they fit — no specific feature requests, just "make it feel native":

- **Swipe actions** on list rows (complete chore, remove meal)
- **Pull-to-refresh** on all data views
- **`.searchable()`** on recipes
- **Large navigation titles** with automatic inline collapse on scroll
- **System animations** for transitions (push, sheet, dismiss)
- **Haptic feedback** via `UIImpactFeedbackGenerator` on completion actions
- **Liquid Glass** tab bar and navigation chrome (automatic with iOS 26 + native components)
- **Dynamic Type** support (automatic with system fonts and native components)
- **Dark Mode** support (automatic with semantic system colors)

## What Does NOT Change

- **View model pattern**: `@Observable` + `@MainActor` stays as-is
- **`ViewState<T>` enum**: good pattern, keep it
- **`APIClientProtocol`** and all networking code: untouched
- **Auth flow** (OIDC/PKCE, `AuthManager`, `KeychainStore`): untouched
- **All models** (`User`, `Chore`, `Recipe`, `MealPlan`, etc.): untouched
- **`ConfigStore`** and `OIDCDiscoveryService`: untouched
- **`FamilyHubApp.swift`** app entry point and state management: untouched
- **Test infrastructure**: test patterns stay the same, update view references as needed
