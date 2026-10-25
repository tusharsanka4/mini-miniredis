package commands

import (
	"math"
	"strconv"
	"strings"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func sortedSetCommand(db *store.Store, command string, args []string) string {
	switch command {
	case "ZADD":
		if len(args) >= 4 && len(args)%2 == 0 {
			// Parse every score first: a bad score must not partially update the key.
			members := make([]store.ScoredMember, 0, (len(args)-2)/2)
			for i := 2; i < len(args); i += 2 {
				score, err := strconv.ParseFloat(args[i], 64)
				if err != nil || math.IsNaN(score) {
					return resp.Error("value is not a valid float")
				}
				members = append(members, store.ScoredMember{Member: args[i+1], Score: score})
			}
			return integerReply(db.SortedSetAdd(args[1], members))
		}
	case "ZRANGE":
		if len(args) == 4 || len(args) == 5 {
			withScores := len(args) == 5
			if withScores && !strings.EqualFold(args[4], "WITHSCORES") {
				return resp.Error("syntax error")
			}
			start, stop, err := parseRange(args[2], args[3])
			if err != nil {
				return resp.Error("value is not an integer or out of range")
			}
			members, err := db.SortedSetRange(args[1], start, stop)
			if err != nil {
				return resp.WrongType()
			}
			var values []string
			for _, member := range members {
				values = append(values, member.Member)
				if withScores {
					values = append(values, formatScore(member.Score))
				}
			}
			return resp.Array(values)
		}
	case "ZSCORE":
		if len(args) == 3 {
			score, exists, err := db.SortedSetScore(args[1], args[2])
			return valueReply(formatScore(score), exists, err)
		}
	case "ZREM":
		if len(args) >= 3 {
			return integerReply(db.SortedSetRemove(args[1], args[2:]))
		}
	case "ZCARD":
		if len(args) == 2 {
			return integerReply(db.SortedSetLen(args[1]))
		}
	}
	return resp.Error("wrong number of arguments")
}

func formatScore(score float64) string {
	if math.IsInf(score, 1) {
		return "inf"
	}
	if math.IsInf(score, -1) {
		return "-inf"
	}
	if score == 0 {
		return "0"
	}
	return strconv.FormatFloat(score, 'f', -1, 64)
}
