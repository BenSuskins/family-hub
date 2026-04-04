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

### `GET /api/me`
- **Usecase:** Current user profile for the token holder.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/me -H "Authorization: Bearer $API_TOKEN" | jq
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
- **Usecase:** Create a chore.
- **Callers:** iOS app.
- **Security:** API token. Body JSON: `name` required.

```bash
curl -s -X POST $BASE_URL/api/chores \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Take out trash","description":"Bins by the curb","assignees":["user-id-1"],"dueDate":"2026-04-10","recurrenceType":"weekly"}' | jq
```

### `GET /api/chores/{id}`
- **Usecase:** Fetch a single chore.
- **Callers:** iOS app detail view.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/chores/<choreID> -H "Authorization: Bearer $API_TOKEN" | jq
```

### `PUT /api/chores/{id}`
- **Usecase:** Update chore (name, description, assignees, due date, recurrence).
- **Callers:** iOS app edit.
- **Security:** API token.

```bash
curl -s -X PUT $BASE_URL/api/chores/<choreID> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated","description":"","assignees":["user-id-1"],"dueDate":"2026-04-15","recurrenceType":"none"}' | jq
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
- **Callers:** iOS app dashboard, HA integration.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/dashboard -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/meals?week=YYYY-MM-DD`
- **Usecase:** Meal plans for the week containing the given date (snapped to Monday).
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
- **Usecase:** Scrape recipe fields from a URL (JSON-LD / microdata).
- **Callers:** iOS app "import from URL".
- **Security:** API token. SSRF-hardened (see commit 273c218).

```bash
curl -s -X POST $BASE_URL/api/recipes/extract \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/recipe"}' | jq
```

### `GET /api/recipes`
- **Usecase:** List all recipes.
- **Callers:** iOS app, meal picker.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/recipes -H "Authorization: Bearer $API_TOKEN" | jq
```

### `GET /api/recipes/{id}`
- **Usecase:** Single recipe with ingredients + steps.
- **Callers:** iOS app detail/cook mode.
- **Security:** API token.

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
  -d '{"title":"Pasta","steps":["Boil","Drain"],"ingredients":[{"name":"","items":[{"name":"pasta","quantity":"200g"}]}],"mealType":"dinner","servings":2}' | jq
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

### `GET /api/recipes/{id}/image`
- **Usecase:** Serves recipe image bytes.
- **Callers:** iOS app.
- **Security:** API token.

```bash
curl -s $BASE_URL/api/recipes/<recipeID>/image -H "Authorization: Bearer $API_TOKEN" -o recipe.jpg
```

### `GET /api/calendar?view=month|week|day&date=YYYY-MM-DD|month=YYYY-MM`
- **Usecase:** Unified view: chores + iCal events + meals for the range.
- **Callers:** iOS app calendar tab.
- **Security:** API token.

```bash
curl -s "$BASE_URL/api/calendar?view=month&month=2026-04" \
  -H "Authorization: Bearer $API_TOKEN" | jq
```

---

### Token admin (admin role required)

### `POST /api/tokens`
- **Usecase:** Generate a new API token. Returns raw token **once**.
- **Callers:** Admin UI (`/admin` page).
- **Security:** Session or Bearer + admin role.

```bash
curl -s -X POST $BASE_URL/api/tokens \
  -b "session=$SESSION" \
  -d "name=My iOS token" | jq
```

### `DELETE /api/tokens/{id}`
- **Usecase:** Revoke a token.
- **Callers:** Admin UI.
- **Security:** Session or Bearer + admin role.

```bash
curl -s -X DELETE $BASE_URL/api/tokens/<tokenID> -b "session=$SESSION" -w "%{http_code}\n"
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
