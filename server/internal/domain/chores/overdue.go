package chores

import (
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

func IsOverdue(chore models.Chore, now time.Time) bool {
	if chore.DueDate == nil {
		return false
	}

	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(chore.DueDate.Year(), chore.DueDate.Month(), chore.DueDate.Day(), 0, 0, 0, 0, now.Location())

	if dueDay.Before(startOfToday) {
		return true
	}

	if dueDay.Equal(startOfToday) && chore.DueTime != nil {
		parsed, err := time.Parse("15:04", *chore.DueTime)
		if err != nil {
			return false
		}
		dueAt := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
		return now.After(dueAt)
	}

	return false
}
