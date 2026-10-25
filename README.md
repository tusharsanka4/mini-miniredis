# mini-miniredis

Run with `go run ./cmd/miniredis`. The server listens on `127.0.0.1:6379`.

Code layout:

- `cmd/miniredis/main.go`: starts the server.
- `internal/server`: TCP listener and client connection handler.
- `internal/resp`: command parser and reply formatting.
- `internal/store`: shared map and its mutex. `expiry.go` is a placeholder for later.
- `internal/commands`: command dispatcher and string command handlers.

Each client gets a `handleConnection` goroutine. It reads RESP commands through
a buffered reader backed by `conn.Read`, dispatches them, and uses `conn.Write`
for replies. All clients share a mutex-protected `map[string]string`.

Supported commands: `PING [message]`, `ECHO message`, `SET key value`, `GET key`,
and `DEL key [key ...]`. Command names are case insensitive. A missing key returns
a RESP null bulk string; `DEL` returns the number of keys deleted.

Requests must be RESP arrays of bulk strings (the format used by Redis clients).
For example, `GET key` is `*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n` on the wire.
Commands may arrive in pieces or be pipelined on one connection. Arguments can
contain arbitrary bytes, including CRLF. Requests allow up to 1024 arguments and
1 MiB per argument. Invalid RESP gets an error and closes the connection;
unknown commands and wrong argument counts get an error and keep it open.

Values live in memory until deleted or the process stops. There is no expiration
or persistence yet.

Run tests with `go test ./...` (or `go test -race ./...` with a C compiler installed).
