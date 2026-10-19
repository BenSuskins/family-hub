# Family Hub — Endpoint Reference

All routes wired in `server/internal/server/server.go`.

**Global middleware** (all routes): logger, recoverer, gzip compression, security headers,
per-IP rate limit (100 req/min), family-name injector.

**Conventions used below:**

```bash
export BASE_URL="http://localhost:8080"
export API_TOKEN="<token from /api/auth/exchange or POST /api/tokens>"
export SESSION="<session cookie value, for web routes>"
```

---

## Public routes (no auth)

### `GET /health`
- **Usecase:** Liveness probe. Returns `200 ok`.
- **Callers:** Docker healthcheck, uptime monitoring, load balancers.
- **Security:** None. Exempt from auth.

```bash
curl -s $BASE_URL/health
```

### `GET /static/*`
- **Usecase:** Serves Tailwind CSS, JS, images from `server/static/`.
- **Callers:** Browser (via templ layouts).
- **Security:** None. `http.FileServer` with `StripPrefix`.

```bash
curl -sI $BASE_URL/static/app.css
```

### `GET /login`
- **Usecase:** Renders login page that redirects to the OIDC provider.
- **Callers:** Browser (any unauthenticated request redirects here).
- **Security:** Rate limited 10/min per IP.

```bash
curl -s $BASE_URL/login
```

### `GET /auth/callback`
- **Usecase:** OIDC redirect target; exchanges code, creates session cookie.
- **Callers:** OIDC provider redirect after successful login.
- **Security:** Rate limited 10/min per IP. State param validated.

```bash
# Triggered by browser redirect, not manually
curl -s "$BASE_URL/auth/callback?code=XYZ&state=ABC"
```

### `GET /logout`
- **Usecase:** Clears session cookie, redirects to `/login`.
- **Callers:** Browser (user clicks logout).
- **Security:** None beyond cookie clear.

```bash
curl -s -b "session=$SESSION" $BASE_URL/logout
```

### `GET /api/client-config`
- **Usecase:** Returns OIDC `clientID` + `issuer` so mobile clients can start the auth flow.
- **Callers:** iOS app on first launch.
- **Security:** None (public, non-sensitive).

```bash
curl -s $BASE_URL/api/client-config | jq
```

---

## Mobile token exchange

