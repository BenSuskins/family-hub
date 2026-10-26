# Family Hub — Endpoint Reference

All routes wired in `server/internal/server/server.go`.

**Global middleware** (all routes): request logger, bounded-cardinality Prometheus request metrics, panic recovery, browser CORS, gzip compression, security headers, and per-IP rate limiting (100 req/min). Request metrics cover every route. Server failures log the method, route, status, and underlying error without logging credentials or request bodies. Authenticated routes require a Bearer token scoped for that route.

**Conventions used below:**

```bash
export BASE_URL="http://localhost:8080"
export API_TOKEN="<token from /api/auth/exchange or POST /api/tokens>"
```

---

## Public routes (no auth)

### `GET /metrics`
- **Usecase:** Expose Prometheus request counts and latency histograms.
- **Callers:** Homelab Prometheus scraper on the private network.
- **Security:** No authentication. The endpoint exposes aggregate request counts and route templates only; keep the scrape target on the private network.
- **Metrics:** `familyhub_http_requests_total` counts by method, route, and status. `familyhub_http_request_duration_seconds` provides cumulative histogram buckets, sum, and count using the same labels.
- **Dependency metrics:** `familyhub_dependency_failures_total` counts failed calls to optional upstream services by dependency and route. Dependencies currently include weather, departures, rail, and calendar feeds.
- **Cardinality:** Uses Chi route patterns rather than concrete path values; unsupported methods are grouped as `OTHER`.

```bash
curl -s "$BASE_URL/metrics"
```

### `GET /health`
- **Usecase:** Liveness probe. Returns `200 ok`.
- **Callers:** Docker healthcheck, uptime monitoring, load balancers.
- **Security:** None. Exempt from auth.

```bash
curl -s $BASE_URL/health
```

### `GET /api/client-config`
- **Usecase:** Returns OIDC `clientID` + `issuer` so the iOS app can start the auth flow.
- **Callers:** iOS app on first launch.
- **Security:** None (public, non-sensitive).

```bash
curl -s $BASE_URL/api/client-config | jq
```

---

## Client token exchange

### `POST /api/auth/exchange`
- **Usecase:** Swap an OIDC bearer for a long-lived Family Hub API token. Creates the user on first login.
- **Callers:** iOS app after OIDC login.
- **Security:** Rate limited 10/min per IP. Validates bearer against OIDC userinfo endpoint and replaces only prior iOS app tokens.

```bash
curl -s -X POST "$BASE_URL/api/auth/exchange" \
  -H "Authorization: Bearer <OIDC_ACCESS_TOKEN>" | jq
```

---

## Authenticated surface (Bearer token)

Every route below resolves a user from `Authorization: Bearer $API_TOKEN`.
The API returns JSON errors for failed requests. Image and backup endpoints keep
their binary success responses.

### JSON API routes

### `GET /api/commute/trains/from/{origin}/to/{destination}`

- **Use case:** Return upcoming trains for the requested National Rail station route. The server does not store Home or Work station codes.
- **Callers:** The iOS Today commute card. iOS sends Home → Work or Work → Home codes for the active card.
- **Security:** Bearer API token. The provider token stays on the server.
- **Parameters:** `origin` and `destination` are three-letter alphabetic CRS codes in the path. The server trims and uppercases them. `count=1..10` is optional and defaults to `5`.
- **Direction:** The path's origin and destination define the route. The response has no Home/Work direction field; the iOS app maps it to the card it requested.
- **Window:** The provider returns up to the requested count within the next two hours. This is not an all-day timetable.
- **Order:** Scheduled departure order, including cancelled trains. Return `trains: []` when no trains are available.
- **Times:** Train times use `HH:mm` in `Europe/London`. Arrival means arrival at the commute destination, including intermediate stops.
- **Platform:** The departure platform. Unknown platforms and times are `null`.
- **Status:** `on_time`, `delayed`, `cancelled`, or `unknown`. A cancelled destination stop means a cancelled commute.
- **Estimates:** An "On time" estimate uses the scheduled time. An unspecified estimate is `null`. Cancelled trains have null estimates.
- **Freshness:** Successful results use a 30-second cache for each origin, destination, and count. Expired results are never returned.
- **Timestamps:** `generatedAt` comes from the provider. `fetchedAt` records the server fetch. Both use RFC 3339.
- **Errors:** `400` for invalid parameters; `503` for an absent provider token; `502` for provider failures; `504` for provider timeouts.
- **Configuration:** Set `HUXLEY_ACCESS_TOKEN` to the National Rail token. `HUXLEY_BASE_URL` defaults to `https://huxley2.azurewebsites.net`.

