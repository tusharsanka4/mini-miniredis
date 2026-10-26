# mini-miniredis

Run with `go run ./cmd/miniredis`. The server listens on `127.0.0.1:6379`.

Code layout:

- `cmd/miniredis/main.go`: starts the server.
- `internal/server`: TCP listener and client connection handler.
- `internal/resp`: command parser and reply formatting.
- `internal/store`: shared map and its mutex. `expiry.go` is a placeholder for later.
- `internal/commands`: command dispatcher and string command handlers.
- `internal/clock`: real time and a manually controlled clock for tests.

Each client gets a `handleConnection` goroutine. It reads RESP commands through
a buffered reader backed by `conn.Read`, dispatches them, and uses `conn.Write`
for replies. All clients share one mutex-protected map of typed values.

Command names are case insensitive. These are the supported forms:

| Type | Commands | Stored as |
| --- | --- | --- |
| General | `PING [message]`, `ECHO message`, `DEL key [key ...]`, `EXISTS key [key ...]`, `TYPE key` | One shared keyspace |
| Strings | `SET key value`, `GET key` | Go string |
| Lists | `LPUSH key value [value ...]`, `RPUSH key value [value ...]`, `LPOP key`, `RPOP key`, `LRANGE key start stop`, `LLEN key` | `[]string` |
| Sets | `SADD key member [member ...]`, `SREM key member [member ...]`, `SMEMBERS key`, `SISMEMBER key member`, `SCARD key` | `map[string]bool` |
| Hashes | `HSET key field value [field value ...]`, `HGET key field`, `HDEL key field [field ...]`, `HGETALL key`, `HLEN key` | `map[string]string` |
| Sorted sets | `ZADD key score member [score member ...]`, `ZRANGE key start stop [WITHSCORES]`, `ZSCORE key member`, `ZREM key member [member ...]`, `ZCARD key` | `map[string]float64`, sorted on reads |

Each key has one type. Using the wrong type returns `WRONGTYPE`. `SET` replaces
any previous value. Removing a collection's last item removes its key too.
Missing values return null, zero, or an empty array as appropriate.

Ranges have inclusive endpoints; negative indexes count from the end (`-1` is
the last item). Sorted sets order by score, then member name for ties. Sets and
hashes return members/fields alphabetically for predictable output (Redis does
not guarantee an order for those commands).

These are deliberately simple implementations, not Redis's optimized internals.
For example, sorted-set ranges sort the whole set, and pushing onto the front
of a list copies the list. Advanced command options, blocking commands, streams,
and other Redis types are not implemented yet.

After changing code, stop the server with Ctrl+C and run `go run ./cmd/miniredis`
again. In a separate terminal, connect with `redis-cli -p 6379` and try:

```text
RPUSH tasks first second
LRANGE tasks 0 -1
LPOP tasks

SADD tags go redis go
SMEMBERS tags

HSET user:1 name tushar language Go
HGETALL user:1

ZADD leaderboard 10 alice 20 bob
ZRANGE leaderboard 0 -1 WITHSCORES

TYPE tasks
GET tasks
```

The last command intentionally returns `WRONGTYPE` because `tasks` is a list.

Requests must be RESP arrays of bulk strings (the format used by Redis clients).
For example, `GET key` is `*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n` on the wire.
Commands may arrive in pieces or be pipelined on one connection. Arguments can
contain arbitrary bytes, including CRLF. Requests allow up to 1024 arguments and
1 MiB per argument. Invalid RESP gets an error and closes the connection;
unknown commands and wrong argument counts get an error and keep it open.

Values live in memory until deleted or the process stops. There is no expiration
or persistence yet.

The store uses real time by default. Tests can supply a manual clock:

```go
c := clock.NewManual(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
db := store.NewWithClock(c)
db.Set("name", "tushar")

c.Advance(5 * time.Second) // Moves instantly, without sleeping.
now := c.Now()
c.Set(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
```

Import `time`, `github.com/tusharsanka4/mini-miniredis/internal/clock`, and
`github.com/tusharsanka4/mini-miniredis/internal/store` in the test using it.
Manual time stays fixed until changed, and it can move forward or backward.
The store keeps the supplied clock for future expiration handling; advancing
it does not expire values yet.

Run tests with `go test ./...` (or `go test -race ./...` with a C compiler installed).
