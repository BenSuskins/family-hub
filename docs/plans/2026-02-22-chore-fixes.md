# Chore Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix four chore-related issues: seed future recurring chore instances in the DB instead of projecting them at render time; add per-entry delete in history tab (admin only); remove the broken admin bulk-delete button; remove the Complete button from the chores list.

**Architecture:** Add a `series_id` column linking recurring chore instances. A new `SeedFutureOccurrences` service method seeds 12 months of pending instances with proper round-robin assignment at creation and completion time. The calendar handler drops its runtime expansion code and queries the DB directly. The history tab gains an admin-only delete button per row. The chores tab `ChoreRow` loses its Complete button.

**Tech Stack:** Go, chi, templ, HTMX, SQLite (modernc.org/sqlite). Templ generates `*_templ.go` from `.templ` files — run `$(go env GOPATH)/bin/templ generate ./...` after every `.templ` change. Run tests with `make test`.

---

### Task 1: Migration — add series_id column

**Files:**
- Create: `internal/database/migrations/009_chore_series.up.sql`

**Step 1: Create the migration file**

```sql
ALTER TABLE chores ADD COLUMN series_id TEXT;
CREATE INDEX idx_chores_series_id ON chores(series_id);
```

**Step 2: Verify migration runs in tests**

Run: `make test`
Expected: all existing tests pass (migration runs automatically via `testutil.NewTestDatabase`)

**Step 3: Commit**

```bash
git add internal/database/migrations/009_chore_series.up.sql
git commit -m "feat: add series_id column to chores table"
```

---

### Task 2: Model — add SeriesID field

**Files:**
- Modify: `internal/models/models.go` — `Chore` struct

**Step 1: Add the field**

In `models.go`, add `SeriesID *string` to the `Chore` struct after `RecurOnComplete`:

```go
type Chore struct {
    ID              string
    Name            string
    Description     string
    CreatedByUserID string
    CategoryID      *string

    AssignedToUserID  *string
    LastAssignedIndex int
    EligibleAssignees []string

    DueDate *time.Time
    DueTime *string

    RecurrenceType  RecurrenceType
    RecurrenceValue string
    RecurOnComplete bool
    SeriesID        *string  // ← add this

    Status            ChoreStatus
    CompletedAt       *time.Time
    CompletedByUserID *string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Step 2: Verify compilation**

Run: `go build ./...`
Expected: compiles cleanly

**Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add SeriesID field to Chore model"
```

---

### Task 3: Repository — thread series_id through all SQL

**Files:**
- Modify: `internal/repository/chores.go`

All SELECT queries must include `series_id`. `Create` must INSERT it. `Update` must SET it. The `scanChores` helper and the inline `FindByID` scan both need the new column.

**Step 1: Update FindByID**

Replace the SELECT and Scan in `FindByID`:

```go
func (repository *SQLiteChoreRepository) FindByID(ctx context.Context, id string) (models.Chore, error) {
    var chore models.Chore
    err := repository.database.QueryRowContext(ctx,
        `SELECT id, name, description, created_by_user_id, category_id,
            assigned_to_user_id, last_assigned_index,
            due_date, due_time,
            recurrence_type, recurrence_value, recur_on_complete, series_id,
            status, completed_at, completed_by_user_id,
            created_at, updated_at
        FROM chores WHERE id = ?`, id,
    ).Scan(
        &chore.ID, &chore.Name, &chore.Description, &chore.CreatedByUserID, &chore.CategoryID,
        &chore.AssignedToUserID, &chore.LastAssignedIndex,
        &chore.DueDate, &chore.DueTime,
        &chore.RecurrenceType, &chore.RecurrenceValue, &chore.RecurOnComplete, &chore.SeriesID,
        &chore.Status, &chore.CompletedAt, &chore.CompletedByUserID,
        &chore.CreatedAt, &chore.UpdatedAt,
    )
    if err != nil {
        return models.Chore{}, fmt.Errorf("finding chore by id: %w", err)
    }
    return chore, nil
}
```

**Step 2: Update FindAll SELECT**

```go
query := `SELECT id, name, description, created_by_user_id, category_id,
    assigned_to_user_id, last_assigned_index,
    due_date, due_time,
    recurrence_type, recurrence_value, recur_on_complete, series_id,
    status, completed_at, completed_by_user_id,
    created_at, updated_at
FROM chores WHERE 1=1`
```

**Step 3: Update FindOverdueChores and FindDueToday SELECT**

In `FindOverdueChores`, update the inline SELECT to include `series_id` (same column list as above). Same for `FindDueToday`.

**Step 4: Update scanChores**

```go
func scanChores(rows *sql.Rows) ([]models.Chore, error) {
    var chores []models.Chore
    for rows.Next() {
        var chore models.Chore
        if err := rows.Scan(
            &chore.ID, &chore.Name, &chore.Description, &chore.CreatedByUserID, &chore.CategoryID,
            &chore.AssignedToUserID, &chore.LastAssignedIndex,
            &chore.DueDate, &chore.DueTime,
            &chore.RecurrenceType, &chore.RecurrenceValue, &chore.RecurOnComplete, &chore.SeriesID,
            &chore.Status, &chore.CompletedAt, &chore.CompletedByUserID,
            &chore.CreatedAt, &chore.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("scanning chore: %w", err)
        }
        chores = append(chores, chore)
    }
    return chores, rows.Err()
}
```

**Step 5: Update Create**