### `POST /api/auth/exchange`
- **Usecase:** Swap an OIDC bearer (from iOS app's OIDC login) for a long-lived API token. Creates user on first login.
- **Callers:** iOS app post-OIDC login.
- **Security:** Rate limited 10/min per IP. Validates bearer against OIDC userinfo endpoint. Revokes prior "iOS App" tokens for the user.

```bash
curl -s -X POST $BASE_URL/api/auth/exchange \
  -H "Authorization: Bearer <OIDC_ACCESS_TOKEN>" | jq
```

---

## Authenticated surface (session cookie OR Bearer token)

Every route below resolves a user via either the session cookie (browser) or
`Authorization: Bearer $API_TOKEN` (mobile/API). A single `RequireUser`
middleware handles both. The `curl` examples use Bearer for brevity; swap in
`-b "session=$SESSION"` for the web equivalent.

### JSON API routes

### `GET /api/commute/trains`

- **Use case:** Return upcoming trains between Kelvedon (`KEL`) and London Liverpool Street (`LST`).
- **Callers:** The iPad display and other authenticated clients.
- **Security:** Session cookie or Bearer API token. The provider token stays on the server.
- **Parameters:** `direction=to_work` (default) or `to_home`; `count=1..10` (default: `5`).
- **Window:** The provider returns up to the requested count within the next two hours. This is not an all-day timetable.
- **Order:** Scheduled departure order, including cancelled trains. Return `trains: []` when no trains are available.
- **Times:** Train times use `HH:mm` in `Europe/London`. Arrival means arrival at the commute destination, including intermediate stops.
- **Platform:** The departure platform. Unknown platforms and times are `null`.
- **Status:** `on_time`, `delayed`, `cancelled`, or `unknown`. A cancelled destination stop means a cancelled commute.
- **Estimates:** An "On time" estimate uses the scheduled time. An unspecified estimate is `null`. Cancelled trains have null estimates.
- **Freshness:** Successful results use a 30-second cache for each direction and count. Expired results are never returned.
- **Timestamps:** `generatedAt` comes from the provider. `fetchedAt` records the server fetch. Both use RFC 3339.
- **Errors:** `400` for invalid parameters; `503` for an absent provider token; `502` for provider failures; `504` for provider timeouts.
- **Configuration:** Set `HUXLEY_ACCESS_TOKEN` to the National Rail token. `HUXLEY_BASE_URL` defaults to `https://huxley2.azurewebsites.net`.

```bash
curl -s "$BASE_URL/api/commute/trains?direction=to_work&count=5" \
  -H "Authorization: Bearer $API_TOKEN" | jq

curl -s "$BASE_URL/api/commute/trains?direction=to_home&count=3" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

Example response:

```json
{
  "direction": "to_work",
  "origin": {"code": "KEL", "name": "Kelvedon"},
  "destination": {"code": "LST", "name": "London Liverpool Street"},
  "generatedAt": "2026-10-18T08:00:00+01:00",
  "fetchedAt": "2026-10-18T07:00:01Z",
  "timezone": "Europe/London",
  "trains": [{
    "serviceId": "example-service",
    "operator": "Greater Anglia",
    "platform": "1",
    "status": "on_time",
    "scheduledDeparture": "08:15",
    "expectedDeparture": "08:15",
    "scheduledArrival": "09:10",
    "expectedArrival": "09:10",
    "cancellationReason": null,
    "delayReason": null
  }]
}
```

The server uses [Huxley expanded departures](https://huxley2.azurewebsites.net/) to read destination arrival times.
The default instance is a public demo. Its availability depends on the provider.

To check the live provider outside CI, set the token in the environment and run:

```bash
cd server
COMMUTE_LIVE_TEST=true go test ./internal/services -run TestHuxleyLive -count=1 -v
```

### `GET /api/me`
- **Usecase:** Current user profile for the token holder.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/me -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/profile/avatar`
- **Usecase:** Upload a custom avatar for the authenticated user. Stores as data URI.
- **Callers:** iOS app settings — Change Photo.
- **Security:** API token. Multipart form-data, field `avatar`, max 1 MB. Returns updated `User`.

```bash
curl -s -X POST $BASE_URL/api/profile/avatar \
  -H "Authorization: Bearer $API_TOKEN" \
  -F "avatar=@photo.jpg" | jq
```

### `DELETE /api/profile/avatar`
- **Usecase:** Remove custom avatar; reverts to OIDC provider photo URL.
- **Callers:** iOS app settings.
- **Security:** API token. Returns 204.

```bash
curl -s -X DELETE $BASE_URL/api/profile/avatar -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `GET /api/settings`
- **Usecase:** App-wide settings readable by all users (currently: `family_name`).
- **Callers:** iOS app settings header.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/settings -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/chores`
- **Usecase:** List chores. Optional filters: `status`, `assigned_to`.
- **Callers:** iOS app chores list.
- **Security:** API token.

```bash
curl -s "$BASE_URL/api/chores?status=pending&assigned_to=<userID>" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/chores`
- **Usecase:** Create a chore (full parity with the web form).
- **Callers:** iOS app.
- **Security:** API token. Body JSON: `name` required; all other fields optional.
- **Body fields:** `name`, `description`, `categoryId`, `assignees` (eligible-assignee
  rotation pool), `dueDate` (`YYYY-MM-DD`), `dueTime` (`HH:MM`), `recurrenceType`
  (`none`/`daily`/`weekly`/`monthly`/`custom`), `recurrenceInterval` (int ≥1),
  `recurrenceDays` (weekly; `["monday",…]`), `recurrenceDayOfMonth` (monthly; 1–31),
  `recurrenceUnit` (custom; `days`/`weeks`/`months`), `recurrenceUntil` (`YYYY-MM-DD`),
  `recurrenceCount` (int ≥1), `recurOnComplete` (bool).

```bash
curl -s -X POST $BASE_URL/api/chores \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Take out trash","description":"Bins by the curb","categoryId":"cat-1","assignees":["user-id-1","user-id-2"],"dueDate":"2026-04-10","dueTime":"18:30","recurrenceType":"weekly","recurrenceInterval":1,"recurrenceDays":["monday","thursday"],"recurrenceUntil":"2026-12-31","recurOnComplete":false}' | jq
```

### `GET /api/chores/{id}`
- **Usecase:** Fetch a single chore.
- **Callers:** iOS app detail view.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/chores/<choreID> -H "Authorization: Bearer $API_TOKEN" | jq
```

### `PUT /api/chores/{id}`
- **Usecase:** Update chore. Accepts the same body fields as `POST /api/chores`
  (name, description, category, assignees, due date/time, full recurrence config, end
  conditions, recur-on-complete). Editing an occurrence that belongs to a series syncs the
  series definition. Omitted optional fields are cleared.
- **Callers:** iOS app edit.
- **Security:** API token.

```bash
curl -s -X PUT $BASE_URL/api/chores/<choreID> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated","description":"","assignees":["user-id-1"],"dueDate":"2026-04-15","recurrenceType":"monthly","recurrenceDayOfMonth":15,"recurrenceCount":6}' | jq
```

### `DELETE /api/chores/{id}`
- **Usecase:** Delete chore; also wipes future pending siblings in its series.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X DELETE $BASE_URL/api/chores/<choreID> -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `POST /api/chores/{id}/complete`
- **Usecase:** Mark chore done by the token's user. Spawns next recurrence.
- **Callers:** iOS app, shortcuts.
- **Security:** API token. 409 if already complete.

```bash
curl -s -X POST $BASE_URL/api/chores/<choreID>/complete -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `GET /api/users`
- **Usecase:** All users (for assignee pickers).
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/users -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/users/{id}`
- **Usecase:** Single user.
- **Callers:** iOS app profile views.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/users/<userID> -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/categories`
- **Usecase:** All chore categories.
- **Callers:** iOS app, chore forms.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/categories -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/dashboard`
- **Usecase:** Counts + lists for today/overdue chores and today/week meals.
- **Callers:** iOS app dashboard, iOS Today widget, HA integration.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/dashboard -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/meals?week=YYYY-MM-DD`
- **Usecase:** Meal plans for the seven days starting at `week`. The date is **not**
  snapped to a week boundary — the value you pass *is* the first day, and the range
  is `[week, week+6]`. The web planner and the iOS app both anchor their weeks to
  the preceding Friday before calling this.
- **Callers:** iOS app meal planner.
- **Security:** API token.

```bash
curl -s "$BASE_URL/api/meals?week=2026-04-06" -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/meals`
- **Usecase:** Upsert a meal plan entry.
- **Callers:** iOS app.
- **Security:** API token. Body requires `date`, `mealType`, `name`.

```bash
curl -s -X POST $BASE_URL/api/meals \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-04-07","mealType":"dinner","name":"Tacos","recipeID":"<optional>"}' | jq
```

### `DELETE /api/meals?date=YYYY-MM-DD&mealType=dinner`
- **Usecase:** Remove a meal plan slot.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X DELETE "$BASE_URL/api/meals?date=2026-04-07&mealType=dinner" \
  -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `POST /api/recipes/extract`
- **Usecase:** Scrape recipe fields from a URL. Sources are tried in descending
  order of reliability: JSON-LD → microdata → platform oEmbed (TikTok) →
  `VideoObject` JSON-LD → Open Graph tags and caption text. Social video posts
  (TikTok, Instagram) carry no structured recipe data, so they land on the last
  two: the caption is parsed for `Ingredients:` / `Method:` sections, and when
  there is nothing to parse the response still carries `title` and `imageURL`.
- **Response:** always `200` with an `ExtractedRecipe`; every field is
  best-effort and may be empty. An empty object means nothing at all was found.
- **Callers:** iOS app "import from URL", iOS share extension.
- **Security:** API token. SSRF-hardened (see commit 273c218) — the oEmbed and
  short-link lookups go through the same guarded client, and a redirect target
  is re-validated before use.

```bash
curl -s -X POST $BASE_URL/api/recipes/extract \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/recipe"}' | jq

# Social post — expect title/imageURL, plus ingredients/steps if the caption lists them
curl -s -X POST $BASE_URL/api/recipes/extract \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.tiktok.com/@someone/video/123"}' | jq
```

### `GET /api/recipes`
- **Usecase:** List all recipes. This is a **summary** response: `Steps` is always
  `null` and `StepDurations` is absent, because the list query does not select
  the steps column. Use `GET /api/recipes/{id}` when you need the method.
  `Ingredients` *is* returned in full on every row.
- **Callers:** iOS app, meal picker.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/recipes -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/recipes/{id}`
- **Usecase:** Single recipe with ingredients + steps.
- **Callers:** iOS app detail/cook mode, watchOS cook mode.
- **Security:** API token.
- **Steps fallback:** recipes created before the `steps` column keep their method
  in the legacy `Instructions` blob. When `Steps` is empty and `Instructions` is
  not, `Steps` is populated by splitting `Instructions` on newlines, so every
  recipe is cookable. `Instructions` is still returned unchanged.
- **`StepDurations`:** a computed, never-persisted array of timer hints in
  seconds, positionally aligned with `Steps`, `null` where a step has no
  recognisable timing. Parsed from the step's own prose ("simmer for 20 minutes"
  → `1200`); a range takes its lower bound and the first duration in a step wins.
  Clients may ignore it entirely.

```bash
curl -s $BASE_URL/api/recipes/<recipeID> -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/recipes`
- **Usecase:** Create recipe (with optional base64 image).
- **Callers:** iOS app.
- **Security:** API token. Body requires `title`.

```bash
curl -s -X POST $BASE_URL/api/recipes \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Pasta","steps":["Boil","Drain"],"ingredients":[{"name":"Main","items":["200g pasta","1 tbsp salt"]}],"mealType":"dinner","servings":2}' | jq
```

### `PUT /api/recipes/{id}`
- **Usecase:** Replace recipe fields.
- **Callers:** iOS app edit.
- **Security:** API token.

```bash
curl -s -X PUT $BASE_URL/api/recipes/<recipeID> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Pasta v2","steps":["Boil"],"ingredients":[]}' | jq
```

### `DELETE /api/recipes/{id}`
- **Usecase:** Delete recipe; also detaches from meal plans.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X DELETE $BASE_URL/api/recipes/<recipeID> -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `GET /api/recipes/{id}/image[?size=thumb]`
- **Usecase:** Serves recipe image bytes. `size=thumb` returns a copy scaled to
  fit a 200px box (aspect ratio preserved, never upscaled) for list rows and
  small screens; any other `size` value serves the original. PNG sources stay
  PNG so transparency survives; JPEG, GIF and WebP thumbnails are re-encoded as
  JPEG.
- **Callers:** iOS app, watchOS app.
- **Security:** API token.
- **Caching:** responds with `ETag` and `Cache-Control: private, max-age=86400`.
  The ETag covers the variant as well as the bytes, so the full image and its
  thumbnail never collide. Send `If-None-Match` to get a `304 Not Modified`.

```bash
curl -s $BASE_URL/api/recipes/<recipeID>/image -H "Authorization: Bearer $API_TOKEN" -o recipe.jpg
curl -s "$BASE_URL/api/recipes/<recipeID>/image?size=thumb" -H "Authorization: Bearer $API_TOKEN" -o thumb.jpg

# Revalidate with the ETag from a previous response — expect 304.
curl -s -o /dev/null -w "%{http_code}\n" $BASE_URL/api/recipes/<recipeID>/image \
  -H "Authorization: Bearer $API_TOKEN" -H 'If-None-Match: "full-<hash>"'
```

### `GET /api/calendar?view=month|week|day&date=YYYY-MM-DD|month=YYYY-MM`
- **Usecase:** Unified view: chores + iCal events + meals for the range.
- **Callers:** iOS app calendar tab; iOS app home screen and the Today widget
  both use `view=day` for today's events.
- **Security:** API token.

```bash
curl -s "$BASE_URL/api/calendar?view=month&month=2026-04" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

---

### Inventory API

Stock tracking across home **areas** (e.g. Laundry cupboard), each holding
**items**. An item's `trackingMode` is either `count` (whole units, default) or
`level` (a 0–100 fill percentage for partially-used items like bottles). The
`lowAt` threshold is read as a count in count mode and a percentage in level mode;
an item is "low" when `quantity <= lowAt` (count) or `level <= lowAt` (level).
That flag and the cross-area "running low" rollup are derived by clients — the
server stores and returns raw fields. Any authenticated user can manage inventory.
`icon` ∈ {box, drop, cart, pills, sparkles, heart, cloud, flame}; `tint` ∈ {blue,
orange, teal, green, purple, indigo, red}; both default to `box`/`blue` when blank.

### `GET /api/inventory`
- **Usecase:** List every area with its items nested (single call backing the iOS
  Inventory home screen). Empty list returned as `[]`.
- **Callers:** iOS app inventory tab.
- **Security:** API token.

```bash
curl -s "$BASE_URL/api/inventory" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/inventory/areas`
- **Usecase:** Create an area. Body JSON: `name` required; `icon`, `tint` optional.
- **Callers:** iOS app (Add Area sheet).
- **Security:** API token.

```bash
curl -s -X POST $BASE_URL/api/inventory/areas \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Laundry cupboard","icon":"drop","tint":"blue"}' | jq
```

### `PUT /api/inventory/areas/{id}`
- **Usecase:** Update an area's name/icon/tint. `name` required.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X PUT $BASE_URL/api/inventory/areas/<id> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Utility room","icon":"sparkles","tint":"green"}' | jq
```

### `DELETE /api/inventory/areas/{id}`
- **Usecase:** Delete an area; its items are cascade-deleted. Returns 204.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X DELETE $BASE_URL/api/inventory/areas/<id> \
  -H "Authorization: Bearer $API_TOKEN" -i
```

### `POST /api/inventory/areas/{id}/items`
- **Usecase:** Add an item to an area. Body JSON: `name` required; `trackingMode`
  (`count` default, or `level`), `quantity`, `level` (0–100), `unit`, `lowAt`
  optional. `quantity`/`lowAt` clamped at 0; `level` and (in level mode) `lowAt`
  clamped to 0–100.
- **Callers:** iOS app (Add Item sheet).
- **Security:** API token.

```bash
curl -s -X POST $BASE_URL/api/inventory/areas/<id>/items \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Fabric softener","trackingMode":"level","level":40,"lowAt":20}' | jq
```

### `PUT /api/inventory/items/{id}`
- **Usecase:** Update an item (name/trackingMode/quantity/level/unit/lowAt). Drives
  the stepper's quantity changes and the level slider. `name` required.
- **Callers:** iOS app (stepper + edit item sheet).
- **Security:** API token.

```bash
curl -s -X PUT $BASE_URL/api/inventory/items/<id> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Fabric softener","trackingMode":"level","level":75,"lowAt":20}' | jq
```

### `DELETE /api/inventory/items/{id}`
- **Usecase:** Delete an item. Returns 204.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X DELETE $BASE_URL/api/inventory/items/<id> \
  -H "Authorization: Bearer $API_TOKEN" -i
```

---

### Admin-only API routes (admin role required)

### `POST /api/users/{id}/promote`
- **Usecase:** Promote a user to admin role.
- **Callers:** iOS app admin settings — Family Members.
- **Security:** API token + admin role. Returns updated `User`.

```bash
curl -s -X POST $BASE_URL/api/users/<userID>/promote -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/users/{id}/demote`
- **Usecase:** Demote a user to member role.
- **Callers:** iOS app admin settings — Family Members.
- **Security:** API token + admin role. Returns updated `User`.

```bash
curl -s -X POST $BASE_URL/api/users/<userID>/demote -H "Authorization: Bearer $API_TOKEN" | jq
```

### `PATCH /api/settings`
- **Usecase:** Update app-wide settings. Currently supports `family_name`.
- **Callers:** iOS app admin settings — Family Name.
- **Security:** API token + admin role. Returns 204.

```bash
curl -s -X PATCH $BASE_URL/api/settings \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"family_name":"The Smiths"}' -w "%{http_code}\n"
```

### `POST /api/categories`
- **Usecase:** Create a chore category.
- **Callers:** iOS app admin settings — Categories.
- **Security:** API token + admin role. Returns created `Category`.

```bash
curl -s -X POST $BASE_URL/api/categories \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Cleaning"}' | jq
```

### `PUT /api/categories/{id}`
- **Usecase:** Rename a category.
- **Callers:** iOS app admin settings — Categories.
- **Security:** API token + admin role. Returns updated `Category`.

```bash
curl -s -X PUT $BASE_URL/api/categories/<categoryID> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Deep Cleaning"}' | jq
```

### `DELETE /api/categories/{id}`
- **Usecase:** Delete a category.
- **Callers:** iOS app admin settings — Categories.
- **Security:** API token + admin role. Returns 204.

```bash
curl -s -X DELETE $BASE_URL/api/categories/<categoryID> -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `GET /api/tokens`
- **Usecase:** List all API tokens with metadata (no plaintext).
- **Callers:** iOS app admin settings — API Tokens.
- **Security:** API token + admin role.