```bash
curl -s "$BASE_URL/api/commute/trains/from/KEL/to/LST?count=5" \
  -H "Authorization: Bearer $API_TOKEN" | jq

curl -s "$BASE_URL/api/commute/trains/from/LST/to/KEL?count=3" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

Example response:

```json
{
  "origin": {"code": "KEL"},
  "destination": {"code": "LST"},
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

### `GET /api/scratchpad` / `PUT /api/scratchpad`
- **Use case:** Read or save the shared drawing-only household scratchpad.
- **Callers:** Authenticated member app and paired display. `PUT` accepts `{"strokes":[{"color":"#3c654c","points":[{"x":0.2,"y":0.3}]}]}`; normalized points must be in the 0..1 range and colors use the fixed palette.
- **Security:** Bearer API token. Text notes and arbitrary SVG are not accepted. The body is limited to 1 MiB.

```bash
curl -s "$BASE_URL/api/scratchpad" -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/display-pairing/challenges`
- **Use case:** Start a short-lived wall display pairing request. Returns an id and a secret code for the display's QR handoff.
- **Callers:** Display setup screen.
- **Security:** Public and rate limited to 5/min per IP. The challenge expires after five minutes.

```bash
curl -s -X POST "$BASE_URL/api/display-pairing/challenges" | jq
```

### `POST /api/display-pairing/challenges/{id}/approve`
- **Use case:** Approve a display pairing and assign its name.
- **Callers:** Authenticated iOS admin after scanning the QR code.
- **Security:** Bearer API token and admin role. Requires the challenge secret; a challenge can be approved once before it expires.

```bash
curl -s -X POST "$BASE_URL/api/display-pairing/challenges/<id>/approve" \
  -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" \
  -d '{"pairingCode":"<code>","name":"Kitchen display"}' | jq
```

### `POST /api/display-pairing/challenges/{id}/claim`
- **Use case:** Exchange an approved challenge for the display's revocable, least-privilege bearer credential.
- **Callers:** Display setup screen.
- **Security:** Public and rate limited to 10/min per IP. Requires the challenge secret. Claim succeeds once.

```bash
curl -s -X POST "$BASE_URL/api/display-pairing/challenges/<id>/claim" \
  -H "Content-Type: application/json" -d '{"pairingCode":"<code>"}' | jq
```

### `GET /api/displays` / `PATCH /api/displays/{id}` / `DELETE /api/displays/{id}`
- **Use case:** List, configure, or revoke paired wall displays.
- **Callers:** iOS admin settings.
- **Security:** Bearer API token and admin role. Revocation deletes the display credential immediately.

```bash
curl -s "$BASE_URL/api/displays" -H "Authorization: Bearer $API_TOKEN" | jq
curl -s -X PATCH "$BASE_URL/api/displays/<id>" -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" -d '{"enabledSections":["calendar","meals","chores","weather","departures","bins"]}' | jq
curl -s -X DELETE "$BASE_URL/api/displays/<id>" -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `PATCH /api/displays/{id}/name` / `GET /api/displays/{id}/commute` / `PUT /api/displays/{id}/commute`
- **Use case:** Rename a paired display or read and update its shared commute schedule.
- **Callers:** iOS admin settings.
- **Security:** Bearer API token and admin role. Commute schedules belong to the display, not its pairing admin.

```bash
curl -s -X PATCH "$BASE_URL/api/displays/<id>/name" -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" -d '{"name":"Kitchen display"}' | jq
curl -s "$BASE_URL/api/displays/<id>/commute" -H "Authorization: Bearer $API_TOKEN" | jq
curl -s -X PUT "$BASE_URL/api/displays/<id>/commute" -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" -d '{"toWork":{"enabled":true,"weekdays":[2,3,4,5,6],"start":"06:30","end":"09:30"},"toHome":{"enabled":false,"weekdays":[2,3,4,5,6],"start":"16:00","end":"19:30"},"homeStationCode":"KGX","workStationCode":"CBG","trainCount":3}' | jq
```

### Display API

These routes accept only a display-scoped Bearer token. The token cannot authenticate to member or admin routes. Pairing creates a token with no expiry; an admin disconnect revokes it. The web display stores it in browser `localStorage`, scoped to the API origin and path. Closing and reopening the same browser profile preserves pairing. Clearing browser site data removes the local token, but an admin must disconnect the old server-side pairing to revoke that credential.

| Route | Use case | Caller | Access |
|---|---|---|---|
| `GET /api/display/home-info` | Read the display's home settings, household timezone, and household name. | Wall display | Display token |
| `GET /api/display/settings` | Read this display's enabled sections. | Wall display | Display token |
| `GET /api/display/people` | Read household member ids and names for chore attribution. | Wall display | Display token |
| `GET /api/display/weather` | Read current, hourly, and seven-day weather. | Wall display | Display token |
| `GET /api/display/departures` | Read today's departure context. | Wall display | Display token |
| `GET /api/display/bin-collections` | Read the household bin collection schedule. | Wall display | Display token |
| `GET /api/display/chores` | Read household chores. | Wall display | Display token |
| `POST /api/display/chores/{id}/complete` | Complete a chore and attribute it to a selected household member. | Wall display | Display token |
| `GET /api/display/meals` | Read the current meal week. | Wall display | Display token |
| `POST /api/display/meals` | Add or replace a meal for its date and meal type. | Wall display | Display token |
| `GET /api/display/calendar` | Read calendar events, chores, and meals for the requested range. | Wall display | Display token |
| `GET /api/display/shopping` | Read the shared shopping list. | Wall display | Display token |
| `PATCH /api/display/shopping/items/{id}` | Mark a shopping item picked up or return it to the list. | Wall display | Display token |
| `POST /api/display/shopping/items` | Add an item to the shared shopping list. | Wall display | Display token |
| `GET /api/display/inventory` | Read inventory grouped by area. | Wall display | Display token |
| `PUT /api/display/inventory/items/{id}/stock` | Change an inventory item's quantity. | Wall display | Display token |
| `GET /api/display/commute` | Read this display's shared commute schedule. | Wall display | Display token |
| `GET /api/display/trains` | Read train boards for active shared commute schedules. | Wall display | Display token |
| `GET /api/display/recipes` | Read family recipes. | Wall display | Display token |
| `GET /api/display/recipes/{id}` | Read a recipe with ingredients and steps. | Wall display | Display token |
| `GET /api/display/scratchpad` | Read the shared drawing. | Wall display | Display token |
| `PUT /api/display/scratchpad` | Save a shared drawing with the fixed palette and normalized points. | Wall display | Display token |

All routes use the same JSON shapes as their member API equivalents. Calendar accepts `view=week|day|month` and the corresponding `date` or `month` query parameters. Scratchpad saves have a 1 MiB request limit.

```bash
curl -s "$BASE_URL/api/display/people" -H "Authorization: Bearer $DISPLAY_TOKEN" | jq
curl -s -X POST "$BASE_URL/api/display/chores/<chore-id>/complete" -H "Authorization: Bearer $DISPLAY_TOKEN" -H "Content-Type: application/json" -d '{"userID":"<household-user-id>"}' -w "%{http_code}\n"
curl -s -X POST "$BASE_URL/api/display/shopping/items" -H "Authorization: Bearer $DISPLAY_TOKEN" -H "Content-Type: application/json" -d '{"name":"Apples"}' | jq
curl -s -X PUT "$BASE_URL/api/display/inventory/items/<item-id>/stock" -H "Authorization: Bearer $DISPLAY_TOKEN" -H "Content-Type: application/json" -d '{"quantityDelta":1}' | jq
curl -s "$BASE_URL/api/display/commute" -H "Authorization: Bearer $DISPLAY_TOKEN" | jq
curl -s "$BASE_URL/api/display/trains" -H "Authorization: Bearer $DISPLAY_TOKEN" | jq
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
- **Security:** API token. Multipart form-data, field `avatar`, max 2 MB. Returns updated `User`.

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

### `GET /avatar/{userID}`
- **Usecase:** Serve the selected user's avatar image bytes.
- **Callers:** iOS app and extensions.
- **Security:** Bearer API token.

```bash
curl -s $BASE_URL/avatar/<userID> \
  -H "Authorization: Bearer $API_TOKEN" -o avatar.png
```

### `GET /api/settings`
- **Usecase:** App-wide settings readable by all users: `family_name` and `meal_week_start`. The meal weekday is a lowercase weekday name and defaults to `friday`.
- **Callers:** iOS app settings header.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/settings -H "Authorization: Bearer $API_TOKEN" | jq
```

### Chore definitions and task history

Home uses the dashboard API for current overdue and today's tasks.
Current tasks without a due date appear in Today.
Tasks uses definitions. A definition owns the rule and eligible people.
Each definition has at most one pending or overdue task.

All routes below require a Bearer API token.
Any signed-in family member can create, edit, delete, complete, or skip.

| Route | Behaviour |
|---|---|
| `GET /api/chore-definitions?include_inactive=false` | List active definitions. Set `include_inactive=true` to include ended and deleted definitions. |
| `POST /api/chore-definitions` | Create a definition and its first task. Use the same body as `POST /api/chores`. Return 201. |
| `GET /api/chore-definitions/{id}` | Return the definition and its current task. Return 404 for an unknown definition. |
| `PUT /api/chore-definitions/{id}` | Update setup. Keep the current task's owner and due date. Apply repeat rules to successors. Return 409 for a deleted definition. |
| `DELETE /api/chore-definitions/{id}` | Soft-delete the definition. Remove its task from actionable lists. Keep history. Return 204. |
| `GET /api/chore-history?definition_id={id}&limit=50&cursor={taskID}` | Return resolved tasks, newest first. The definition filter is optional. Limits are 1–100. |
| `POST /api/chores/{id}/skip` | Skip the current task and create its successor atomically. Return 204, or 409 for an already resolved task. |

Definition responses use `{ID, Chore, CurrentTask, Active, DeletedAt}`.
`Chore` contains setup with the definition ID. `CurrentTask` contains the actionable task ID, or null.
History responses use `{Items, NextCursor}`. An empty cursor ends the list.
History adds `skipped` and `superseded` states, plus `ResolvedAt` and `ResolvedByUserID`.
Legacy chore lists exclude these states so older clients still decode them.

Completion and Skip advance rotation from the assigned owner, regardless of the actor.
The server does not skip people with overdue tasks.
Completed and skipped tasks count toward repeat limits. Superseded tasks do not.
Fixed schedules keep their cadence and select the next future date after resolution.
Completion-based schedules calculate the next date from the action time.
Calendar forecasts use `IsForecast=true` and provisional names and owners.
Forecast IDs are display identifiers. They cannot complete or skip a task.

### `GET /api/chores`
- **Usecase:** List chores. Optional filters: `status`, `assigned_to`.
- **Callers:** Legacy clients. The current iOS Tasks tab uses chore definitions.
- **Security:** API token.

```bash
curl -s "$BASE_URL/api/chores?status=pending&assigned_to=<userID>" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

### `POST /api/chores`
- **Usecase:** Create a chore.
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
  series definition. The current owner and due date remain unchanged. Omitted optional rule fields are cleared.
- **Callers:** iOS app edit.
- **Security:** API token.

```bash
curl -s -X PUT $BASE_URL/api/chores/<choreID> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated","description":"","assignees":["user-id-1"],"dueDate":"2026-04-15","recurrenceType":"monthly","recurrenceDayOfMonth":15,"recurrenceCount":6}' | jq
```

### `DELETE /api/chores/{id}`
- **Usecase:** Soft-delete the chore definition. Keep task history.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s -X DELETE $BASE_URL/api/chores/<choreID> -H "Authorization: Bearer $API_TOKEN" -w "%{http_code}\n"
```

### `POST /api/chores/{id}/complete`
- **Usecase:** Mark chore done by the token's user. Spawns next recurrence.
- **Callers:** iOS app, shortcuts.
- **Security:** API token. 409 if the task is no longer outstanding.

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
- **Callers:** iOS app dashboard and iOS Today widget.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/dashboard -H "Authorization: Bearer $API_TOKEN" | jq
```

The response also includes `bin_collections_tomorrow`. It contains the uploaded
bin schedule entries for the household's next calendar day, or an empty array.
The iOS Home screen shows this list only when at least one bin is due tomorrow.

### `GET /api/bin-collections`

- **Usecase:** Read the bin schedule upload status and reminder settings.
- **Callers:** iOS admin settings.
- **Security:** Bearer API token.

```sh
curl -s $BASE_URL/api/bin-collections -H "Authorization: Bearer $API_TOKEN" | jq
```

The response includes `enabled`, `uploaded_at`, `collection_count`, and
`push_available`. The server stores parsed dates and bin labels. It does not
store the original file or the household address, and this response does not
return the stored schedule.

### `POST /api/bin-collections`

- **Usecase:** Validate and replace the household bin schedule from an ICS file.
- **Callers:** iOS admin settings.
- **Security:** Admin bearer token.
- **Body:** `multipart/form-data` with an ICS file in the `schedule` field. The
  file must be under 1 MB and include upcoming bin collection events.

```sh
curl -s $BASE_URL/api/bin-collections -H "Authorization: Bearer $API_TOKEN" \
  -F "schedule=@route-17.ics;type=text/calendar" | jq
```

An invalid upload leaves the last valid schedule in place. A valid upload
replaces it and enables reminders when server push is configured.

### `PATCH /api/bin-collections`

- **Usecase:** Enable or disable household bin reminders.
- **Callers:** iOS admin settings.
- **Security:** Admin bearer token.
- **Body:** `{"enabled":true}`. Enabling requires server push configuration.

```sh
curl -s -X PATCH $BASE_URL/api/bin-collections \
  -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" \
  -d '{"enabled":true}' | jq
```

The server sends one reminder at 7pm household time to each registered device
when one or more bins are due the next day.

### `GET /api/meals?week=YYYY-MM-DD`
- **Usecase:** Meal plans for the seven days starting at `week`. The date is **not**
  snapped to a week boundary — the value you pass *is* the first day, and the range
  is `[week, week+6]`. The iOS app anchors its weeks to the preceding Friday
  before calling this.
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
  Periods use UTC dates, with an inclusive start and an exclusive end.
  Events include any overlap with the period. Month results include the full last day.
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
- **Usecase:** Update `family_name`, `meal_week_start`, or both. Omitted fields stay unchanged. Supply at least one field. The name must be nonempty; the weekday must be one of `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, or `sunday`. Invalid requests return 400 before any field is saved.
- **Callers:** iOS app admin settings — Family name and Meal planning.
- **Security:** API token + admin role. Returns 204.

```bash
curl -s -X PATCH $BASE_URL/api/settings \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"family_name":"The Smiths","meal_week_start":"monday"}' -w "%{http_code}\n"
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