```go
_, err := repository.database.ExecContext(ctx,
    `INSERT INTO chores (id, name, description, created_by_user_id, category_id,
        assigned_to_user_id, last_assigned_index,
        due_date, due_time,
        recurrence_type, recurrence_value, recur_on_complete, series_id,
        status, completed_at, completed_by_user_id,
        created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    chore.ID, chore.Name, chore.Description, chore.CreatedByUserID, chore.CategoryID,
    chore.AssignedToUserID, chore.LastAssignedIndex,
    chore.DueDate, chore.DueTime,
    chore.RecurrenceType, chore.RecurrenceValue, chore.RecurOnComplete, chore.SeriesID,
    chore.Status, chore.CompletedAt, chore.CompletedByUserID,
    chore.CreatedAt, chore.UpdatedAt,
)
```

**Step 6: Update Update**

```go
_, err := repository.database.ExecContext(ctx,
    `UPDATE chores SET name = ?, description = ?, category_id = ?,
        assigned_to_user_id = ?, last_assigned_index = ?,
        due_date = ?, due_time = ?,
        recurrence_type = ?, recurrence_value = ?, recur_on_complete = ?, series_id = ?,
        status = ?, completed_at = ?, completed_by_user_id = ?,
        updated_at = ?
    WHERE id = ?`,
    chore.Name, chore.Description, chore.CategoryID,
    chore.AssignedToUserID, chore.LastAssignedIndex,
    chore.DueDate, chore.DueTime,
    chore.RecurrenceType, chore.RecurrenceValue, chore.RecurOnComplete, chore.SeriesID,
    chore.Status, chore.CompletedAt, chore.CompletedByUserID,
    chore.UpdatedAt, chore.ID,
)
```

**Step 7: Run tests**

Run: `make test`
Expected: all tests pass

**Step 8: Commit**

```bash
git add internal/repository/chores.go
git commit -m "feat: thread series_id through chore repository SQL"
```

---

### Task 4: Repository — new methods + tests

**Files:**
- Modify: `internal/repository/chores.go` — interface + implementation
- Modify: `internal/repository/chores_test.go` — new test functions

**Step 1: Add methods to ChoreRepository interface**

```go
type ChoreRepository interface {
    // ... existing methods ...
    DeleteFuturePendingBySeries(ctx context.Context, seriesID string) error
    FindLastFuturePendingInSeries(ctx context.Context, seriesID string) (*models.Chore, error)
    DeleteCompletedByName(ctx context.Context, name string) error
}
```

**Step 2: Write failing tests**

In `internal/repository/chores_test.go`, add:

```go
func TestChoreRepository_DeleteFuturePendingBySeries(t *testing.T) {
    db := testutil.NewTestDatabase(t)
    repo := repository.NewChoreRepository(db)
    userRepo := repository.NewUserRepository(db)
    ctx := context.Background()

    user, _ := userRepo.Create(ctx, models.User{OIDCSubject: "sub1", Email: "a@b.com", Name: "A", Role: models.RoleMember})
    seriesID := "series-1"

    past := time.Now().AddDate(0, 0, -1)
    future1 := time.Now().AddDate(0, 0, 7)
    future2 := time.Now().AddDate(0, 0, 14)

    completed, _ := repo.Create(ctx, models.Chore{
        Name: "Clean", CreatedByUserID: user.ID, SeriesID: &seriesID,
        DueDate: &past, Status: models.ChoreStatusCompleted,
    })
    _ = completed

    pending1, _ := repo.Create(ctx, models.Chore{
        Name: "Clean", CreatedByUserID: user.ID, SeriesID: &seriesID,
        DueDate: &future1, Status: models.ChoreStatusPending,
    })
    pending2, _ := repo.Create(ctx, models.Chore{
        Name: "Clean", CreatedByUserID: user.ID, SeriesID: &seriesID,
        DueDate: &future2, Status: models.ChoreStatusPending,
    })
    _ = pending1
    _ = pending2

    if err := repo.DeleteFuturePendingBySeries(ctx, seriesID); err != nil {
        t.Fatalf("DeleteFuturePendingBySeries: %v", err)
    }

    remaining, err := repo.FindAll(ctx, repository.ChoreFilter{})
    if err != nil {
        t.Fatalf("FindAll: %v", err)
    }
    if len(remaining) != 1 {
        t.Errorf("want 1 chore remaining (completed), got %d", len(remaining))
    }
    if remaining[0].Status != models.ChoreStatusCompleted {
        t.Errorf("remaining chore should be completed, got %s", remaining[0].Status)
    }
}

func TestChoreRepository_FindLastFuturePendingInSeries(t *testing.T) {
    db := testutil.NewTestDatabase(t)
    repo := repository.NewChoreRepository(db)
    userRepo := repository.NewUserRepository(db)
    ctx := context.Background()

    user, _ := userRepo.Create(ctx, models.User{OIDCSubject: "sub2", Email: "b@b.com", Name: "B", Role: models.RoleMember})
    seriesID := "series-2"

    t.Run("returns nil when no future pending instances", func(t *testing.T) {
        result, err := repo.FindLastFuturePendingInSeries(ctx, seriesID)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if result != nil {
            t.Errorf("want nil, got %+v", result)
        }
    })

    future1 := time.Now().AddDate(0, 0, 7)
    future2 := time.Now().AddDate(0, 0, 14)
    repo.Create(ctx, models.Chore{
        Name: "Clean", CreatedByUserID: user.ID, SeriesID: &seriesID,
        DueDate: &future1, Status: models.ChoreStatusPending,
    })
    repo.Create(ctx, models.Chore{
        Name: "Clean", CreatedByUserID: user.ID, SeriesID: &seriesID,
        DueDate: &future2, Status: models.ChoreStatusPending,
    })

    t.Run("returns furthest-ahead pending instance", func(t *testing.T) {
        result, err := repo.FindLastFuturePendingInSeries(ctx, seriesID)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if result == nil {
            t.Fatal("want a chore, got nil")
        }
        if !result.DueDate.Equal(future2) {
            t.Errorf("want due date %v, got %v", future2, result.DueDate)
        }
    })
}

