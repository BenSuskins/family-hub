package server

import (
	"context"
	"database/sql"
	notifications "github.com/bensuskins/family-hub/internal/application/notifications"
	"github.com/bensuskins/family-hub/internal/config"
)

func New(database *sql.DB, cfg config.Config, senders ...notifications.MessageSender) *Server {
	var sender notifications.MessageSender
	if len(senders) > 0 {
		sender = senders[0]
	}
	srv, _ := Compose(context.Background(), database, cfg, Dependencies{MessageSender: sender})
	return srv
}
