# Family Hub API

Read-only REST API for accessing Family Hub data. Useful for integrations, dashboards, and home automation.

## Setup

```bash
export TOKEN=your-token-here
export BASE_URL=https://hub.example.com
```

## Authentication

All API endpoints require a Bearer token. Create tokens from the Admin page.

Tokens are shown only once at creation. If lost, create a new one.

### Token Scopes

| Scope | Access |
|-------|--------|
| `api` | Full read access to all API endpoints |
| `ical` | iCal feed access only (`GET /ical`) |

Use `api`-scoped tokens for integrations. Use `ical`-scoped tokens when subscribing an external calendar app to your family feed.

## Endpoints

### Dashboard

#### `GET /api/dashboard`

Summary stats for the dashboard.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/dashboard" | jq
```

### Chores

#### `GET /api/chores`

List all chores. Supports query parameters:

| Parameter     | Description                                          |
|---------------|------------------------------------------------------|
| `status`      | Filter by status: `pending`, `completed`, `overdue`  |
| `assigned_to` | Filter by user ID                                    |

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/chores" | jq
```

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/chores?status=pending" | jq
```

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/chores?status=overdue&assigned_to=USER_ID" | jq
```

#### `GET /api/chores/{id}`

Get a single chore by ID.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/chores/CHORE_ID" | jq
```

### Users

#### `GET /api/users`

List all users.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/users" | jq
```

#### `GET /api/users/{id}`

Get a single user by ID.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/users/USER_ID" | jq
```

### Categories

#### `GET /api/categories`

List all chore categories.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/categories" | jq
```

### Home Assistant

#### `GET /api/ha/sensors`

Sensor data for Home Assistant integration. Requires `HA_API_TOKEN` to be configured server-side.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/ha/sensors" | jq
```

## Admin Endpoints

The following endpoints require an admin-role token.

### Tokens

#### `POST /api/tokens`

Create a new API token. Returns the token value (shown once only).

#### `DELETE /api/tokens/{id}`

Revoke an API token by ID.