func TestChoreRepository_DeleteCompletedByName(t *testing.T) {
    db := testutil.NewTestDatabase(t)
    repo := repository.NewChoreRepository(db)
    userRepo := repository.NewUserRepository(db)
    ctx := context.Background()

    user, _ := userRepo.Create(ctx, models.User{OIDCSubject: "sub3", Email: "c@b.com", Name: "C", Role: models.RoleMember})
    past := time.Now().AddDate(0, 0, -1)

    repo.Create(ctx, models.Chore{Name: "Dishes", CreatedByUserID: user.ID, DueDate: &past, Status: models.ChoreStatusCompleted})
    repo.Create(ctx, models.Chore{Name: "Dishes", CreatedByUserID: user.ID, DueDate: &past, Status: models.ChoreStatusCompleted})
    pending := time.Now().AddDate(0, 0, 1)
    repo.Create(ctx, models.Chore{Name: "Dishes", CreatedByUserID: user.ID, DueDate: &pending, Status: models.ChoreStatusPending})
    repo.Create(ctx, models.Chore{Name: "Laundry", CreatedByUserID: user.ID, DueDate: &past, Status: models.ChoreStatusCompleted})

    if err := repo.DeleteCompletedByName(ctx, "Dishes"); err != nil {
        t.Fatalf("DeleteCompletedByName: %v", err)
    }

    all, _ := repo.FindAll(ctx, repository.ChoreFilter{})
    if len(all) != 2 {
        t.Errorf("want 2 remaining (1 pending Dishes + 1 completed Laundry), got %d", len(all))
    }
}
```

**Step 3: Run tests to confirm they fail**

Run: `make test`
Expected: FAIL — methods not defined on SQLiteChoreRepository

**Step 4: Implement the three methods**

Add to `internal/repository/chores.go`:

```go
func (repository *SQLiteChoreRepository) DeleteFuturePendingBySeries(ctx context.Context, seriesID string) error {
    _, err := repository.database.ExecContext(ctx,
        `DELETE FROM chores WHERE series_id = ? AND status = 'pending' AND due_date > CURRENT_TIMESTAMP`,
        seriesID,
    )
    if err != nil {
        return fmt.Errorf("deleting future pending by series: %w", err)
    }
    return nil
}

func (repository *SQLiteChoreRepository) FindLastFuturePendingInSeries(ctx context.Context, seriesID string) (*models.Chore, error) {
    rows, err := repository.database.QueryContext(ctx,
        `SELECT id, name, description, created_by_user_id, category_id,
            assigned_to_user_id, last_assigned_index,
            due_date, due_time,
            recurrence_type, recurrence_value, recur_on_complete, series_id,
            status, completed_at, completed_by_user_id,
            created_at, updated_at
        FROM chores
        WHERE series_id = ? AND status = 'pending' AND due_date > CURRENT_TIMESTAMP
        ORDER BY due_date DESC
        LIMIT 1`,
        seriesID,
    )
    if err != nil {
        return nil, fmt.Errorf("finding last future pending in series: %w", err)
    }
    defer rows.Close()

    chores, err := scanChores(rows)
    if err != nil {
        return nil, err
    }
    if len(chores) == 0 {
        return nil, nil
    }
    return &chores[0], nil
}

func (repository *SQLiteChoreRepository) DeleteCompletedByName(ctx context.Context, name string) error {
    _, err := repository.database.ExecContext(ctx,
        `DELETE FROM chores WHERE name = ? AND status = 'completed'`,
        name,
    )
    if err != nil {
        return fmt.Errorf("deleting completed chores by name: %w", err)
    }
    return nil
}
```

**Step 5: Run tests**

Run: `make test`
Expected: all tests pass

**Step 6: Commit**

```bash
git add internal/repository/chores.go internal/repository/chores_test.go
git commit -m "feat: add series-based and name-based chore deletion methods"
```

---

### Task 5: Service — SeedFutureOccurrences

**Files:**
- Modify: `internal/services/chores.go`
- Modify: `internal/services/chores_test.go`

**Step 1: Write the failing test**

Add to `internal/services/chores_test.go`:

```go
func TestChoreService_SeedFutureOccurrences(t *testing.T) {
    service, choreRepo, _, userRepo := setupChoreService(t)
    ctx := context.Background()
    users := createUsers(t, userRepo, 2)

    now := time.Now()
    base := now.AddDate(0, 0, 1) // due tomorrow
    chore, _ := choreRepo.Create(ctx, models.Chore{
        Name:            "Weekly Cleanup",
        CreatedByUserID: users[0].ID,
        RecurrenceType:  models.RecurrenceWeekly,
        RecurrenceValue: `{"interval":1}`,
        DueDate:         &base,
        Status:          models.ChoreStatusPending,
        LastAssignedIndex: -1,
    })
    // Set series_id = chore.ID
    seriesID := chore.ID
    chore.SeriesID = &seriesID
    choreRepo.Update(ctx, chore)

    until := now.AddDate(0, 0, 28) // seed 4 weeks ahead
    if err := service.SeedFutureOccurrences(ctx, chore, until); err != nil {
        t.Fatalf("SeedFutureOccurrences: %v", err)
    }

    all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
        Statuses: []models.ChoreStatus{models.ChoreStatusPending},
    })
    // base chore + 3 seeded instances (4 weeks - 1 already exists = 3 new ones)
    if len(all) < 3 {
        t.Errorf("want at least 3 pending chores, got %d", len(all))
    }
    for _, c := range all {
        if c.ID == chore.ID {
            continue // skip base
        }
        if c.Status != models.ChoreStatusPending {
            t.Errorf("seeded chore %s has status %s, want pending", c.ID, c.Status)
        }
        if c.SeriesID == nil || *c.SeriesID != seriesID {
            t.Errorf("seeded chore %s has wrong series_id", c.ID)
        }
        if c.AssignedToUserID == nil {
            t.Errorf("seeded chore %s has no assignee", c.ID)
        }
    }
}

