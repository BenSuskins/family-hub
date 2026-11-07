package chores_test

import (
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/chores"
)

func TestAssignmentRotation(t *testing.T) {
	if got := chores.RotationStart(nil, nil, 0); got != -1 {
		t.Fatalf("empty candidates: got %d", got)
	}
	candidates := []string{"alice", "bob", "carol"}
	bob := "bob"
	missing := "departed"
	for _, test := range []struct {
		name     string
		previous *string
		index    int
		want     int
	}{
		{"first assignment", nil, -1, 0},
		{"continue from member", &bob, -1, 2},
		{"wrap from previous index", nil, 2, 0},
		{"departed member uses position", &missing, 0, 1},
		{"negative prior index", nil, -2, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := chores.RotationStart(candidates, test.previous, test.index); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}
