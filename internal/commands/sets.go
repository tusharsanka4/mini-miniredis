package commands

import (
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func setCommand(db *store.Store, command string, args []string) string {
	switch command {
	case "SADD":
		if len(args) >= 3 {
			return integerReply(db.SetAdd(args[1], args[2:]))
		}
	case "SREM":
		if len(args) >= 3 {
			return integerReply(db.SetRemove(args[1], args[2:]))
		}
	case "SMEMBERS":
		if len(args) == 2 {
			return arrayReply(db.SetMembers(args[1]))
		}
	case "SISMEMBER":
		if len(args) == 3 {
			return integerReply(db.SetContains(args[1], args[2]))
		}
	case "SCARD":
		if len(args) == 2 {
			return integerReply(db.SetLen(args[1]))
		}
	}
	return resp.Error("wrong number of arguments")
}
