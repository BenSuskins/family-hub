package chores

import (
	"errors"
	"time"
)

var ErrAlreadyResolved = errors.New("chore is already completed")

type Resolution struct {
	status     string
	userID     string
	resolvedAt time.Time
}

func Restore(status string) Resolution              { return Resolution{status: status} }
func (resolution Resolution) Status() string        { return resolution.status }
func (resolution Resolution) UserID() string        { return resolution.userID }
func (resolution Resolution) ResolvedAt() time.Time { return resolution.resolvedAt }
func (resolution Resolution) Complete(userID string, now time.Time) (Resolution, error) {
	return resolution.resolve("completed", userID, now)
}
func (resolution Resolution) Skip(userID string, now time.Time) (Resolution, error) {
	return resolution.resolve("skipped", userID, now)
}
func (resolution Resolution) resolve(status, userID string, now time.Time) (Resolution, error) {
	if resolution.status != "pending" && resolution.status != "overdue" {
		return Resolution{}, ErrAlreadyResolved
	}
	return Resolution{status: status, userID: userID, resolvedAt: now}, nil
}
