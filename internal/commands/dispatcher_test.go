package commands

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func TestDataStructures(t *testing.T) {
	db := store.New()
	tests := []struct {
		command string
		want    string
	}{
		{"TYPE missing", "+none\r\n"},
		{"EXISTS missing", ":0\r\n"},
		{"LPOP missing", "$-1\r\n"},
		{"RPOP missing", "$-1\r\n"},
		{"LLEN missing", ":0\r\n"},
		{"LRANGE missing 0 -1", "*0\r\n"},
		{"LPUSH tasks a b", ":2\r\n"},
		{"RPUSH tasks c", ":3\r\n"},
		{"TYPE tasks", "+list\r\n"},
		{"EXISTS tasks tasks missing", ":2\r\n"},
		{"LLEN tasks", ":3\r\n"},
		{"LRANGE tasks 0 -1", "*3\r\n$1\r\nb\r\n$1\r\na\r\n$1\r\nc\r\n"},
		{"LRANGE tasks -2 -1", "*2\r\n$1\r\na\r\n$1\r\nc\r\n"},
		{"LRANGE tasks -99 0", "*1\r\n$1\r\nb\r\n"},
		{"LRANGE tasks 0 -99", "*0\r\n"},
		{"LRANGE tasks 99 100", "*0\r\n"},
		{"LRANGE tasks 2 1", "*0\r\n"},
		{"LRANGE tasks 2 999", "*1\r\n$1\r\nc\r\n"},
		{"LPOP tasks", "$1\r\nb\r\n"},
		{"RPOP tasks", "$1\r\nc\r\n"},
		{"LPOP tasks", "$1\r\na\r\n"},
		{"TYPE tasks", "+none\r\n"},
		{"SCARD missing", ":0\r\n"},
		{"SISMEMBER missing a", ":0\r\n"},
		{"SMEMBERS missing", "*0\r\n"},
		{"SREM missing a", ":0\r\n"},
		{"SADD tags b a a", ":2\r\n"},
		{"TYPE tags", "+set\r\n"},
		{"SADD tags a", ":0\r\n"},
		{"SCARD tags", ":2\r\n"},
		{"SISMEMBER tags a", ":1\r\n"},
		{"SISMEMBER tags x", ":0\r\n"},
		{"SMEMBERS tags", "*2\r\n$1\r\na\r\n$1\r\nb\r\n"},
		{"SREM tags a a x", ":1\r\n"},
		{"SREM tags b", ":1\r\n"},
		{"TYPE tags", "+none\r\n"},
		{"HGET missing name", "$-1\r\n"},
		{"HGETALL missing", "*0\r\n"},
		{"HLEN missing", ":0\r\n"},
		{"HDEL missing name", ":0\r\n"},
		{"HSET user name Ada age 30 name Grace", ":2\r\n"},
		{"TYPE user", "+hash\r\n"},
		{"HSET user age 31", ":0\r\n"},
		{"HGET user age", "$2\r\n31\r\n"},
		{"HGET user absent", "$-1\r\n"},
		{"HLEN user", ":2\r\n"},
		{"HGETALL user", "*4\r\n$3\r\nage\r\n$2\r\n31\r\n$4\r\nname\r\n$5\r\nGrace\r\n"},
		{"HDEL user age age missing", ":1\r\n"},
		{"HDEL user name", ":1\r\n"},
		{"TYPE user", "+none\r\n"},
		{"ZCARD missing", ":0\r\n"},
		{"ZSCORE missing a", "$-1\r\n"},
		{"ZRANGE missing 0 -1", "*0\r\n"},
		{"ZREM missing a", ":0\r\n"},
		{"ZADD scores 2 b 1 a 2 c 4 a", ":3\r\n"},
		{"TYPE scores", "+zset\r\n"},
		{"ZADD scores 1 a", ":0\r\n"},
		{"ZCARD scores", ":3\r\n"},
		{"ZSCORE scores a", "$1\r\n1\r\n"},
		{"ZRANGE scores 0 -1", "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"},
		{"ZRANGE scores -2 -1 withscores", "*4\r\n$1\r\nb\r\n$1\r\n2\r\n$1\r\nc\r\n$1\r\n2\r\n"},
		{"ZRANGE scores -99 0", "*1\r\n$1\r\na\r\n"},
		{"ZRANGE scores 2 1", "*0\r\n"},
		{"ZADD scores 3 d NaN e", "-ERR value is not a valid float\r\n"},
		{"ZSCORE scores d", "$-1\r\n"},
		{"ZCARD scores", ":3\r\n"},
		{"ZREM scores a a b c", ":3\r\n"},
		{"TYPE scores", "+none\r\n"},
		{"ZADD new 1 a nope b", "-ERR value is not a valid float\r\n"},
		{"TYPE new", "+none\r\n"},
		{"ZADD extremes +inf high -inf low -0 zero", ":3\r\n"},
		{"ZSCORE extremes high", "$3\r\ninf\r\n"},
		{"ZSCORE extremes low", "$4\r\n-inf\r\n"},
		{"ZSCORE extremes zero", "$1\r\n0\r\n"},
		{"LPUSH list a", ":1\r\n"},
		{"SADD set a", ":1\r\n"},
		{"HSET hash a b", ":1\r\n"},
		{"ZADD zset 1 a", ":1\r\n"},
		{"DEL list set hash zset list", ":4\r\n"},
		{"EXISTS list set hash zset", ":0\r\n"},
	}
	for _, test := range tests {
		if got := Dispatch(db, strings.Fields(test.command)); got != test.want {
			t.Fatalf("%s: got %q, want %q", test.command, got, test.want)
		}
	}
}

