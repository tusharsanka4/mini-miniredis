package server

import (
	"strings"

	"github.com/tusharsanka4/mini-miniredis/internal/commands"
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

// Each handleConnection owns one transaction. A disconnect drops its queue.
type transaction struct {
	active bool
	failed bool
	queue  [][]string
}

func (t *transaction) handle(db *store.Store, args []string) string {
	if err := commands.Validate(args); err != nil {
		if t.active {
			t.failed = true
		}
		return resp.Error(err.Error())
	}

	switch strings.ToUpper(args[0]) {
	case "MULTI":
		if t.active {
			return resp.Error("MULTI calls can not be nested")
		}
		t.active = true
		return resp.SimpleString("OK")
	case "EXEC":
		if !t.active {
			return resp.Error("EXEC without MULTI")
		}
		if t.failed {
			*t = transaction{}
			return "-EXECABORT Transaction discarded because of previous errors.\r\n"
		}
		queue := t.queue
		*t = transaction{}
		return commands.ExecuteTransaction(db, queue)
	case "DISCARD":
		if !t.active {
			return resp.Error("DISCARD without MULTI")
		}
		*t = transaction{}
		return resp.SimpleString("OK")
	}

	if t.active {
		t.queue = append(t.queue, args)
		return resp.SimpleString("QUEUED")
	}
	return commands.Dispatch(db, args)
}
