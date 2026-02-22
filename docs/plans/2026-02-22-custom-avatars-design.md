# Custom User Avatars — Design

**Date:** 2026-02-22
**Status:** Approved

## Summary

Allow each user to upload a custom avatar image. Store the image as a data URI in the database. Serve it via a dedicated HTTP endpoint so the rest of the UI can reference it as a normal URL. OIDC-provided avatars remain the default but are not overwritten once a custom avatar is set.

---

## Data Layer

### Migration 011

Add one column to the `users` table:

```sql
ALTER TABLE users ADD COLUMN avatar_data TEXT NOT NULL DEFAULT '';
```

- `avatar_data` — stores the full data URI of the custom-uploaded image (e.g. `data:image/png;base64,...`). Empty when no custom avatar is set.
- `avatar_url` (existing) — continues to hold the effective display URL. For custom avatars it is set to `/avatar/{userID}`; for OIDC-only users it holds the OIDC picture URL or is empty.

No server-side resizing. Images are accepted as-is up to 1 MB.

---

## OIDC Sync Behaviour

`provisionUser` in `services/auth.go` currently overwrites `avatar_url` on every login. Change: skip the avatar update when `avatar_data != ""`. This means once a user uploads a custom avatar, OIDC no longer clobbers it.

---

## Repository

Three new methods on `UserRepository`:

```go
FindAvatarData(ctx context.Context, userID string) (string, error)
UpdateAvatar(ctx context.Context, userID string, dataURI string) error
ClearAvatar(ctx context.Context, userID string) error
```

- `FindAvatarData` — reads only `avatar_data` for a user, without loading the full row (avoids pulling the data URI on every `FindAll`).
- `UpdateAvatar` — sets `avatar_data = dataURI` and `avatar_url = /avatar/{userID}`.
- `ClearAvatar` — clears both `avatar_data` and `avatar_url`. On the user's next login, OIDC will repopulate `avatar_url` from the provider.

---

## Routes and Handlers

New file: `internal/handlers/profile.go`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/profile` | Profile page for the current user |
| `POST` | `/profile/avatar` | Upload a new avatar (multipart, ≤ 1 MB) |
| `DELETE` | `/profile/avatar` | Remove custom avatar, revert to OIDC |
| `GET` | `/avatar/{userID}` | Serve the stored avatar image |

### `GET /avatar/{userID}`

- Reads `avatar_data` via `FindAvatarData`.
- Parses the data URI prefix to extract content-type and base64 payload.
- Decodes base64, writes the raw bytes with the correct `Content-Type` header.
- Returns 404 if `avatar_data` is empty.
- Auth-gated via `RequireAuth` middleware (same as the rest of the app).

### `POST /profile/avatar`

- Parses multipart form, reads the uploaded file.
- Rejects if the decoded size exceeds 1 MB.
- Reads the content-type from the file header.
- Encodes the bytes to base64 and constructs the data URI.
- Calls `UpdateAvatar`.
- Redirects to `/profile`.

### `DELETE /profile/avatar`

- Calls `ClearAvatar` for the current user.
- Redirects to `/profile`.

---

## UI

### `/profile` page

- Displays the current avatar using the existing `UserAvatar` component.
- File upload form (`enctype="multipart/form-data"`, `POST /profile/avatar`).
- "Remove custom avatar" button (visible only when `avatar_data != ""`), sends `DELETE /profile/avatar` via HTMX or a small form.
- Name and email displayed as read-only (sourced from OIDC).

### Navigation

- The existing sidebar bottom user section (avatar + name + logout icon) is wrapped in `<a href="/profile">`, making the whole area a clickable link to the profile page.
- The mobile top bar avatar is also wrapped in `<a href="/profile">`.
- No new nav item is added to the main nav list.

---

## Testing

- **Repository tests** (`users_test.go`): `UpdateAvatar`, `ClearAvatar`, `FindAvatarData` using in-memory SQLite.
- **Handler tests** (`profile_test.go`):
  - Upload: valid image stored correctly.
  - Upload: file > 1 MB rejected with 400.
  - Remove: clears avatar data.
  - Serve (`GET /avatar/{userID}`): returns correct content-type and bytes.
  - Serve: returns 404 when no custom avatar.
- **OIDC sync test**: `provisionUser` does not overwrite `avatar_url` when `avatar_data` is non-empty.
