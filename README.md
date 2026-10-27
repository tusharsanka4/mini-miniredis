# mini-miniredis

A small Redis-like server written in Go, built to understand how the pieces work.
It accepts Redis TCP commands and HTTP JSON requests, and stores values in memory.
The code favors straightforward functions, slices, and maps.

It supports strings, lists, sets, hashes, sorted sets, and a controllable clock
for tests. This is a learning project with a subset of Redis commands.

## Run the server

Install Go matching the version in `go.mod` (currently `1.27.2`). The project
uses only the Go standard library.

From the project root:

```sh
go run ./cmd/miniredis
```

You should see:

```text
MiniRedis listening on 127.0.0.1:6379
MiniRedis HTTP API listening on 127.0.0.1:8080
```

Keep this terminal open. Press **Ctrl+C** to stop the server. After editing Go
code, stop and start it again. Stopping the process loses all stored data.

The two startup messages may appear in either order. Both listeners must start
successfully. If a port is occupied, stop the other process or choose different
addresses:

```sh
go run ./cmd/miniredis -redis-addr 127.0.0.1:6380 -http-addr 127.0.0.1:8081
```

Both interfaces share the same store. Changing a value over HTTP makes it
immediately available to TCP clients, and vice versa.

## HTTP JSON API

The HTTP API runs at `http://127.0.0.1:8080` by default. The contract is in
[openapi.json](openapi.json), which can be imported into Postman. It uses the
same commands and argument rules listed below.

`MULTI`, `EXEC`, and `DISCARD` require a persistent TCP connection and are not
available through this stateless HTTP endpoint. HTTP commands still use the
shared command mutex, so they cannot interrupt a TCP transaction's execution.

### Check the server

```powershell
Invoke-RestMethod http://127.0.0.1:8080/health
```

`GET /health` returns `200` with `{"status":"ok"}` when the HTTP handler is running.

### Send a command

Use `POST /command` with `Content-Type: application/json`. The body contains one
`command` array. Every element must be a string, including numbers such as list
indexes and sorted-set scores.

```powershell
$base = 'http://127.0.0.1:8080'

Invoke-RestMethod -Uri "$base/command" -Method Post -ContentType 'application/json' -Body '{"command":["SET","name","tushar"]}'

Invoke-RestMethod -Uri "$base/command" -Method Post -ContentType 'application/json' -Body '{"command":["GET","name"]}'

Invoke-RestMethod -Uri "$base/command" -Method Post -ContentType 'application/json' -Body '{"command":["RPUSH","tasks","first","second"]}'

Invoke-RestMethod -Uri "$base/command" -Method Post -ContentType 'application/json' -Body '{"command":["LRANGE","tasks","0","-1"]}'
```

The JSON replies are:

```json
{"result":"OK"}
```

```json
{"result":"tushar"}
```

```json
{"result":2}
```

```json
{"result":["first","second"]}
```

PowerShell displays these JSON objects as properties. For a reusable helper:

```powershell
function Send-MiniRedis {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Command)
    $body = @{ command = $Command } | ConvertTo-Json -Compress
    Invoke-RestMethod -Uri 'http://127.0.0.1:8080/command' -Method Post -ContentType 'application/json; charset=utf-8' -Body ([System.Text.Encoding]::UTF8.GetBytes($body))
}

Send-MiniRedis HSET user:1 name tushar
Send-MiniRedis HGETALL user:1
Send-MiniRedis ZADD scores 10 alice 20 bob
Send-MiniRedis ZRANGE scores 0 -1 WITHSCORES
```

Strings become JSON strings, integer replies become numbers, and collection
replies become arrays. A missing value returns `{"result":null}` with status
`200`; an empty collection returns `{"result":[]}`. `HGETALL` and `ZRANGE ...
WITHSCORES` return flat arrays of alternating fields/values or members/scores.
Scores stay strings, including `inf` and `-inf`.

### HTTP errors and limits

Errors have a JSON body such as `{"error":"ERR unknown command"}`.

| Status | Meaning |
| --- | --- |
| `200` | Successful command or health check |
| `400` | Invalid JSON or command, wrong argument count, or `WRONGTYPE` |
| `404` | Unknown endpoint |
| `405` | Wrong HTTP method; the `Allow` header gives the supported method |
| `413` | Request body exceeds 2 MiB |
| `415` | Content-Type is not `application/json` |
| `500` | Unexpected internal reply format |

Requests allow 1 to 1024 command elements, each at most 1 MiB in UTF-8 bytes.
The body must contain one JSON object with no unknown fields or trailing values.
Invalid request bodies are rejected before executing a command. Writes are not
automatically retried; repeating a write such as `LPUSH` repeats its effect.

The JSON API is for UTF-8 text. JSON output replaces invalid UTF-8 bytes stored
through TCP with replacement characters; use the TCP interface for arbitrary
binary data. Both listeners bind to localhost by default and have no authentication.

## Connect and send commands

Open a second terminal with `redis-cli` installed:

```sh
redis-cli -h 127.0.0.1 -p 6379
```

On Windows, if the executable is not on your PATH, open PowerShell in the folder
containing `redis-cli.exe` and run:

```powershell
.\redis-cli.exe -h 127.0.0.1 -p 6379
```

Your Go program is the server; use only the CLI from a Redis download while
working with this project.

At the CLI prompt:

```text
127.0.0.1:6379> PING
PONG
127.0.0.1:6379> SET name tushar
OK
127.0.0.1:6379> GET name
"tushar"
127.0.0.1:6379> DEL name
(integer) 1
127.0.0.1:6379> GET name
(nil)
```

Try the collection types:

```text
RPUSH tasks first second
LPUSH tasks urgent
LRANGE tasks 0 -1
LPOP tasks

SADD tags go redis go
SMEMBERS tags
SISMEMBER tags go

HSET user:1 name tushar language Go
HGET user:1 name
HGETALL user:1

ZADD leaderboard 10 alice 20 bob 15 charlie
ZRANGE leaderboard 0 -1 WITHSCORES
ZSCORE leaderboard bob

TYPE tasks
EXISTS tasks user:1
```

`LRANGE tasks 0 -1` returns the whole list. The sorted-set range returns members
from lowest to highest score, with each score following its member.

## Supported commands

Command names are case insensitive. Square brackets mark optional arguments;
do not type the brackets.

### General and strings

| Command | Result |
| --- | --- |
| `PING [message]` | `PONG`, or the supplied message |
| `ECHO message` | The supplied message |
| `SET key value` | Stores a string and returns `OK` |
| `GET key` | The string, or null if absent |
| `DEL key [key ...]` | Number of keys deleted |
| `EXISTS key [key ...]` | Number of supplied keys that exist; duplicates count |
| `TYPE key` | `string`, `list`, `set`, `hash`, `zset`, or `none` |

### Lists

| Command | Result |
| --- | --- |
| `LPUSH key value [value ...]` | Prepends values and returns the new length |
| `RPUSH key value [value ...]` | Appends values and returns the new length |
| `LPOP key` | Removes the first value, or returns null if absent |
| `RPOP key` | Removes the last value, or returns null if absent |
| `LRANGE key start stop` | Values in the requested range |
| `LLEN key` | List length, or zero if absent |

`LPUSH key a b` produces `b, a`: each supplied value goes to the front.

### Sets

| Command | Result |
| --- | --- |
| `SADD key member [member ...]` | Number of new members added |
| `SREM key member [member ...]` | Number of members removed |
| `SMEMBERS key` | All members |
| `SISMEMBER key member` | `1` if present, otherwise `0` |
| `SCARD key` | Number of members, or zero if absent |

### Hashes

| Command | Result |
| --- | --- |
| `HSET key field value [field value ...]` | Number of new fields added |
| `HGET key field` | Field value, or null if absent |
| `HDEL key field [field ...]` | Number of fields removed |
| `HGETALL key` | Alternating field names and values |
| `HLEN key` | Number of fields, or zero if absent |

Updating an existing field changes its value without counting it as a new field.

### Sorted sets

| Command | Result |
| --- | --- |
| `ZADD key score member [score member ...]` | Number of new members added |
| `ZRANGE key start stop [WITHSCORES]` | Members in rank order, optionally with scores |
| `ZSCORE key member` | Member's score, or null if absent |
| `ZREM key member [member ...]` | Number of members removed |
| `ZCARD key` | Number of members, or zero if absent |

`ZADD` updates existing members' scores without counting them as new. Invalid
scores reject the whole command without partially changing the set.

### Shared behavior

- Each key holds one type. Using a command on the wrong type returns `WRONGTYPE`.
- `SET` replaces any previous value, including a collection.
- Removing a collection's last item also removes its key.
- Ranges have inclusive endpoints. Negative indexes count from the end;
  `-1` means the last item.
- Equal sorted-set scores are ordered by member bytes.
- Set members and hash fields are returned in sorted order for predictable
  output. Callers should not rely on this ordering with a real Redis server.
- Missing collections return empty arrays, zero counts, or null values,
  depending on the command.

For example, `GET tasks` after creating a list named `tasks` returns:

```text
(error) WRONGTYPE Operation against a key holding the wrong kind of value
```

## Transactions (TCP)

Each client connection has its own transaction queue. Enter `MULTI` to start
queueing commands and `EXEC` to execute them in order:

```text
127.0.0.1:6379> MULTI
OK
127.0.0.1:6379> SET color blue
QUEUED
127.0.0.1:6379> GET color
QUEUED
127.0.0.1:6379> EXEC
1) OK
2) "blue"
```

| Command | Result |
| --- | --- |
| `MULTI` | Starts this connection's transaction and returns `OK` |
| `EXEC` | Runs queued commands and returns an array of their replies |
| `DISCARD` | Cancels the queue without executing it and returns `OK` |