func TestChoreService_SeedFutureOccurrences_SkipsRecurOnComplete(t *testing.T) {
    service, choreRepo, _, userRepo := setupChoreService(t)
    ctx := context.Background()
    users := createUsers(t, userRepo, 1)

    now := time.Now()
    base := now.AddDate(0, 0, 1)
    chore, _ := choreRepo.Create(ctx, models.Chore{
        Name:            "Ad-hoc",
        CreatedByUserID: users[0].ID,
        RecurrenceType:  models.RecurrenceWeekly,
        RecurrenceValue: `{"interval":1}`,
        RecurOnComplete: true,
        DueDate:         &base,
        Status:          models.ChoreStatusPending,
    })

    if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 1, 0)); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{})
    if len(all) != 1 {
        t.Errorf("RecurOnComplete chores should not be seeded; want 1 chore, got %d", len(all))
    }
}

func TestChoreService_SeedFutureOccurrences_Idempotent(t *testing.T) {
    service, choreRepo, _, userRepo := setupChoreService(t)
    ctx := context.Background()
    users := createUsers(t, userRepo, 1)

    now := time.Now()
    base := now.AddDate(0, 0, 1)
    chore, _ := choreRepo.Create(ctx, models.Chore{
        Name:            "Daily",
        CreatedByUserID: users[0].ID,
        RecurrenceType:  models.RecurrenceDaily,
        DueDate:         &base,
        Status:          models.ChoreStatusPending,
        LastAssignedIndex: -1,
    })
    seriesID := chore.ID
    chore.SeriesID = &seriesID
    choreRepo.Update(ctx, chore)

    until := now.AddDate(0, 0, 7)
    service.SeedFutureOccurrences(ctx, chore, until)
    firstCount, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})

    // Seed again — should not create duplicates
    service.SeedFutureOccurrences(ctx, chore, until)
    secondCount, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})

    if len(firstCount) != len(secondCount) {
        t.Errorf("seeding twice should be idempotent: first=%d second=%d", len(firstCount), len(secondCount))
    }
}
```

**Step 2: Run tests to confirm they fail**

Run: `make test`
Expected: FAIL — `SeedFutureOccurrences` not defined

**Step 3: Implement SeedFutureOccurrences**

Add to `internal/services/chores.go` (also uses `advanceToNextOccurrence` and `parseConfig` from `recurrence.go` in the same package):

```go
// SeedFutureOccurrences creates pending chore instances from the chore's series ahead to `until`.
// No-op for RecurOnComplete chores (can't predict completion dates) or chores without a DueDate.
// Idempotent: starts from the last existing future pending instance in the series.
func (service *ChoreService) SeedFutureOccurrences(ctx context.Context, chore models.Chore, until time.Time) error {
    if chore.RecurrenceType == models.RecurrenceNone || chore.RecurOnComplete {
        return nil
    }
    if chore.DueDate == nil {
        return nil
    }

    // Ensure series_id is set (handles legacy chores completed for the first time)
    if chore.SeriesID == nil {
        chore.SeriesID = &chore.ID
        if err := service.choreRepo.Update(ctx, chore); err != nil {
            return fmt.Errorf("setting series_id: %w", err)
        }
    }

    // Find the furthest-ahead pending instance — seed from there
    startChore := chore
    lastFuture, err := service.choreRepo.FindLastFuturePendingInSeries(ctx, *chore.SeriesID)
    if err != nil {
        return fmt.Errorf("finding last future pending: %w", err)
    }
    if lastFuture != nil {
        startChore = *lastFuture
    }

    config, err := parseConfig(chore.RecurrenceValue)
    if err != nil {
        return fmt.Errorf("parsing recurrence config: %w", err)
    }

    current := *startChore.DueDate
    currentChore := startChore
    now := time.Now()

    for i := 0; i < maxExpansionIterations; i++ {
        nextDate := advanceToNextOccurrence(current, chore.RecurrenceType, config)
        if !nextDate.Before(until) {
            break
        }
        current = nextDate

        if nextDate.Before(now) {
            continue // skip dates already in the past
        }

        newChore := models.Chore{
            Name:              chore.Name,
            Description:       chore.Description,
            CreatedByUserID:   chore.CreatedByUserID,
            CategoryID:        chore.CategoryID,
            SeriesID:          chore.SeriesID,
            LastAssignedIndex: currentChore.LastAssignedIndex,
            DueDate:           &nextDate,
            DueTime:           chore.DueTime,
            RecurrenceType:    chore.RecurrenceType,
            RecurrenceValue:   chore.RecurrenceValue,
            RecurOnComplete:   chore.RecurOnComplete,
            Status:            models.ChoreStatusPending,
        }

        created, err := service.choreRepo.Create(ctx, newChore)
        if err != nil {
            return fmt.Errorf("creating seeded chore instance: %w", err)
        }

        eligibleIDs, err := service.choreRepo.GetEligibleAssignees(ctx, chore.ID)
        if err == nil && len(eligibleIDs) > 0 {
            if err := service.choreRepo.SetEligibleAssignees(ctx, created.ID, eligibleIDs); err != nil {
                return fmt.Errorf("copying eligible assignees: %w", err)
            }
        }

        assigned, err := service.AssignNextUser(ctx, created)
        if err != nil {
            return fmt.Errorf("assigning seeded chore: %w", err)
        }
        currentChore = assigned
    }

    return nil
}
```

**Step 4: Run tests**

Run: `make test`
Expected: all tests pass

**Step 5: Commit**

```bash
git add internal/services/chores.go internal/services/chores_test.go
git commit -m "feat: add SeedFutureOccurrences service method"
```

---

### Task 6: Service — SeedExistingRecurringChores + modify CompleteChore + update createNextRecurrence

**Files:**
- Modify: `internal/services/chores.go`
- Modify: `internal/services/chores_test.go`

**Step 1: Write tests**

```go
func TestChoreService_CompleteChore_SeedsAhead(t *testing.T) {
    service, choreRepo, _, userRepo := setupChoreService(t)
    ctx := context.Background()
    users := createUsers(t, userRepo, 2)

    now := time.Now()
    dueDate := now.AddDate(0, 0, -1) // overdue
    seriesID := "test-series"
    chore, _ := choreRepo.Create(ctx, models.Chore{
        Name:            "Weekly",
        CreatedByUserID: users[0].ID,
        RecurrenceType:  models.RecurrenceWeekly,
        RecurrenceValue: `{"interval":1}`,
        DueDate:         &dueDate,
        Status:          models.ChoreStatusPending,
        SeriesID:        &seriesID,
        LastAssignedIndex: -1,
    })
    chore, _ = service.AssignNextUser(ctx, chore)

    if err := service.CompleteChore(ctx, chore.ID, users[0].ID); err != nil {
        t.Fatalf("CompleteChore: %v", err)
    }

    pending, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
        Statuses: []models.ChoreStatus{models.ChoreStatusPending},
    })
    if len(pending) == 0 {
        t.Error("completing a recurring chore should seed future pending instances")
    }
}

