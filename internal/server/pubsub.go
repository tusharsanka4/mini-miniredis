package server

import (
	"strings"

	"github.com/tusharsanka4/mini-miniredis/internal/commands"
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func (c *client) handle(db *store.Store, tx *transaction, args []string) string {
	command := strings.ToUpper(args[0])
	if db.PubSub.Count(c) > 0 {
		switch command {
		case "SUBSCRIBE", "UNSUBSCRIBE":
		case "PING":
			if err := commands.Validate(args); err != nil {
				return resp.Error(err.Error())
			}
			message := ""
			if len(args) == 2 {
				message = args[1]
			}
			return resp.Array([]string{"pong", message})
		default:
			return resp.Error("only SUBSCRIBE, UNSUBSCRIBE and PING are allowed while subscribed")
		}
	}

	if command == "SUBSCRIBE" || command == "UNSUBSCRIBE" {
		if err := commands.Validate(args); err != nil {
			if tx.active {
				tx.failed = true
			}
			return resp.Error(err.Error())
		}
		if tx.active {
			tx.failed = true
			return resp.Error("SUBSCRIBE and UNSUBSCRIBE are not supported inside MULTI")
		}
		db.LockCommands()
		defer db.UnlockCommands()
		if command == "SUBSCRIBE" {
			db.PubSub.Subscribe(c, args[1:])
		} else {
			db.PubSub.Unsubscribe(c, args[1:])
		}
		// The registry already queued one acknowledgement for each channel.
		return ""
	}
	return tx.handle(db, args)
}
