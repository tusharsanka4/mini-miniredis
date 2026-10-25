package commands

import (
	"strconv"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func listCommand(db *store.Store, command string, args []string) string {
	switch command {
	case "LPUSH", "RPUSH":
		if len(args) >= 3 {
			return integerReply(db.Push(args[1], args[2:], command == "LPUSH"))
		}
	case "LPOP", "RPOP":
		if len(args) == 2 {
			return valueReply(db.Pop(args[1], command == "LPOP"))
		}
	case "LRANGE":
		if len(args) == 4 {
			start, stop, err := parseRange(args[2], args[3])
			if err != nil {
				return resp.Error("value is not an integer or out of range")
			}
			return arrayReply(db.ListRange(args[1], start, stop))
		}
	case "LLEN":
		if len(args) == 2 {
			return integerReply(db.ListLen(args[1]))
		}
	}
	return resp.Error("wrong number of arguments")
}

func parseRange(first, last string) (int, int, error) {
	start, err := strconv.Atoi(first)
	if err != nil {
		return 0, 0, err
	}
	stop, err := strconv.Atoi(last)
	return start, stop, err
}
