# Family Hub API

Read-only REST API for accessing Family Hub data. Useful for integrations, dashboards, and home automation.

## Setup

```bash
export TOKEN=your-token-here
export BASE_URL=https://hub.suskins.co.uk
```

## Authentication

All API endpoints require a Bearer token. Create tokens from the Admin page.

Tokens are shown only once at creation. If lost, create a new one.

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

### Events

#### `GET /api/events`

List all events. Supports query parameters:

| Parameter | Description                              |
|-----------|------------------------------------------|
| `after`   | Only events starting after (YYYY-MM-DD)  |
| `before`  | Only events starting before (YYYY-MM-DD) |

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/events" | jq
```

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/events?after=2025-02-01&before=2025-02-28" | jq
```

#### `GET /api/events/{id}`

Get a single event by ID.

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/events/EVENT_ID" | jq
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
