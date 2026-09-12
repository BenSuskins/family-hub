package services

import (
	"regexp"
	"strconv"
	"strings"
)

// SplitIntoSteps splits a legacy free-text instructions blob into one step per
// non-empty line. Recipes created before the `steps` column existed still carry
// their method in Recipe.Instructions, and both the web cook mode and the JSON
// API fall back to this so those recipes remain cookable.
func SplitIntoSteps(text string) []string {
	var steps []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			steps = append(steps, line)
		}
	}
	return steps
}

// durationPattern matches a quantity and a time unit, optionally as a range
// ("10-12 minutes"). Bare "m" and "s" are deliberately excluded: they collide
// with ordinary recipe prose far too often to be worth the extra coverage.
var durationPattern = regexp.MustCompile(
	`(?i)\b(\d+(?:\.\d+)?)\s*(?:-|–|to)?\s*(?:\d+(?:\.\d+)?)?\s*(hours?|hrs?|h|minutes?|mins?|seconds?|secs?)\b`,
)

// unitSeconds maps a matched unit to its length in seconds.
var unitSeconds = map[string]int{
	"h": 3600, "hr": 3600, "hrs": 3600, "hour": 3600, "hours": 3600,
	"min": 60, "mins": 60, "minute": 60, "minutes": 60,
	"sec": 1, "secs": 1, "second": 1, "seconds": 1,
}

// joinPattern matches the only text allowed to sit between the two halves of a
// compound duration such as "1 hr 30 mins".
var joinPattern = regexp.MustCompile(`^[\s,]*(and\s+)?$`)

// ParseStepDuration finds a timer-worthy duration in a recipe step and returns
// it in seconds, or nil when the step has no recognisable timing.
//
// It takes the *first* duration in the step, so "bake 20 minutes, then rest 10
// minutes" yields 20 minutes rather than 30 — the step's own timer should fire
// at the first thing the cook is waiting for. A range ("10-12 minutes") yields
// its lower bound for the same reason: better to check early than to burn it.
// A compound duration written as adjacent units ("1 hr 30 mins") is summed.
func ParseStepDuration(step string) *int {
	matches := durationPattern.FindAllStringSubmatchIndex(step, -1)
	if len(matches) == 0 {
		return nil
	}

	seconds, ok := matchSeconds(step, matches[0])
	if !ok {
		return nil
	}

	// Sum an immediately adjacent smaller unit, so "1 hr 30 mins" is one timer.
	if len(matches) > 1 {
		gap := step[matches[0][1]:matches[1][0]]
		if joinPattern.MatchString(gap) {
			if next, ok := matchSeconds(step, matches[1]); ok && next < seconds {
				seconds += next
			}
		}
	}

	if seconds <= 0 {
		return nil
	}
	return &seconds
}

// matchSeconds converts one durationPattern match into seconds.
func matchSeconds(step string, match []int) (int, bool) {
	quantity, err := strconv.ParseFloat(step[match[2]:match[3]], 64)
	if err != nil {
		return 0, false
	}
	unit, ok := unitSeconds[strings.ToLower(step[match[4]:match[5]])]
	if !ok {
		return 0, false
	}
	return int(quantity * float64(unit)), true
}

// StepDurations parses every step in order, returning a slice positionally
// aligned with steps where a nil entry means "no timer for this step". It
// returns nil for an empty step list so the field is omitted rather than
// serialising as an empty array.
func StepDurations(steps []string) []*int {
	if len(steps) == 0 {
		return nil
	}
	durations := make([]*int, len(steps))
	for i, step := range steps {
		durations[i] = ParseStepDuration(step)
	}
	return durations
}