func TestChoreService_SeedExistingRecurringChores(t *testing.T) {
    service, choreRepo, _, userRepo := setupChoreService(t)
    ctx := context.Background()
    users := createUsers(t, userRepo, 1)

    now := time.Now()
    dueDate := now.AddDate(0, 0, 1)
    // Create a legacy recurring chore with no series_id
    chore, _ := choreRepo.Create(ctx, models.Chore{
        Name:            "Legacy Weekly",
        CreatedByUserID: users[0].ID,
        RecurrenceType:  models.RecurrenceWeekly,
        RecurrenceValue: `{"interval":1}`,
        DueDate:         &dueDate,
        Status:          models.ChoreStatusPending,
        LastAssignedIndex: -1,
    })
    if chore.SeriesID != nil {
        t.Fatal("freshly created chore should have nil series_id for this test")
    }

    until := now.AddDate(0, 0, 21) // 3 weeks
    if err := service.SeedExistingRecurringChores(ctx, until); err != nil {
        t.Fatalf("SeedExistingRecurringChores: %v", err)
    }

    all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
    if len(all) < 3 {
        t.Errorf("want at least 3 pending instances (1 + 2 seeded), got %d", len(all))
    }

    // Verify series_id was set on the original chore
    updated, _ := choreRepo.FindByID(ctx, chore.ID)
    if updated.SeriesID == nil {
        t.Error("original chore should have series_id set after seeding")
    }
}
```

**Step 2: Run tests to confirm they fail**

Run: `make test`
Expected: FAIL

**Step 3: Modify CompleteChore**

Replace the `createNextRecurrence` call at the end of `CompleteChore` with:

```go
if chore.RecurrenceType != models.RecurrenceNone {
    if chore.RecurOnComplete {
        if err := service.createNextRecurrence(ctx, chore, now); err != nil {
            return fmt.Errorf("creating next recurrence: %w", err)
        }
    } else {
        if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(1, 0, 0)); err != nil {
            return fmt.Errorf("seeding future occurrences: %w", err)
        }
    }
}
```

**Step 4: Update createNextRecurrence to copy SeriesID**

In `createNextRecurrence`, add `SeriesID` to `newChore`:

```go
newChore := models.Chore{
    Name:              chore.Name,
    Description:       chore.Description,
    CreatedByUserID:   chore.CreatedByUserID,
    CategoryID:        chore.CategoryID,
    SeriesID:          chore.SeriesID,  // ← add this
    LastAssignedIndex: chore.LastAssignedIndex,
    DueDate:           nextDueDate,
    DueTime:           chore.DueTime,
    RecurrenceType:    chore.RecurrenceType,
    RecurrenceValue:   chore.RecurrenceValue,
    RecurOnComplete:   chore.RecurOnComplete,
    Status:            models.ChoreStatusPending,
}
```

If `chore.SeriesID == nil` (legacy chore), also set it before creating:

```go
if chore.SeriesID == nil {
    chore.SeriesID = &chore.ID
    if err := service.choreRepo.Update(ctx, chore); err != nil {
        return fmt.Errorf("setting series_id on legacy chore: %w", err)
    }
}
newChore.SeriesID = chore.SeriesID
```

**Step 5: Implement SeedExistingRecurringChores**

Add to `internal/services/chores.go`:

```go
// SeedExistingRecurringChores seeds future instances for all pending recurring chores
// that have not yet been assigned a series_id. Call once at server startup.
func (service *ChoreService) SeedExistingRecurringChores(ctx context.Context, until time.Time) error {
    chores, err := service.choreRepo.FindAll(ctx, repository.ChoreFilter{
        Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
        RecurrenceTypes: []models.RecurrenceType{
            models.RecurrenceDaily,
            models.RecurrenceWeekly,
            models.RecurrenceMonthly,
            models.RecurrenceCustom,
            models.RecurrenceCalendar,
        },
    })
    if err != nil {
        return fmt.Errorf("finding chores to seed: %w", err)
    }

    for _, chore := range chores {
        if chore.SeriesID != nil {
            continue
        }
        if chore.RecurOnComplete || chore.DueDate == nil {
            continue
        }
        if err := service.SeedFutureOccurrences(ctx, chore, until); err != nil {
            slog.Error("seeding existing chore", "chore_id", chore.ID, "error", err)
        }
    }
    return nil
}
```

Note: add `"log/slog"` to the import if not already present.

**Step 6: Run tests**

Run: `make test`
Expected: all tests pass

**Step 7: Commit**

```bash
git add internal/services/chores.go internal/services/chores_test.go
git commit -m "feat: seed future occurrences on completion and backfill existing chores"
```

---

### Task 7: Handler — modify Create to seed on creation

**Files:**
- Modify: `internal/handlers/chores.go`

**Step 1: Update the Create handler**

After the `AssignNextUser` call, add seeding. The Create handler in `handlers/chores.go` currently ends with:

```go
if _, err := handler.choreService.AssignNextUser(ctx, created); err != nil {
    slog.Error("assigning chore", "error", err)
}

