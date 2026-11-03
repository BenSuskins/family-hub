package chores_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/domain/chores"
)

func TestResolutionTransitions(t *testing.T) {
	instant := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	for _, status := range []chores.Status{chores.StatusPending, chores.StatusOverdue} {
		original := chores.Restore(status)
		completed, err := original.Complete("member", instant)
		if err != nil {
			t.Fatal(err)
		}
		if original.Status() != status || completed.Status() != chores.StatusCompleted || completed.UserID() != "member" || completed.ResolvedAt() != instant {
			t.Fatalf("invalid completed transition")
		}
		skipped, err := original.Skip("member", instant)
		if err != nil {
			t.Fatal(err)
		}
		if skipped.Status() != chores.StatusSkipped || original.Status() != status {
			t.Fatalf("invalid skipped transition")
		}
		if _, err := completed.Skip("member", instant); !errors.Is(err, chores.ErrAlreadyResolved) {
			t.Fatalf("resolved chore accepted: %v", err)
		}
	}
	for _, status := range []chores.Status{chores.StatusCompleted, chores.StatusSkipped, chores.StatusSuperseded, "legacy"} {
		if _, err := chores.Restore(status).Complete("member", instant); !errors.Is(err, chores.ErrAlreadyResolved) {
			t.Fatalf("status %q accepted: %v", status, err)
		}
	}
}
