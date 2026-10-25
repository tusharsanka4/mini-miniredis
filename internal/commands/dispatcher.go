package commands

import (
	"strings"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func Dispatch(db *store.Store, args []string) string {
	if len(args) == 0 {
		return resp.Error("empty command")
	}
	switch strings.ToUpper(args[0]) {
	case "PING":
		if len(args) == 1 {
			return resp.SimpleString("PONG")
		}
		if len(args) == 2 {
			return resp.BulkString(args[1])
		}
	case "ECHO":
		if len(args) == 2 {
			return resp.BulkString(args[1])
		}
	case "SET":
		if len(args) == 3 {
			return set(db, args[1], args[2])
		}
	case "GET":
		if len(args) == 2 {
			return get(db, args[1])
		}
	case "DEL":
		if len(args) >= 2 {
			return del(db, args[1:])
		}
	default:
		return resp.Error("unknown command")
	}
	return resp.Error("wrong number of arguments")
}
