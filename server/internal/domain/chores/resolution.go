package chores

import (
	"errors"
	"time"
)

var ErrAlreadyResolved = errors.New("chore is already completed")

type Status string

const (
	StatusPending    Status = "pending"
	StatusOverdue    Status = "overdue"
	StatusCompleted  Status = "completed"
	StatusSkipped    Status = "skipped"
	StatusSuperseded Status = "superseded"
)

type Resolution struct {
	status     Status
	userID     string
	resolvedAt time.Time
}

func Restore(status Status) Resolution              { return Resolution{status: status} }
func (resolution Resolution) Status() Status        { return resolution.status }
func (resolution Resolution) UserID() string        { return resolution.userID }
func (resolution Resolution) ResolvedAt() time.Time { return resolution.resolvedAt }
func (resolution Resolution) Complete(userID string, now time.Time) (Resolution, error) {
	return resolution.resolve(StatusCompleted, userID, now)
}
func (resolution Resolution) Skip(userID string, now time.Time) (Resolution, error) {
	return resolution.resolve(StatusSkipped, userID, now)
}
func (resolution Resolution) resolve(status Status, userID string, now time.Time) (Resolution, error) {
	if resolution.status != StatusPending && resolution.status != StatusOverdue {
		return Resolution{}, ErrAlreadyResolved
	}
	return Resolution{status: status, userID: userID, resolvedAt: now}, nil
}
