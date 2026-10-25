package commands

import (
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func hashCommand(db *store.Store, command string, args []string) string {
	switch command {
	case "HSET":
		if len(args) >= 4 && len(args)%2 == 0 {
			return integerReply(db.HashSet(args[1], args[2:]))
		}
	case "HGET":
		if len(args) == 3 {
			return valueReply(db.HashGet(args[1], args[2]))
		}
	case "HDEL":
		if len(args) >= 3 {
			return integerReply(db.HashDelete(args[1], args[2:]))
		}
	case "HGETALL":
		if len(args) == 2 {
			return arrayReply(db.HashGetAll(args[1]))
		}
	case "HLEN":
		if len(args) == 2 {
			return integerReply(db.HashLen(args[1]))
		}
	}
	return resp.Error("wrong number of arguments")
}
