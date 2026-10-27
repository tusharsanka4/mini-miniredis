package commands

import (
	"errors"
	"strings"
)

// Validate checks the command name and argument count without touching data.
// Value-dependent checks (types, numeric values, etc.) happen during execution.
func Validate(args []string) error {
	if len(args) == 0 {
		return errors.New("empty command")
	}
	valid := false
	switch strings.ToUpper(args[0]) {
	case "MULTI", "EXEC", "DISCARD":
		valid = len(args) == 1
	case "PING":
		valid = len(args) == 1 || len(args) == 2
	case "ECHO", "GET", "TYPE", "LPOP", "RPOP", "LLEN", "SMEMBERS", "SCARD", "HGETALL", "HLEN", "ZCARD":
		valid = len(args) == 2
	case "SET", "SISMEMBER", "HGET", "ZSCORE":
		valid = len(args) == 3
	case "DEL", "EXISTS":
		valid = len(args) >= 2
	case "LPUSH", "RPUSH", "SADD", "SREM", "HDEL", "ZREM":
		valid = len(args) >= 3
	case "HSET", "ZADD":
		valid = len(args) >= 4 && len(args)%2 == 0
	case "LRANGE":
		valid = len(args) == 4
	case "ZRANGE":
		valid = len(args) == 4 || len(args) == 5
	default:
		return errors.New("unknown command")
	}
	if !valid {
		return errors.New("wrong number of arguments")
	}
	return nil
}
