package models

import "errors"

type ChoreReminders struct {
	Enabled       bool      `json:"enabled"`
	Times         [2]string `json:"times"`
	Timezone      string    `json:"timezone"`
	PushAvailable bool      `json:"pushAvailable"`
}

type NotificationDevice struct {
	APITokenHash *string `json:"-"`
	ID           string  `json:"id"`
	UserID       string  `json:"-"`
	Token        string  `json:"token"`
	Environment  string  `json:"environment"`
}

var ErrDeviceOwnership = errors.New("device belongs to another user")

type BroadcastResult struct {
	Accepted int `json:"accepted"`
	Failed   int `json:"failed"`
}

var ErrBroadcastConflict = errors.New("broadcast identifier already used")
