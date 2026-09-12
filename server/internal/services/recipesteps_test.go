package services_test

import (
	"testing"

	"github.com/bensuskins/family-hub/internal/services"
)

func TestSplitIntoSteps(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "  \n\t\n  ", nil},
		{"single line", "Boil the pasta", []string{"Boil the pasta"}},
		{
			"trims and drops blanks",
			"  Boil the pasta  \n\n\t Drain it \n",
			[]string{"Boil the pasta", "Drain it"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := services.SplitIntoSteps(testCase.text)
			if len(got) != len(testCase.want) {
				t.Fatalf("got %d steps %q, want %d %q", len(got), got, len(testCase.want), testCase.want)
			}
			for i := range got {
				if got[i] != testCase.want[i] {
					t.Errorf("step %d = %q, want %q", i, got[i], testCase.want[i])
				}
			}
		})
	}
}

func TestParseStepDuration(t *testing.T) {
	tests := []struct {
		name string
		step string
		want *int // nil = no timer
	}{
		{"no duration", "Season generously with salt", nil},
		{"temperature is not a duration", "Preheat the oven to 180C", nil},
		{"bare number", "Add 3 eggs", nil},

		{"minutes", "Simmer for 20 minutes", ptr(1200)},
		{"abbreviated mins", "Rest for 5 mins", ptr(300)},
		{"singular minute", "Blanch for 1 minute", ptr(60)},
		{"hours", "Braise for 2 hours", ptr(7200)},
		{"abbreviated hr", "Chill for 1 hr", ptr(3600)},
		{"bare h", "Prove for 2h", ptr(7200)},
		{"seconds", "Blitz for 30 seconds", ptr(30)},
		{"abbreviated secs", "Whisk for 45 secs", ptr(45)},
		{"case insensitive", "Bake for 25 MINUTES", ptr(1500)},
		{"fractional", "Steep for 1.5 hours", ptr(5400)},

		{"range takes lower bound", "Bake for 10-12 minutes", ptr(600)},
		{"range with en dash", "Bake for 10–12 minutes", ptr(600)},
		{"range with to", "Fry for 3 to 4 minutes", ptr(180)},

		{"compound is summed", "Roast for 1 hr 30 mins", ptr(5400)},
		{"compound with and", "Prove for 1 hour and 15 minutes", ptr(4500)},

		{"first duration wins", "Bake 20 minutes, then rest 10 minutes", ptr(1200)},
		{"zero is not a timer", "Rest for 0 minutes", nil},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := services.ParseStepDuration(testCase.step)
			switch {
			case testCase.want == nil && got != nil:
				t.Fatalf("got %d seconds, want no timer", *got)
			case testCase.want != nil && got == nil:
				t.Fatalf("got no timer, want %d seconds", *testCase.want)
			case testCase.want != nil && *got != *testCase.want:
				t.Errorf("got %d seconds, want %d", *got, *testCase.want)
			}
		})
	}
}

func TestStepDurations(t *testing.T) {
	t.Run("nil for no steps", func(t *testing.T) {
		if got := services.StepDurations(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("aligns positionally with steps", func(t *testing.T) {
		steps := []string{"Chop the onions", "Simmer for 20 minutes", "Serve"}
		got := services.StepDurations(steps)

		if len(got) != len(steps) {
			t.Fatalf("got %d durations, want %d", len(got), len(steps))
		}
		if got[0] != nil {
			t.Errorf("step 0 got %d seconds, want no timer", *got[0])
		}
		if got[1] == nil || *got[1] != 1200 {
			t.Errorf("step 1 got %v, want 1200 seconds", got[1])
		}
		if got[2] != nil {
			t.Errorf("step 2 got %d seconds, want no timer", *got[2])
		}
	})
}

func ptr(seconds int) *int { return &seconds }
