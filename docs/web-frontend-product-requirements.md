# Family Hub Web Frontend — Product Handoff

## Purpose

Build a web frontend that complements the Family Hub iOS app. It serves two
related uses:

1. A responsive household web app for members to manage shared family data.
2. A persistent wall display that gives the household useful, time-aware
   context at a glance.

The Go server remains a separate JSON API. The web frontend must not turn the Go
server into a web page server or require it to serve the app bundle.

## Current design artifact

The interactive, sample-data-only mockup is
[`web/prototypes/wall-display.html`](../web/prototypes/wall-display.html). Open
it in a browser and use the controls above the display to explore pairing,
time-of-day changes, light and dark appearance, household sections, offline
state, the touch drawing pad, and the app launcher. The phone beside the wall
display illustrates the iOS pairing flow. Mockup actions do not call the API
or save real household data.

The [Photo Frame Dashboard reference](https://github.com/wesbos/photo-frame-dashboard)
is inspiration for the ambient, contextual wall-display use case. Reuse the
product idea of a glanceable family display, not its implementation or visual
design by default.

Use the mockup to discuss layout and interaction. Use `DESIGN.md` as the visual
source of truth. Extend it with web-specific guidance before implementation.
Apply the UI UX Pro Max skill when designing or implementing web screens. Keep
the existing Warm Sage Cosier direction, readable type, clear states, and
minimum 44 px touch targets. Make cards and rows respond across their full
surface when the card represents one action.

## Product principles

### Household comes first

- The household is the data owner and the primary context in the product.
- Each person signs in as themselves. Shared meals, shopping, calendar,
  scratchpad, chores, and inventory belong to the household and are visible to
  its members according to their role.
- A display belongs to a household. It must not depend on the personal login
  session of the person who paired it.
- Treat household membership and display pairing as separate concepts. A
  household member can leave without accidentally transferring their personal
  identity to a wall device.
- The current API has one shared data scope. Before multi-household use,
  introduce an explicit household boundary in the data model and authorize each
  request against membership. Do not present per-user sign-in as household
  isolation.

### Separate clients, shared API

- Keep the web client deployable on its own origin and independent from the Go
  server process.
- Use the existing JSON API where its data model fits. Add or change API
  contracts deliberately when a web flow needs a capability that the API does
  not provide.
- Keep OIDC credentials and user sessions distinct from a wall display's
  device credential. Never place a personal long-lived API token in a public
  display URL or QR code.
- Choose the web stack and implementation patterns by inspecting the user's
  existing repositories. Reuse their conventions and components where they fit.
  Do not introduce a stack or pattern without precedent in those repositories.

## Main experiences

### 1. Household web app

Provide these top-level areas:

| Area | Requirements |
|---|---|
| Today | Shared day overview with meals, chores, calendar context, and weather. |
| Calendar | Weekly grid. Provide previous week, next week, and Today controls. |
| Chores | See due and overdue tasks. Let eligible household members complete tasks. |
| Food | Separate Meal Plan and Recipes destinations. The meal plan covers breakfast, lunch, and dinner, Friday through Thursday. Tapping a meal opens its edit or swap flow. |
| Household | Provide Shopping, Inventory, and Scratchpad destinations. Group inventory by its named areas. |

The shopping list must support adding an item and marking it picked up, with a
clear undo path where the API supports it. The scratchpad is a touch drawing
surface with a small palette of colors. Do not add free-text notes to this
scratchpad.

### 2. Wall display

- Make the display useful from across a room. Prioritize time, date, household
  agenda, and the next relevant action.
- Keep the household's key information together in a glanceable layout.
- Provide an app launcher for Shopping, Scratchpad, Calendar, Recipes, Meal
  Plan, and Inventory.
- Let the household choose which sections appear on the display.
- Make the whole card tappable when it opens that card. Keep controls inside a
  card independently operable and large enough for touch.
- Follow the display's system appearance by default. Support light and dark
  appearance, with an explicit override only if it remains easy to understand.
- Do not show a household label or household name in the bottom edge/footer.
- Do not include “Returns Home after 5 idle minutes” in pop-ups or other user
  facing copy. Any automatic return to the overview must not interrupt an
  active edit or drawing gesture.

### Meals on the display

- Represent breakfast, lunch, and dinner as one day's meal plan. Do not show
  separate “current meal” and “next meal” cards that repeat the same plan.
- Make the next upcoming meal clear using the household's local time.
- As the day moves toward evening, change the meal area to a clearly labelled
  preview of tomorrow's breakfast, lunch, and dinner.
- Tapping a meal opens its edit/swap flow. Empty slots offer an add action.
- A swap must stay within the selected date and meal type. Show the household a
  sensible alternative before saving; never silently change another meal slot.
- Define exact time boundaries and fallback behavior during implementation.
  Prefer household-local time and sensible defaults over hard-coded display
  text.

### Weather

- Show current conditions and an hourly forecast on the display.
- Tapping the weather opens a detail pop-up with hourly conditions and a
  seven-day forecast, including useful high/low, precipitation, and condition
  details when available.
- Make unavailable or stale data clear. Do not imply that a missing forecast is
  a zero-rain or clear-sky forecast.

### Calendar

- Show a weekly grid in the Calendar app, with meals, events, and chores using
  distinct labels or symbols in addition to color.
- Previous and next move by exactly one week. Today returns to the current
  week.
- Let users open an item for details. Event creation or editing is outside the
  first web scope unless the existing API already supports the intended flow.

### Scratchpad

- Support touch, mouse, and stylus drawing.
- Offer a small set of colors, undo, and clear.
- Persist the drawing as shared household data and show save state.
- On a failed save, preserve the drawing and provide clear retry copy. Replace
  “Could not save. Nothing changed. Try again.” with calmer language such as
  “We couldn’t save that. Please try again.”

## Pairing and access

The mockup demonstrates the desired setup shape: an iOS user scans a short-lived
QR code, confirms the target household and display, then approves the pairing.
This flow is a product direction, not a production security implementation.

For production:

- Generate a short-lived, single-use pairing challenge on the server.
- Require an authenticated household member to approve it in the iOS app.
- Issue a revocable, least-privilege display credential scoped to the selected
  household and its allowed display data.
- Provide a way to name, inspect, and revoke paired displays from the iOS app.
- Do not show personal account credentials or a reusable user API token in the
  QR code.
- Keep the iOS household copy accurate. Remove “Household-owned / Approved by
  Alex. Independent of Alex’s session.” from the iOS app; describe the device
  as paired with the household instead.
- The latest released iOS version is 1.3. Ship any iOS changes for pairing or
  display management as version 1.4.

The current backend does not model paired display devices or household
membership boundaries. Implement those API and data changes before calling a
production display household-owned or exposing it to an untrusted network.

## Error, loading, and offline behavior

- Give each screen clear loading, empty, stale/unavailable, saving, and failed
  states.
- Keep user changes visible until the server confirms them. On failure, explain
  what the user can do next. Avoid blaming or implying that nothing changed
  when a local draft remains visible.
- The wall display should make loss of connection visible without replacing
  the household content with a blank page. Clearly distinguish cached data from
  current data.
- Do not expose implementation details such as token type, internal server
  errors, or idle timer rules in user-facing pop-ups.

## Out of scope for the first delivery

- Replacing or rewriting the Go API server.
- Server-side rendering or serving the web bundle from the Go process.
- New household social or messaging features.
- Turning the static mockup's simulated state into production behavior without
  an API and household-access review.
- A public, unauthenticated display endpoint.

## Delivery sequence

1. Extend `DESIGN.md` with responsive web and wall-display guidance. Revisit the
   mockup against those decisions.
2. Build the member web app with household-scoped authorization and existing
   API capabilities.
3. Define and implement household membership and device-pairing API contracts.
4. Add iOS 1.4 pairing approval, device management, and corrected copy.
5. Build the kiosk display on the paired-device credential. Validate privacy,
   reconnect behavior, revocation, and touch use on the target display.

Do not skip the household and device authorization work by using a member's
web or iOS session as the permanent wall-display credential.

## Acceptance criteria

- The web app runs and deploys independently from the Go API server.
- Household members see the same shared data, subject to household membership
  and role authorization.
- The weekly calendar navigates backward and forward by one week and returns to
  the current week with Today.
- The meal plan supports all three meal types and does not duplicate current
  and next meal content on the display.
- The display previews tomorrow's three meals toward the end of the day.
- Weather details include hourly and seven-day forecasts when the API provides
  them.
- The launcher opens Shopping, Scratchpad, Calendar, Recipes, Meal Plan, and
  Inventory.
- Inventory is grouped by area. Shopping supports add and purchase actions.
- Scratchpad drawing works with touch and can be saved, retried, undone, and
  cleared without a free-text field.
- Light and dark appearances follow the display's system setting by default.
- Interactive cards and controls meet the 44 px minimum target in `DESIGN.md`.
- Production display pairing uses a revocable household-scoped device
  credential and can be revoked from iOS.
- iOS changes are versioned as 1.4.
