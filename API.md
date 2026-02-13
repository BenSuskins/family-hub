# Family Hub API

Read-only REST API for accessing Family Hub data. Useful for integrations, dashboards, and home automation.

## Authentication

All API endpoints require a Bearer token. Create tokens from the Admin page.

```
Authorization: Bearer <your-token>
```

Tokens are shown only once at creation. If lost, create a new one.

## Endpoints

### Dashboard

#### `GET /api/dashboard`

Summary stats for the dashboard.

```json
{
  "chores_due_today": 3,
  "chores_overdue": 1,
  "upcoming_events": 5
}
```

### Chores

#### `GET /api/chores`

List all chores. Supports query parameters:

| Parameter     | Description                                          |
|---------------|------------------------------------------------------|
| `status`      | Filter by status: `pending`, `completed`, `overdue`  |
| `assigned_to` | Filter by user ID                                    |

Example: `GET /api/chores?status=pending&assigned_to=abc123`

#### `GET /api/chores/{id}`

Get a single chore by ID.

### Events

#### `GET /api/events`

List all events. Supports query parameters:

| Parameter | Description                          |
|-----------|--------------------------------------|
| `after`   | Only events starting after (YYYY-MM-DD)  |
| `before`  | Only events starting before (YYYY-MM-DD) |

Example: `GET /api/events?after=2025-02-01&before=2025-02-28`

#### `GET /api/events/{id}`

Get a single event by ID.

### Users

#### `GET /api/users`

List all users.

#### `GET /api/users/{id}`

Get a single user by ID.

### Categories

#### `GET /api/categories`

List all chore categories.

## Example

```bash
curl -H "Authorization: Bearer your-token-here" \
  https://hub.example.com/api/chores?status=overdue
```
