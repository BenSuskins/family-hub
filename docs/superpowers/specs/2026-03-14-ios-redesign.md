# iOS App Redesign — Design Spec

**Date:** 2026-03-14
**Status:** Approved

---

## Context

The Family Hub iOS app is functional but visually inconsistent and violates several iOS design principles. The web UI (Suskins Hub) has a polished dark-navy design that serves as the reference aesthetic. This redesign brings the iOS app to the same visual standard while fixing key UX gaps.

---

## Design Direction

**Style:** Flat Dark (B1) — no gradients, high contrast, muted chrome
**Reference:** Suskins Hub web UI
**Colour system:**

| Token | Hex | Usage |
|---|---|---|
| `background` | `#0f172a` | App background, tab bar |
| `surface` | `#1e293b` | Cards, section backgrounds |
| `surfaceElevated` | `#334155` | Segment control selected-thumb background |
| `borderDivider` | `#0f172a` | Row dividers inside cards — intentionally matches `background`, creating a dark-gap slot effect between `surface` rows rather than a hairline |
| `textPrimary` | `#f1f5f9` | Titles, body |
| `textSecondary` | `#94a3b8` | Assignee names, subtitles |
| `textMuted` | `#475569` | Section labels, tab bar inactive |
| `accent` | `#60a5fa` | Active tab, selected segment, completed chore count in leaderboard, links |
| `statusRed` | `#ef4444` | Overdue badge + date text |
| `statusAmber` | `#f59e0b` | Today / Due Soon badge + date text |
| `statusGreen` | `#4ade80` | Done button text and checkmark |
| `doneButtonBg` | `#1e3a2f` | Done button fill |
| `doneButtonBorder` | `#16a34a33` | Done button stroke |
| `avatarFallback` | `#6366f1` | Background fill for avatars with no photo (all users) |

**Typography:** SF Pro (system default). No custom fonts.
**Icons:** SF Symbols throughout. No emoji in UI chrome.
**Corners:** Cards `cornerRadius: 14`, stat cards `12`, avatar `circle`, Done button `8`
**Spacing base:** 14pt horizontal margins, 8pt gap between sections

---

## Shared Components

### `UserAvatar`

Circle view. If the user has a photo URL, loads it async with an initials placeholder while loading. If no photo, shows initials on `avatarFallback` background.

| Size | Context |
|---|---|
| 32pt | Chore rows (Dashboard, Chores list) |
| 24pt | Leaderboard rows |

### `StatusBadge`

Pill label. Variants:

| Variant | Label | Text colour | Background |
|---|---|---|---|
| `.overdue` | "Overdue" | `statusRed` | `statusRed` at 13% opacity |
| `.dueToday` | "Today" | `statusAmber` | `statusAmber` at 13% opacity |
| `.dueSoon` | "Due Soon" | `statusAmber` | `statusAmber` at 13% opacity |

### `SectionCard`

`surface` background, `cornerRadius: 14`. Header row: SF Symbol icon + section title in `textPrimary`, separated from content rows by a `borderDivider` line. Content rows separated by `borderDivider` horizontal lines.

### `StatCard`

`surface` background, `cornerRadius: 12`. Label (small caps, `textMuted`), large number (`textPrimary`), subtitle below.

---

## Screens

### 1. Home (Dashboard)

Large title "Overview" + date subtitle. User avatar (`UserAvatar`, 32pt) top-right — tapping opens the Profile sheet.

**Stat row** — three equal `StatCard`s. The large number always uses `textPrimary`.
- Chores: active chore count; subtitle "N overdue" in `statusRed` (or hidden if zero)
- Events: upcoming event count; subtitle "Next 7 days" in `textMuted`
- Meals: meals-planned count; subtitle "of 21 planned" in `textMuted`

**Chores Due section** — `SectionCard` with `checkmark.circle` icon (teal). Rows: `UserAvatar` (32pt), chore name, assignee name (`textSecondary`), `StatusBadge`, date, Done button. Done button: `doneButtonBg` fill, `doneButtonBorder` stroke, `statusGreen` checkmark + "Done" label. Tapping Done calls mark-complete API with optimistic local update; failure shows an inline error row at the bottom of the card (not an alert).

**Today's Meals section** — `SectionCard` with `flame` icon. Shows Lunch and Dinner rows only (Breakfast excluded from Dashboard for brevity — Meals tab shows all three). Meal name, or "— not planned —" in `textMuted` if empty.

**Leaderboard section** — `SectionCard` with `trophy` icon. Week / Month pill-segment toggle. Columns: rank #, `UserAvatar` (24pt) + name, completed count in `accent`, pending count in `textMuted`.

Pull-to-refresh on the whole scroll view. Load errors shown as an inline `Text` at the top of the scroll view (no alert).

---

### 2. Chores

Large title "Chores". Filter SF Symbol in nav bar trailing (visible but non-functional; reserved for future use).

**Segment control** — Pending / Completed. Container background: `surface`. Selected segment thumb background: `surfaceElevated`. Selected label: `textPrimary`. Unselected label: `textMuted`.