func TestWrongTypesAndOverwrite(t *testing.T) {
	creators := []string{"SET key text", "LPUSH key a", "SADD key a", "HSET key a b", "ZADD key 1 a"}
	groups := [][]string{
		{"GET key"},
		{"LPUSH key b", "RPUSH key b", "LPOP key", "RPOP key", "LRANGE key 0 -1", "LLEN key"},
		{"SADD key b", "SREM key b", "SMEMBERS key", "SISMEMBER key b", "SCARD key"},
		{"HSET key b c", "HGET key b", "HDEL key b", "HGETALL key", "HLEN key"},
		{"ZADD key 2 b", "ZRANGE key 0 -1", "ZSCORE key b", "ZREM key b", "ZCARD key"},
	}
	for i, creator := range creators {
		db := store.New()
		Dispatch(db, strings.Fields(creator))
		for j, commands := range groups {
			if i == j {
				continue
			}
			for _, command := range commands {
				want := "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
				if got := Dispatch(db, strings.Fields(command)); got != want {
					t.Errorf("after %s, %s: got %q", creator, command, got)
				}
			}
		}
		if got := Dispatch(db, []string{"SET", "key", "new"}); got != "+OK\r\n" {
			t.Fatal(got)
		}
		if got := Dispatch(db, []string{"GET", "key"}); got != "$3\r\nnew\r\n" {
			t.Fatal(got)
		}
	}
}

func TestInvalidArgumentsDoNotCreateKeys(t *testing.T) {
	commands := []string{
		"LPUSH key", "RPUSH key", "LPOP key 1", "RPOP key extra", "LLEN key extra",
		"LRANGE key x 2", "LRANGE key 0 9999999999999999999999999999", "LRANGE key 0",
		"SADD key", "SREM key", "SMEMBERS key extra", "SISMEMBER key", "SCARD key extra",
		"HSET key a", "HSET key a b c", "HGET key", "HDEL key", "HGETALL key extra", "HLEN key extra",
		"ZADD key 1", "ZADD key NaN a", "ZADD key nope a", "ZADD key 1e9999 a",
		"ZRANGE key 0 -1 REV", "ZRANGE key nope -1", "ZSCORE key", "ZREM key", "ZCARD key extra",
		"TYPE", "EXISTS",
	}
	for _, command := range commands {
		db := store.New()
		if got := Dispatch(db, strings.Fields(command)); !strings.HasPrefix(got, "-ERR ") {
			t.Errorf("%s: expected error, got %q", command, got)
		}
		if db.Type("key") != "none" {
			t.Errorf("%s created a key", command)
		}
	}
}

func TestConcurrentCollections(t *testing.T) {
	db := store.New()
	var clients sync.WaitGroup
	for i := 0; i < 8; i++ {
		clients.Add(1)
		go func() {
			defer clients.Done()
			for j := 0; j < 20; j++ {
				member := fmt.Sprintf("%d:%d", i, j)
				for _, args := range [][]string{
					{"RPUSH", "list", member}, {"SADD", "set", member},
					{"HSET", "hash", member, member}, {"ZADD", "zset", "1", member},
				} {
					if got := Dispatch(db, args); !strings.HasPrefix(got, ":") {
						t.Error(got)
					}
				}
				Dispatch(db, []string{"LRANGE", "list", "0", "-1"})
				Dispatch(db, []string{"ZRANGE", "zset", "0", "-1"})
			}
		}()
	}
	clients.Wait()
	for _, command := range []string{"LLEN list", "SCARD set", "HLEN hash", "ZCARD zset"} {
		if got := Dispatch(db, strings.Fields(command)); got != ":160\r\n" {
			t.Errorf("%s: got %q", command, got)
		}
	}
}
