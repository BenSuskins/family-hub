# Chore Fixes Design — 2026-02-22

## Summary

Four changes to the chores feature:

1. **History tab delete** — admin can delete all completed records for a chore name from the history tab
2. **Remove admin bulk delete** — the "Clear Chore History" maintenance button in admin does nothing visible (deletes assignment records, not chore rows) and is being replaced by per-entry deletes
3. **Calendar future recurrences** — seed real DB rows instead of runtime-projected virtual copies that inherit the wrong status and assignee
4. **Remove Complete button from chores tab** — completion only allowed from the Dashboard

---

## Schema change

New migration `009_chore_series.up.sql`:

```sql
ALTER TABLE chores ADD COLUMN series_id TEXT;
CREATE INDEX idx_chores_series_id ON chores(series_id);
```

`series_id` is a plain string (no FK constraint). All instances in a recurring series share the same value, equal to the original chore's `id`. Non-recurring and existing chores have `NULL`.

---

## Data model & repository

- Add `SeriesID *string` to `models.Chore`
- Update `ChoreRepository` Create, Update, FindAll scan to include `series_id`
- New repository methods:
  - `DeleteFuturePendingBySeries(ctx, seriesID string) error` — `DELETE FROM chores WHERE series_id = ? AND status = 'pending' AND due_date > CURRENT_TIMESTAMP`
  - `FindLastFuturePendingInSeries(ctx, seriesID string) (*models.Chore, error)` — finds the row with the latest `due_date` where `series_id = ?` and `status = 'pending'` and `due_date > now`
  - `DeleteCompletedByName(ctx, name string) error` — `DELETE FROM chores WHERE name = ? AND status = 'completed'` (assignments cascade via existing FK)

---

## Service — seeding

### `SeedFutureOccurrences(ctx, startChore models.Chore, until time.Time) error`

1. Call `FindLastFuturePendingInSeries` to find the furthest-ahead pending instance
2. Use that as the starting point (fall back to `startChore` if none exists)
3. Advance the due date using `advanceToNextOccurrence` repeatedly until `until`
4. For each date: create a chore row (copying Name, Description, RecurrenceType, RecurrenceValue, RecurOnComplete, CategoryID, DueTime, SeriesID from the start chore), then call `AssignNextUser` with the accumulated `LastAssignedIndex` from the previous iteration

### `ensureSeededAhead(ctx, chore models.Chore) error`

Calls `SeedFutureOccurrences(ctx, chore, time.Now().AddDate(1, 0, 0))`.

### `SeedExistingRecurringChores(ctx) error`

One-time startup task. Finds all pending recurring chores where `series_id IS NULL`, sets `series_id = chore.ID`, then calls `SeedFutureOccurrences` on each.

---

## Modified flows

### Chore Create (handler)

1. Create the chore (existing)
2. If recurring: set `chore.SeriesID = &chore.ID`, call `choreRepo.Update`
3. Call `choreService.ensureSeededAhead(ctx, chore)`

### `CompleteChore` (service)

Replace `createNextRecurrence(ctx, chore, now)` with `ensureSeededAhead(ctx, chore)`.

The next pending instance is already in the DB (seeded at creation). `ensureSeededAhead` checks the horizon and creates more if needed (idempotent — starts from the last existing future instance).

### Chore Update (handler)

After updating:
- If `RecurrenceType` or `RecurrenceValue` changed: call `choreRepo.DeleteFuturePendingBySeries(ctx, *chore.SeriesID)`, then `choreService.ensureSeededAhead(ctx, chore)`

### Chore Delete (handler)

Before or after deleting the chore:
- If `chore.SeriesID != nil`: call `choreRepo.DeleteFuturePendingBySeries(ctx, *chore.SeriesID)`

(The chore's own row deletion cascades assignments. Sibling pending rows in the series do not cascade — they must be deleted explicitly.)

### Server startup

Call `choreService.SeedExistingRecurringChores(ctx)` once after DB init. Log errors but don't fail startup.

---

## Calendar handler

Remove the `recurringChores` fetch and `ExpandChoreOccurrences` expansion block. The first query (chores by date range, no status filter) returns all real DB rows including pre-seeded future instances.

`services.ExpandChoreOccurrences` can be deleted along with its tests.

---

## History tab delete

- Pass `User` to `ChoreHistoryContent` and `ChoreHistoryEntry` rendering
- Add a 5th column "Actions" in the desktop grid header for admin users
- Each history row gets a trash icon button (admin only):
  - `hx-post="/chores/history/delete"`
  - `hx-vals` with the chore name
  - `hx-confirm="Delete all history for [name]? This cannot be undone."`
  - `hx-target` points to the row, `hx-swap="outerHTML"` with empty response removes it
- New handler `ChoreHandler.DeleteHistory` calls `choreRepo.DeleteCompletedByName`
- New route (admin-only): `POST /chores/history/delete`

---

## Admin page cleanup

- Remove the Maintenance section (Clear Chore History button) from `admin.templ`
- Remove `DeleteChoreHistory` method from `admin.go`
- Remove route `POST /admin/chores/history/delete` from `server.go`

---

## Chores tab — remove Complete button

In `templates/pages/chores.templ`, remove the Complete button block from `ChoreRow` (the `if chore.Status != models.ChoreStatusCompleted && chore.AssignedToUserID != nil && *chore.AssignedToUserID == user.ID` block). The `POST /chores/{id}/complete` route and handler remain for the Dashboard.