### Calendar feed management

These routes require a Bearer API token and the admin role.
List and create feeds with `GET /api/calendars` and `POST /api/calendars`.
The create body is `{"name":"School","url":"https://example.com/calendar.ics","color":"indigo"}`.
The response includes feed metadata but never includes cached calendar contents.

| Route | Behaviour |
|---|---|
| `GET /api/calendars` | List feeds with `id`, `name`, `url`, `color`, `lastFetchedAt`, and `createdAt`. |
| `POST /api/calendars` | Create a feed. Name and URL are required; color defaults to `indigo`. |
| `PATCH /api/calendars/{id}/color` | Update the color with `{"color":"teal"}`. |
| `POST /api/calendars/{id}/refresh` | Force a fetch and cache refresh. Returns 204. |
| `DELETE /api/calendars/{id}` | Delete a feed. Returns 204. |

Colors: `indigo`, `violet`, `rose`, `red`, `orange`, `amber`, `emerald`, `teal`, `sky`, or `blue`.
Create validates the feed URL with the existing external URL safety checks.

### Database backup and restore

These routes require a Bearer API token and the admin role.
`GET /api/admin/backup` streams a gzip-compressed SQLite database with a
`Content-Disposition` filename. `POST /api/admin/restore` accepts a multipart
file field named `backup` whose filename ends in `.db.gz`. A successful restore
replaces the database and restarts the server.