```bash
curl -s $BASE_URL/api/tokens -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/tokens`
- **Usecase:** Generate a new API token. Returns raw token **once**.
- **Callers:** Admin UI (`/admin` page), iOS app admin settings.
- **Security:** API token + admin role. Body: form `name=` or JSON `{"name":"..."}`.

```bash
curl -s -X POST $BASE_URL/api/tokens \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"API Client"}' | jq
```

### `DELETE /api/tokens/{id}`
- **Usecase:** Revoke a token.
- **Callers:** Admin UI, iOS app admin settings.
- **Security:** API token + admin role. Returns 200.

```bash
curl -s -X DELETE $BASE_URL/api/tokens/<tokenID> -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

---

## Web routes

All routes below render HTML (full page or HTMX fragment). Called by the browser
navigating the app. Gated by `RequireUser` (session cookie or Bearer).

### Dashboard / profile (authenticated)

| Method + Path | Usecase | Extra Security |
|---|---|---|
| `GET /` | Dashboard (stats, today's chores, meals) | — |
| `GET /leaderboard` | Chore completion leaderboard | — |
| `GET /profile` | Profile page | — |
| `POST /profile/avatar` | Upload avatar (multipart) | — |
| `POST /profile/avatar/delete` | Remove avatar | — |
| `GET /avatar/{userID}` | Serve avatar bytes | — |

```bash
curl -s $BASE_URL/ -b "session=$SESSION"
curl -s $BASE_URL/avatar/<userID> -b "session=$SESSION" -o avatar.png
```

### Chores (web)

| Method + Path | Usecase | Admin? |
|---|---|---|
| `GET /chores` | Chore list page/HTMX partial | no |
| `GET /chores/{id}/detail` | Chore detail fragment | no |
| `POST /chores/{id}/complete` | Mark complete | no |
| `GET /chores/new` | Create form | no |
| `POST /chores` | Create | no |
| `GET /chores/{id}/edit` | Edit form | no |
| `POST /chores/{id}` | Update | no |
| `POST /chores/{id}/delete` | Delete | no |
| `POST /chores/history/delete` | Clear completed chore history | no |

`POST /chores` and `POST /chores/{id}` accept optional recurrence end conditions:
`recurrence_until` (date, `YYYY-MM-DD`) and `recurrence_count` (positive integer).
Either bounds how far a recurring series is generated.

```bash
curl -s $BASE_URL/chores -b "session=$SESSION"
curl -s -X POST $BASE_URL/chores/<id>/complete -b "session=$SESSION"
```

### Calendar subscriptions (`/calendars`, web)

| Method + Path | Usecase | Admin? |
|---|---|---|
| `GET /calendars` | List external iCal subscriptions | no |
| `POST /calendars` | Add subscription | yes |
| `POST /calendars/{id}/delete` | Remove | yes |
| `POST /calendars/{id}/refresh` | Force refetch feed | yes |
| `POST /calendars/{id}/color` | Update display color | yes |

```bash
curl -s $BASE_URL/calendars -b "session=$SESSION"
```

### Meals (web)

| Method + Path | Usecase |
|---|---|
| `GET /meals` | Weekly planner page |
| `POST /meals` | Save meal cell |
| `POST /meals/delete` | Clear meal cell |
| `GET /meals/cell` | HTMX fragment for a single cell |
| `GET /meals/recipes` | Recipe picker fragment |
| `GET /meals/dismiss` | Dismiss picker fragment |

```bash
curl -s $BASE_URL/meals -b "session=$SESSION"
```

### Recipes (web)

| Method + Path | Usecase |
|---|---|
| `GET /recipes/import` | Import-from-URL form |
| `GET /recipes` | List page |
| `GET /recipes/new` | Create form |
| `GET /recipes/ingredient-group` | HTMX: add ingredient group row |
| `GET /recipes/step` | HTMX: add step row |
| `GET /recipes/{id}` | Detail page |
| `GET /recipes/{id}/image` | Serve image |
| `GET /recipes/{id}/cook` | Cook mode page |
| `POST /recipes` | Create |
| `POST /recipes/{id}/image` | Upload image |
| `POST /recipes/{id}/image/delete` | Remove image |
| `GET /recipes/{id}/edit` | Edit form |
| `POST /recipes/{id}` | Update |
| `POST /recipes/{id}/delete` | Delete |

```bash
curl -s $BASE_URL/recipes -b "session=$SESSION"
curl -s $BASE_URL/recipes/<id> -b "session=$SESSION"
```

### Calendar view (web)

| Method + Path | Usecase |
|---|---|
| `GET /calendar` | Unified calendar page |
| `GET /calendar/event-detail` | Event detail fragment |

```bash
curl -s "$BASE_URL/calendar?view=month&month=2026-04" -b "session=$SESSION"
```

### Categories (admin, web)

| Method + Path | Usecase |
|---|---|
| `POST /categories` | Create |
| `GET /categories/{id}/edit` | Edit form fragment |
| `GET /categories/{id}/cancel` | Cancel edit fragment |
| `POST /categories/{id}` | Update |
| `POST /categories/{id}/delete` | Delete |

### Admin panel (admin, web)

| Method + Path | Usecase |
|---|---|
| `GET /admin/users` | User management page |
| `POST /admin/users/{id}/promote` | Grant admin role |
| `POST /admin/users/{id}/demote` | Revoke admin role |
| `POST /admin/settings` | Update family settings |
| `POST /admin/tokens` | Create API token |
| `GET /admin/backup` | Download SQLite backup |
| `POST /admin/restore` | Upload SQLite backup to restore |

```bash
curl -s $BASE_URL/admin/users -b "session=$SESSION"
curl -s $BASE_URL/admin/backup -b "session=$SESSION" -o backup.db
```

---

## Auth summary

Only three flows:

| Flow | Routes |
|---|---|
| **Public** | `/health`, `/static/*`, `/api/client-config`, `/login`, `/auth/callback`, `/logout` |
| **OIDC bearer (one-time)** | `POST /api/auth/exchange` |
| **Authed user** (session OR Bearer, via `RequireUser`) | Everything else |
| **Admin user** (`+ RequireAdmin`) | `/admin/*`, categories write, calendars write, `/api/tokens*` |

Session cookie and Bearer token both resolve to the same `User` via `RequireUser`
— handlers are mechanism-agnostic. Chores, recipes, and meals are open to any
authed user (no admin gate).

All login-adjacent routes (`/login`, `/auth/callback`, `/api/auth/exchange`) carry an
additional 10-req/min per-IP limit on top of the global 100/min limit.

## Household location and weather

These endpoints require a session cookie or Bearer API token. The iOS app calls
them. Existing clients do not need to call them.

### `GET /api/home-info`

Return `latitude`, `longitude`, `timezone`, and `arrivalMarginMinutes`. Coordinates
are null before setup. The default margin is 10 minutes.

```bash
curl -s "$BASE_URL/api/home-info" -H "Authorization: Bearer $API_TOKEN"
```

### `PATCH /api/home-info`

Admin only. Save confirmed device coordinates, an IANA timezone, and an integer
arrival margin from 0 to 120 minutes. Return the saved home information. Return
400 for invalid values and 403 for a member. Unspecified properties retain their
current values. Initial setup requires both coordinates.

```bash
curl -s -X PATCH "$BASE_URL/api/home-info" \
  -H "Authorization: Bearer $API_TOKEN" -H 'Content-Type: application/json' \
  -d '{"latitude":51.8,"longitude":0.7,"timezone":"Europe/London","arrivalMarginMinutes":10}'
```

### `GET /api/weather`

Return `status`, `fetchedAt`, `current`, `high`, `low`, `hours`, and `advice`.
Each weather hour contains `time`, `temperature`, `code`, `rainProbability`, and
`windSpeed`. Times use RFC 3339. Units are Celsius, percent, and km/h.

Return `setup_required` before the home location is set. Return 502 when the
provider fails or returns an incomplete forecast. Reuse successful forecasts
for 15 minutes by home coordinates and timezone. Never return expired forecasts.
Open-Meteo supplies the data. Advice uses fixed rules for the next six hours.

```bash
curl -s "$BASE_URL/api/weather" -H "Authorization: Bearer $API_TOKEN"
```

### `GET /api/departures`

Return `status`, `items`, and `selectedEventID`. Each item contains `eventID`,
`title`, `destination`, `startTime`, `leaveAt`, `drivingSeconds`, and `fetchedAt`.
Times use RFC 3339. Select today's next eligible event by start time in the home
timezone. `selectedEventID` is null when there are no eligible events.

Use current traffic-aware Google driving estimates from home. Subtract the
arrival margin from the event start. Skip all-day, started, online, unresolved,
and ambiguous events. Address resolution requires one complete, numbered address
match. A vague location such as School does not receive a countdown.

Return `setup_required` before location setup. Return 502 for provider failure,
including a missing `GOOGLE_MAPS_API_KEY`. An empty list is a successful result.
Return `Cache-Control: no-store`. Do not persist Google route responses.

```bash
curl -s "$BASE_URL/api/departures" -H "Authorization: Bearer $API_TOKEN"
```

## Shared shopping list

All shopping endpoints require a session cookie or Bearer API token. All members
can change the shared household list. The iOS app calls these routes.

An item has `id`, `name`, `inventoryItemID`, `purchased`, `purchaseID`, `restocked`,
and `version`. Names contain 1–200 UTF-8 bytes after whitespace removal at the
edges. There are no shopping quantities or notes.

### `GET /api/shopping`

Return every active and purchased item as an array. Return `[]` for an empty list.

```bash
curl -s "$BASE_URL/api/shopping" -H "Authorization: Bearer $API_TOKEN"
```

### `POST /api/shopping/items`

Accept `name` and an optional `inventoryItemID`. Return 201 and the item. If an
active item already links to that inventory item, return the existing item.
Return 400 for an invalid name and 404 for a missing inventory item.

```bash
curl -s -X POST "$BASE_URL/api/shopping/items" \
  -H "Authorization: Bearer $API_TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Milk"}'
```

### `PATCH /api/shopping/items/{id}`

Accept `version`, an optional `name`, and an optional `purchased` boolean. Return
the updated item. Return 409 for a stale version or a duplicate active inventory
link. Each new purchase receives a new `purchaseID`. Restore does not reverse a
confirmed restock.

```bash
curl -s -X PATCH "$BASE_URL/api/shopping/items/$ITEM_ID" \
  -H "Authorization: Bearer $API_TOKEN" -H 'Content-Type: application/json' \
  -d '{"version":0,"purchased":true}'
```

### `DELETE /api/shopping/items/{id}`

Delete one item and its purchase records. Return 204, or 404 if it is absent.

```bash
curl -s -X DELETE "$BASE_URL/api/shopping/items/$ITEM_ID" -H "Authorization: Bearer $API_TOKEN"
```

### `DELETE /api/shopping/purchased`

Clear all purchased items and their purchase records. Return 204.

```bash
curl -s -X DELETE "$BASE_URL/api/shopping/purchased" -H "Authorization: Bearer $API_TOKEN"
```

### `POST /api/shopping/items/{id}/restock`

Accept `purchaseID` and either `quantity` for a count item or `level` for a level
item. Quantity adds 1–1,000,000 units. Level sets an integer percentage from 0 to
100. Return the updated shopping item. Reject the wrong tracking mode with 400.
Return 409 if the item is not purchased or the purchase identifier is stale.

Apply the inventory update and record the restock in one transaction. Repeated
requests for the same purchase return success without applying stock twice.
Deleting linked inventory preserves the shopping item and removes its link.

```bash
curl -s -X POST "$BASE_URL/api/shopping/items/$ITEM_ID/restock" \
  -H "Authorization: Bearer $API_TOKEN" -H 'Content-Type: application/json' \
  -d '{"purchaseID":"purchase-id","quantity":2}'
```


## Chore reminders and notification devices

All endpoints use the existing session or Bearer authentication.

### `GET /api/chore-reminders`

Return the family schedule and the server push status. All users can read it.

```json
{"enabled":false,"times":["10:00","16:30"],"timezone":"Europe/London","pushAvailable":false}
```

The timezone comes from Home Info. The default timezone is `Europe/London`.
`pushAvailable` reports whether the server has APNs credentials configured.

### `PATCH /api/chore-reminders`

Admin only. Replace the enabled flag and both reminder times.

```json
{"enabled":true,"times":["10:00","16:30"]}
```

Use two distinct times in `HH:mm` format. Invalid times return 400.
Enablement returns 503 when the server has no APNs credentials.
The response uses the same shape as GET.

At each time, the server counts all unfinished family chores due today or overdue.
A chore due later today counts. An undated chore counts only when its status is overdue.
The server sends nothing when the count is zero.

### `PUT /api/notification-devices/{id}`

Register or update the authenticated user's installation. Return 204.

```json
{"token":"<hexadecimal APNs device token>","environment":"production"}
```

Use a stable installation identifier of at most 128 characters.
The environment is `sandbox` or `production`.
A device identifier or token owned by another user returns 409.
Bearer registrations stop receiving reminders when their API token expires or is revoked.

### `DELETE /api/notification-devices/{id}`

Remove the authenticated user's installation. Return 204, including when it is absent.
This endpoint does not remove another user's registration.

See [Chore reminder setup](chore-reminders.md) for Apple credentials and device checks.

### `POST /api/notifications/broadcast`

Admin only. Send a plain-text message to all enabled family iPhones.
The family chore schedule does not need to be enabled.

```json
{"requestID":"4f2e6250-1821-4c0f-b566-3769a534f0c2","message":"Dinner is ready."}
```

Use a UUID for `requestID`. Reuse that UUID when you retry the same message.
Use a new UUID for a new message. Reusing a UUID with different text returns 409.
The trimmed message must contain 1 to 500 Unicode code points. Invalid input returns 400.
Missing push configuration returns 503. The endpoint permits five requests per minute per IP address.

```json
{"accepted":2,"failed":0}
```

`accepted` counts notifications accepted by Apple. It does not prove presentation on an iPhone.
`failed` includes pending deliveries and rejected deliveries.
A retry attempts transient failures within five minutes of the first request.
It does not resend accepted notifications. Expired requests do not send.
The notification title is “Family message”. Tap it to open Today.
The server retains messages and delivery results for 30 days, then removes them on the next broadcast request.

### `PATCH /api/inventory/items/{id}/stock`

Change stock without replacing the item's name, area, unit, tracking mode, or
low-stock threshold. Use the same session or bearer authentication as the other
inventory endpoints. Siri uses this endpoint for stock updates.

Supply exactly one integer field:

| Field | Tracking mode | Behavior |
| --- | --- | --- |
| `quantity` | `count` | Set a nonnegative count. |
| `quantityDelta` | `count` | Add a signed count. Clamp the result at zero. |
| `level` | `level` | Set a percentage from 0 to 100. |

The server applies count adjustments atomically and returns the saved
`InventoryItem` with status `200`. It returns `400` for invalid values,
multiple fields, a tracking-mode mismatch, or count overflow. It returns `404`
when the item does not exist. Existing full-item endpoints remain compatible.

Do not automatically retry a relative change after an uncertain response.
Read the current stock before the user submits another change.

```sh
curl -s -X PATCH "$BASE_URL/api/inventory/items/$ITEM_ID/stock" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"quantityDelta":-1}'
```