http.Redirect(w, r, "/chores", http.StatusFound)
```

Replace with:

```go
assigned, err := handler.choreService.AssignNextUser(ctx, created)
if err != nil {
    slog.Error("assigning chore", "error", err)
}

if created.RecurrenceType != models.RecurrenceNone && !created.RecurOnComplete {
    // Set series_id = chore's own ID, then seed 12 months ahead
    seriesID := created.ID
    assigned.SeriesID = &seriesID
    if err := handler.choreRepo.Update(ctx, assigned); err != nil {
        slog.Error("setting series_id on new chore", "error", err)
    } else {
        if err := handler.choreService.SeedFutureOccurrences(ctx, assigned, time.Now().AddDate(1, 0, 0)); err != nil {
            slog.Error("seeding future occurrences for new chore", "error", err)
        }
    }
}

http.Redirect(w, r, "/chores", http.StatusFound)
```

Also ensure `"time"` is in the imports (it already is).

**Step 2: Build to verify no compile errors**

Run: `go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add internal/handlers/chores.go
git commit -m "feat: seed future occurrences when creating a recurring chore"
```

---

### Task 8: Handler — modify Update to delete+reseed on recurrence change

**Files:**
- Modify: `internal/handlers/chores.go`

**Step 1: Update the Update handler**

The Update handler in `handlers/chores.go` currently ends with:

```go
if err := handler.choreRepo.Update(ctx, chore); err != nil { ... }
// ... SetEligibleAssignees ...
http.Redirect(w, r, "/chores", http.StatusFound)
```

Before the redirect, after updating the chore, add:

```go
// If recurrence settings changed, delete stale future instances and re-seed
oldRecurrenceType := models.RecurrenceType(r.FormValue("recurrence_type"))
oldRecurrenceValue := buildRecurrenceValue(oldRecurrenceType, r)
recurrenceChanged := chore.RecurrenceType != oldRecurrenceType || chore.RecurrenceValue != oldRecurrenceValue
```

Wait — the problem is that by the time we're in the Update handler, `chore` has already been mutated with the new form values. We need to capture the *old* values before overwriting them.

Refactor the Update handler to save the old recurrence settings before modifying:

```go
func (handler *ChoreHandler) Update(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    choreID := chi.URLParam(r, "id")

    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form", http.StatusBadRequest)
        return
    }

    chore, err := handler.choreRepo.FindByID(ctx, choreID)
    if err != nil {
        http.NotFound(w, r)
        return
    }

    // Capture old recurrence settings before overwriting
    oldRecurrenceType := chore.RecurrenceType
    oldRecurrenceValue := chore.RecurrenceValue

    recurrenceType := models.RecurrenceType(r.FormValue("recurrence_type"))
    recurrenceValue := buildRecurrenceValue(recurrenceType, r)

    chore.Name = r.FormValue("name")
    chore.Description = r.FormValue("description")
    chore.RecurrenceType = recurrenceType
    chore.RecurrenceValue = recurrenceValue
    chore.RecurOnComplete = r.FormValue("recur_on_complete") == "on"

    // ... rest of existing field updates (CategoryID, DueDate, DueTime) unchanged ...

    if err := handler.choreRepo.Update(ctx, chore); err != nil {
        slog.Error("updating chore", "error", err)
        http.Error(w, "Error updating chore", http.StatusInternalServerError)
        return
    }

    // ... existing SetEligibleAssignees logic unchanged ...

    // If recurrence changed, delete stale future instances and re-seed
    recurrenceChanged := recurrenceType != oldRecurrenceType || recurrenceValue != oldRecurrenceValue
    if recurrenceChanged && chore.SeriesID != nil && !chore.RecurOnComplete {
        if err := handler.choreRepo.DeleteFuturePendingBySeries(ctx, *chore.SeriesID); err != nil {
            slog.Error("deleting stale future instances", "error", err)
        } else if err := handler.choreService.SeedFutureOccurrences(ctx, chore, time.Now().AddDate(1, 0, 0)); err != nil {
            slog.Error("re-seeding after recurrence change", "error", err)
        }
    }

    http.Redirect(w, r, "/chores", http.StatusFound)
}
```

**Step 2: Build**

Run: `go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add internal/handlers/chores.go
git commit -m "feat: delete and reseed future occurrences when recurrence changes"
```

---

### Task 9: Handler — modify Delete to clean up future pending instances

**Files:**
- Modify: `internal/handlers/chores.go`

**Step 1: Update the Delete handler**

Replace the current `Delete` handler:

```go
func (handler *ChoreHandler) Delete(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    choreID := chi.URLParam(r, "id")

    chore, err := handler.choreRepo.FindByID(ctx, choreID)
    if err != nil {
        http.NotFound(w, r)
        return
    }

    // Delete future pending siblings before deleting the chore itself
    if chore.SeriesID != nil {
        if err := handler.choreRepo.DeleteFuturePendingBySeries(ctx, *chore.SeriesID); err != nil {
            slog.Error("deleting future pending siblings", "error", err)
        }
    }

    if err := handler.choreRepo.Delete(ctx, choreID); err != nil {
        slog.Error("deleting chore", "error", err)
        http.Error(w, "Error deleting chore", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/chores", http.StatusFound)
}
```

**Step 2: Build**

Run: `go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add internal/handlers/chores.go
git commit -m "feat: delete future pending series instances when deleting a chore"
```

---

### Task 10: Startup — call SeedExistingRecurringChores

**Files:**
- Modify: `internal/server/server.go`

**Step 1: Add startup seeding**

At the end of `New()` in `server.go`, after all handlers/repos are created and before returning `server`, add:

```go
// One-time seed of existing recurring chores that predate series_id tracking
go func() {
    ctx := context.Background()
    if err := choreService.SeedExistingRecurringChores(ctx, time.Now().AddDate(1, 0, 0)); err != nil {
        slog.Error("seeding existing recurring chores", "error", err)
    }
}()
```

Add required imports: `"context"` and `"time"` (may already be present).

**Step 2: Build**

Run: `go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add internal/server/server.go
git commit -m "feat: seed existing recurring chores at startup"
```

---

### Task 11: Calendar — remove runtime expansion

**Files:**
- Modify: `internal/handlers/calendar.go`
- Delete: `internal/services/recurrence_test.go` — `TestExpandChoreOccurrences` tests (if present)
- Optionally remove `ExpandChoreOccurrences` from `internal/services/recurrence.go`

**Step 1: Remove the expansion block from calendar.go**

In `handlers/calendar.go`, delete the entire `recurringChores` fetch and expansion loop (roughly lines 129–150):

```go
// DELETE this entire block:
recurringChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
    Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
    RecurrenceTypes: []models.RecurrenceType{
        models.RecurrenceDaily,
        models.RecurrenceWeekly,
        models.RecurrenceMonthly,
        models.RecurrenceCustom,
        models.RecurrenceCalendar,
    },
})
if err != nil {
    slog.Error("finding recurring chores for calendar", "error", err)
}