```bash
curl -s $BASE_URL/api/admin/backup -H "Authorization: Bearer $API_TOKEN" -o backup.db.gz
curl -s -X POST $BASE_URL/api/admin/restore \
  -H "Authorization: Bearer $API_TOKEN" \
  -F "backup=@backup.db.gz"
```

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

## Auth summary

Authentication uses these flows:

| Flow | Routes |
|---|---|
| **Public** | `/health`, `/api/client-config` |
| **OIDC bearer (one-time)** | `POST /api/auth/exchange` |
| **Authed user** (Bearer, via `RequireUser`) | JSON API and media routes |
| **Admin user** (`+ RequireAdmin`) | User/setting/category/calendar-feed management, `/api/tokens*`, `/api/admin/*` |
| **Display credential** (Bearer, via `RequireDisplay`) | Display routes, including shopping purchase updates, meal updates, and scratchpad saves |
| **Pairing challenge** | Public challenge creation and one-time claim; member-authenticated approval |

Chores, recipes, and meals are available to any authenticated user. Admin
management routes require the admin role.

`POST /api/auth/exchange` carries an additional 10-req/min per-IP limit. Pairing
challenge creation and claim routes have their own 5/min and 10/min limits.

## Household location and weather

These endpoints require a Bearer API token. The iOS app calls them.

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

All shopping endpoints require a Bearer API token. All members
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

All endpoints require a Bearer API token.

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

At each time, the server selects current unfinished family chores due today or overdue.
Each alert names up to three assignees and chores, then shows the remaining count.
A tap opens Home and loads current chores.
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
low-stock threshold. Use a Bearer API token as for the other inventory
endpoints. Siri uses this endpoint for stock updates.

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