**Pending tab** — rows grouped into two sections:
- **Overdue** — `statusRed` section label; rows use red date text
- **Due Soon** — `statusAmber` section label; rows use amber or `textMuted` date text

Each row: `UserAvatar` (32pt), chore name (`textPrimary`), assignee + due date (`textSecondary`), trailing `chevron.right` in `textMuted`.

**Completed tab** — flat list, same row style but no `StatusBadge`. The `UserAvatar` (32pt) is replaced entirely by a `checkmark.circle.fill` SF Symbol (32pt, `statusGreen`) in the leading position. The subtitle row retains the same format as Pending (assignee + due date); the due date shown is the original due date, not a completion timestamp.

Tapping a row pushes `ChoreDetailView`. Pull-to-refresh on list. Load errors shown inline below the segment control.

**ChoreDetailView** — `LabeledContent` rows: name, description, recurrence, assignee, due date. "Mark Complete" full-width button (`doneButtonBg`). Button shows `ProgressView` while request is in flight. On success, pops back to the list with an optimistic update already applied. On failure, shows inline error text below the button (no alert).

---

### 3. Meals

Large title "Meals". Prev / Next week chevron buttons in toolbar.

Week header row showing Mon–Sun dates.

Seven collapsible day sections. Each header: weekday + date. Rows: Breakfast / Lunch / Dinner. Meal name, or "—" in `textMuted` if not planned. Read-only.

Pull-to-refresh. Load errors shown inline below the week header.

---

### 4. Recipes

Large title "Recipes". `.searchable` modifier adds search bar below nav title; filters by recipe title client-side.

Two-column `LazyVGrid`. Each card: `surface` bg, `cornerRadius: 14`, `surfaceElevated` image placeholder, recipe title, prep + cook time, servings count.

Tapping a card pushes `RecipeDetailView`. Pull-to-refresh on grid. Load errors shown inline above the grid.

**RecipeDetailView** — title, metadata row (servings / prep / cook), ingredient groups as `List` sections, steps as numbered list. "Cook Mode" button prevents the screen from sleeping while the detail view is visible; the screen sleep lock is released automatically when the view disappears.

---

### 5. Calendar

Large title "Calendar". Month + year displayed in nav. Prev / Next month in toolbar.

7-column day grid. Selected day: `accent` filled circle. Days with chores: small `statusAmber` dot below the number.

Below grid: agenda list for selected day showing chore rows (chore name, assignee, `StatusBadge`). Empty state: `ContentUnavailableView` with `calendar` SF Symbol and label "No chores on this day".

Calendar shows chores only. Event/iCal display is out of scope for this redesign.

Pull-to-refresh reloads the current month. Load errors shown inline above the day grid.

---

### 6. Profile (new screen)

Accessed via the `UserAvatar` button on the Dashboard nav bar. Presented as a `.sheet`.

Single-level sheet — no navigation stack.

**Account section:** display name, initials avatar, email address (all read-only, sourced from the existing OIDC session claims).

**Actions section:** "Sign Out" button, `.destructive` role, calls `AuthManager.logout()` and dismisses the sheet, returning to `LoginView`.

---

## UX Fixes in Scope

| Fix | Detail |
|---|---|
| Pull-to-refresh | Replace all `toolbar { Button("Refresh") }` with `.refreshable {}` on each scroll view / list |
| Inline errors | Remove all `.alert` error presentation; show `Text` with `statusRed` colour inline on each screen |
| Logout | Profile sheet Sign Out button wired to `AuthManager.logout()` |

---

## Out of Scope

- Creating or editing meals, chores, or recipes
- Push notifications
- Swipe-to-complete chore rows
- Admin screen on iOS
- Offline caching
- Event / iCal display on Calendar

---

## Colour Definitions

Define all tokens in `Assets.xcassets` as named color sets. All appearances (light, dark, any) should point to the dark values since the app is dark-mode only. Tokens must be centralised and referenced by name rather than hard-coded hex values throughout the codebase.

---

## Verification

1. Build and run on iPhone 15 Simulator (iOS 17+)
2. **Dashboard:** stat cards show correct counts; chore rows display avatar, assignee name, `StatusBadge`, and Done button; Today's Meals shows Lunch + Dinner only; Leaderboard shows rank, avatar, completed (blue), pending
3. **Chores tab:** Pending/Completed segment switches correctly; Pending tab groups rows into Overdue and Due Soon; tapping a row opens detail; Mark Complete succeeds and pops back with row removed
4. **Recipes:** search bar filters the grid by title as you type
5. **Calendar:** selecting a day updates the agenda list; days with chores show amber dot
6. **Pull-to-refresh:** drag down on each of the five tabs triggers a reload (no toolbar Refresh button visible)
7. **Inline errors:** with network disconnected, trigger a Dashboard refresh — error text appears in the scroll view without a system alert dialog
8. **Profile / logout:** tap avatar on Dashboard → sheet opens with name and email; tap Sign Out → sheet dismisses and `LoginView` is presented
