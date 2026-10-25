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
	command := strings.ToUpper(args[0])
	switch command {
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
	case "TYPE":
		if len(args) == 2 {
			return resp.SimpleString(db.Type(args[1]))
		}
	case "EXISTS":
		if len(args) >= 2 {
			return resp.Integer(db.Exists(args[1:]...))
		}
	case "LPUSH", "RPUSH", "LPOP", "RPOP", "LRANGE", "LLEN":
		return listCommand(db, command, args)
	case "SADD", "SREM", "SMEMBERS", "SISMEMBER", "SCARD":
		return setCommand(db, command, args)
	case "HSET", "HGET", "HDEL", "HGETALL", "HLEN":
		return hashCommand(db, command, args)
	case "ZADD", "ZRANGE", "ZSCORE", "ZREM", "ZCARD":
		return sortedSetCommand(db, command, args)
	default:
		return resp.Error("unknown command")
	}
	return resp.Error("wrong number of arguments")
}

func integerReply(value int, err error) string {
	if err != nil {
		return resp.WrongType()
	}
	return resp.Integer(value)
}

func valueReply(value string, exists bool, err error) string {
	if err != nil {
		return resp.WrongType()
	}
	if !exists {
		return resp.NullBulkString()
	}
	return resp.BulkString(value)
}

func arrayReply(values []string, err error) string {
	if err != nil {
		return resp.WrongType()
	}
	return resp.Array(values)
}