for _, chore := range recurringChores {
    expanded, err := services.ExpandChoreOccurrences(chore, start, end)
    if err != nil {
        slog.Error("expanding chore occurrences", "error", err, "chore_id", chore.ID)
        continue
    }
    chores = append(chores, expanded...)
}
```

Also remove the `services` import from `calendar.go` if it's no longer used.

**Step 2: Remove ExpandChoreOccurrences from recurrence.go**

In `internal/services/recurrence.go`, delete the `ExpandChoreOccurrences` function and the `maxExpansionIterations` constant (if only used there — check: `maxExpansionIterations` is also used in `SeedFutureOccurrences`, so keep it).

Actually `maxExpansionIterations` is used in `SeedFutureOccurrences`. Keep it. Just delete `ExpandChoreOccurrences`.

**Step 3: Remove or update tests for ExpandChoreOccurrences**

In `internal/services/recurrence_test.go`, delete any `TestExpandChoreOccurrences` test cases.

**Step 4: Build and test**

Run: `make test`
Expected: all tests pass

**Step 5: Commit**

```bash
git add internal/handlers/calendar.go internal/services/recurrence.go internal/services/recurrence_test.go
git commit -m "feat: remove runtime chore expansion from calendar (use seeded DB rows)"
```

---

### Task 12: Handler — add DeleteHistory + route

**Files:**
- Modify: `internal/handlers/chores.go`
- Modify: `internal/server/server.go`

**Step 1: Add DeleteHistory handler**

Add to `internal/handlers/chores.go`:

```go
func (handler *ChoreHandler) DeleteHistory(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    name := r.FormValue("name")
    if name == "" {
        http.Error(w, "name is required", http.StatusBadRequest)
        return
    }

    if err := handler.choreRepo.DeleteCompletedByName(ctx, name); err != nil {
        slog.Error("deleting chore history by name", "error", err, "name", name)
        http.Error(w, "Error deleting history", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

**Step 2: Add route in server.go**

In the admin-only router group (where `/chores/{id}/delete` already lives), add:

```go
r.Post("/chores/history/delete", choreHandler.DeleteHistory)
```

**Step 3: Build**

Run: `go build ./...`
Expected: success

**Step 4: Commit**

```bash
git add internal/handlers/chores.go internal/server/server.go
git commit -m "feat: add admin endpoint to delete chore history by name"
```

---

### Task 13: Template — history delete button + remove Complete button

**Files:**
- Modify: `templates/pages/chores.templ`

**Step 1: Pass User/isAdmin to ChoreHistoryContent**

Change the `ChoreHistoryContent` signature:

```go
templ ChoreHistoryContent(entries []ChoreHistoryEntry, isAdmin bool) {
```

**Step 2: Update ChoreHistoryContent callers**

In `ChoreList` templ (inside the `if props.ActiveTab == "history"` block):

```go
@ChoreHistoryContent(props.HistoryEntries, props.User.Role == models.RoleAdmin)
```

In `List` handler (`handlers/chores.go`), the HTMX partial response:

```go
component := pages.ChoreHistoryContent(historyEntries, user.Role == models.RoleAdmin)
```

**Step 3: Add Actions column to ChoreHistoryContent**

Update the history content templ. The grid header becomes `md:grid-cols-5` when isAdmin, `md:grid-cols-4` otherwise. Each row likewise.

Replace the `ChoreHistoryContent` templ body:

```go
templ ChoreHistoryContent(entries []ChoreHistoryEntry, isAdmin bool) {
    <div id="chore-table-content">
        if len(entries) == 0 {
            <div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-8 text-center text-stone-500 dark:text-slate-400">
                <p>No completion history</p>
            </div>
        } else {
            <!-- Desktop grid header -->
            <div class={ "hidden md:grid gap-4 px-4 py-2 text-xs font-medium text-stone-500 dark:text-slate-300 uppercase tracking-wider " + historyGridClass(isAdmin) }>
                <div>Chore Name</div>
                <div>Times Completed</div>
                <div>Last Completed</div>
                <div>Last Completed By</div>
                if isAdmin {
                    <div>Actions</div>
                }
            </div>
            <div class="space-y-2 md:space-y-0">
                for _, entry := range entries {
                    <div class={ "history-row bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-4 md:shadow-none md:ring-0 md:border-b md:border-zinc-100 dark:md:border-slate-700 md:rounded-none md:grid md:gap-4 md:items-center md:px-4 md:py-3 " + historyGridClass(isAdmin) }>
                        <div class="text-sm font-medium text-stone-900 dark:text-slate-100">{ entry.Name }</div>
                        <div class="text-sm text-stone-500 dark:text-slate-400 mt-1 md:mt-0">
                            <span class="md:hidden text-xs text-stone-400 dark:text-slate-500">Completed: </span>
                            { strconv.Itoa(entry.CompletionCount) } times
                        </div>
                        <div class="text-sm text-stone-500 dark:text-slate-400 mt-1 md:mt-0">
                            if entry.LastCompletedAt != nil {
                                <span class="md:hidden text-xs text-stone-400 dark:text-slate-500">Last: </span>
                                { entry.LastCompletedAt.Format("Jan 2, 2006") }
                            }
                        </div>
                        <div class="text-sm text-stone-500 dark:text-slate-400 mt-1 md:mt-0">
                            if entry.LastCompletedBy != "" {
                                <span class="md:hidden text-xs text-stone-400 dark:text-slate-500">By: </span>
                                { entry.LastCompletedBy }
                            }
                        </div>
                        if isAdmin {
                            <div class="mt-2 md:mt-0">
                                <form
                                    hx-post="/chores/history/delete"
                                    hx-target="closest .history-row"
                                    hx-swap="outerHTML"
                                    hx-confirm={ "Delete all history for \"" + entry.Name + "\"? This cannot be undone." }
                                >
                                    <input type="hidden" name="name" value={ entry.Name }/>
                                    <button type="submit" class="inline-flex items-center gap-1 text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 transition-colors duration-150 text-sm">
                                        @components.IconTrash("h-4 w-4")
                                        Delete
                                    </button>
                                </form>
                            </div>
                        }
                    </div>
                }
            </div>
        }
    </div>
}
```

Add a helper function at the bottom of the file:

```go
func historyGridClass(isAdmin bool) string {
    if isAdmin {
        return "md:grid-cols-5"
    }
    return "md:grid-cols-4"
}
```

**Step 4: Remove Complete button from ChoreRow**

In `ChoreRow`, delete this block entirely:

```go
if chore.Status != models.ChoreStatusCompleted && chore.AssignedToUserID != nil && *chore.AssignedToUserID == user.ID {
    <button
        hx-post={ fmt.Sprintf("/chores/%s/complete", chore.ID) }
        hx-target={ "#chore-" + chore.ID }
        hx-swap="outerHTML"
        class="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400 hover:text-emerald-800 dark:hover:text-emerald-300 font-medium transition-colors duration-150"
    >
        @components.IconCheck("h-4 w-4")
        Complete
    </button>
}
```

**Step 5: Run templ generate**

```bash
$(go env GOPATH)/bin/templ generate ./...
```

Expected: regenerates `*_templ.go` files with no errors

**Step 6: Build and test**

Run: `go build ./... && make test`
Expected: all pass

**Step 7: Commit**

```bash
git add templates/pages/chores.templ templates/pages/chores_templ.go internal/handlers/chores.go
git commit -m "feat: add admin history delete button; remove complete button from chores tab"
```

---

### Task 14: Admin — remove bulk delete

**Files:**
- Modify: `templates/pages/admin.templ`
- Modify: `internal/handlers/admin.go`
- Modify: `internal/server/server.go`

**Step 1: Remove Maintenance section from admin.templ**

Delete the entire `<!-- Maintenance -->` section and the `DeleteChoreHistory` form (the last `<div>` block before the closing `</div>` of the outer space-y-8).

**Step 2: Remove DeleteChoreHistory from admin.go**

Delete the `DeleteChoreHistory` method entirely from `internal/handlers/admin.go`.

**Step 3: Remove the route from server.go**

Delete this line:

```go
r.Post("/admin/chores/history/delete", adminHandler.DeleteChoreHistory)
```

**Step 4: Run templ generate**

```bash
$(go env GOPATH)/bin/templ generate ./...
```

**Step 5: Build and test**

Run: `go build ./... && make test`
Expected: all pass

**Step 6: Commit**

```bash
git add templates/pages/admin.templ templates/pages/admin_templ.go internal/handlers/admin.go internal/server/server.go
git commit -m "feat: remove broken bulk chore history delete from admin page"
```

---

### Task 15: Final verification

**Step 1: Run full test suite**

Run: `make test`
Expected: all tests pass, no failures

**Step 2: Verify build**

Run: `go build ./...`
Expected: clean

**Step 3: Check for any unused imports**

Run: `go vet ./...`
Expected: no issues

**Step 4: Commit if any lint fixes needed**

```bash
git add -A
git commit -m "fix: address any vet warnings"
```