- Queued commands do not read or change the store until `EXEC`.
- Other clients can use the server while a transaction is being queued.
- During `EXEC`, no other TCP or HTTP command can execute between its commands.
- Replies preserve each command's type, including nested arrays and errors.
- An empty transaction returns an empty array.
- An unknown command or bad argument count during queueing marks the transaction
  as failed. `EXEC` then returns `EXECABORT` without running any queued commands.
- Execution-time errors, such as `WRONGTYPE` or an invalid score, become error
  elements in the result array. Remaining commands still run; there is no rollback.
- Nested `MULTI` calls are rejected. `EXEC` and `DISCARD` outside a transaction
  return errors.
- `EXEC`, `DISCARD`, or disconnecting clears the connection's transaction state.

`WATCH` and optimistic locking are not implemented. See the
[Redis transaction documentation](https://redis.io/docs/latest/develop/using-commands/transactions/)
for the underlying queueing and error behavior.

## How it works

1. `net.Listen` opens the TCP listener.
2. `listener.Accept` accepts a client and starts a `handleConnection` goroutine.
3. A buffered reader backed by `conn.Read` passes bytes to the RESP parser.
4. The dispatcher checks the command and arguments, then calls the store.
5. Reply helpers format RESP responses, sent through `conn.Write`.

All clients share one store. Its command mutex serializes normal commands and
holds off other clients for the whole execution of `EXEC`. A separate map mutex
protects each store operation. `EXEC` calls the dispatcher without reacquiring
the command mutex, so the transaction cannot deadlock on its own lock. Both
locks are released before writing a reply to the network.

| Value type | Go representation |
| --- | --- |
| String | `string` |
| List | `[]string` |
| Set | `map[string]bool` |
| Hash | `map[string]string` |
| Sorted set | `map[string]float64` |

Sorted-set range queries sort the members when read. Pushing onto the front of
a list copies the list. These choices keep the code easy to follow rather than
matching Redis's internal data structures or performance.

### RESP support

Requests must be RESP arrays of bulk strings. For example, `GET key` is sent as:

```text
*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n
```

Here `\r\n` represents actual carriage-return and newline bytes. Requests can
arrive in fragments or be pipelined over one connection. Bulk strings can contain
arbitrary bytes, including newlines.

The parser accepts up to 1024 arguments and 1 MiB per argument. Malformed or
truncated RESP produces an error and closes the connection. Unknown commands
and invalid command arguments produce errors while leaving the connection open.

## Controllable clock

`store.New()` supplies a real clock whose `Now()` method calls `time.Now()`.
Tests can supply a manual clock that stays fixed until explicitly changed:

```go
package store_test

import (
    "testing"
    "time"

    "github.com/tusharsanka4/mini-miniredis/internal/clock"
    "github.com/tusharsanka4/mini-miniredis/internal/store"
)

func TestManualTime(t *testing.T) {
    start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
    c := clock.NewManual(start)
    db := store.NewWithClock(c)
    db.Set("name", "tushar")

    c.Advance(10 * time.Second) // Moves instantly, without sleeping.
    if !c.Now().Equal(start.Add(10 * time.Second)) {
        t.Fatal("unexpected clock time")
    }

    c.Set(start) // Set an exact time, including moving backward.
}
```

The manual clock is safe to read and change from multiple goroutines. It is a
Go testing helper; there are no Redis commands for changing it. The store keeps
the clock for future expiration handling. Advancing time does not expire keys yet.

## Project layout

```text
mini-miniredis/
├── go.mod
├── cmd/miniredis/main.go
├── internal/
│   ├── clock/       # Real and manual clocks, with tests
│   ├── api/         # HTTP handlers and JSON replies, with tests
│   ├── server/      # Listener, connection handler, transactions, and tests
│   ├── resp/        # RESP parser and reply formatting
│   ├── store/       # Typed values, mutex, and data-type operations
│   └── commands/    # Dispatcher, command handlers, and command tests
└── README.md
```

`openapi.json` at the project root describes the HTTP endpoints and examples.

Collection operations are separated into `lists.go`, `sets.go`, `hashes.go`,
and `sortedsets.go` in both `store` and `commands`. `store/expiry.go` is a
placeholder for future expiration handling.

## Tests and checks

```sh
go test ./...
go vet ./...
go build ./...
```

Tests cover RESP parsing, fragmented and pipelined requests, shared client data,
data-type operations, wrong-type errors, concurrent access, manual time, HTTP
request validation, JSON responses, per-connection transaction queues, aborted
transactions, and atomic execution against competing TCP and HTTP clients.

If your Go environment supports cgo and has a C compiler installed, also run:

```sh
go test -race ./...
```

## Current limits

- Data exists only in memory; there is no persistence or expiration.
- Only the command forms listed above are implemented. Advanced options,
  WATCH, blocking commands, pub/sub, and streams are not available.
- The protocol is a subset of RESP2; RESP3 is not supported.
- There is one shared keyspace, with no database selection, authentication, or TLS.
- The server binds to localhost by default and is intended for local learning
  and testing.
