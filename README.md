# Mini-Redis

This project involves building a lightweight, server-based storage/caching system that can be deployed alongside your project to test its Redis compatibility and readiness in an easier format.

It is generally used for unit testing and has its own TCP interface while supporting a wide range of Redis commands.

## Architecture

The architecture is fairly simple.

1. The Redis client connects to the server/socket via a TCP listener, generally over port `6379`.

2. This then creates a **Connection Handler**, which is responsible for reading client data. It is important to note that every client gets its own Connection Handler so that concurrent commands are easier to handle.

3. The client data is then passed through a **RESP Parser**, which is RESPonsible for following and decoding Redis commands encoded using the Redis Serialization Protocol (RESP).

4. The commands are then identified and passed to a **Command Dispatcher**. This part identifies commands such as `SET`, `GET`, `DEL`, etc.

5. Based on these commands, the program branches to its **Storage Engine**, which is responsible for actually storing the data using various Go-supported data types and data structures for further access. This database is protected by mutexes for concurrency control, ensuring that there are no faulty data reads or writes.

6. Accordingly, the results are then returned to the client as per their requirement.

---

## TCP Server

The TCP server uses Go's built-in `net` package.

- `net.Listen()` — Opens a listening socket.
- `listener.Accept()` — Waits for new connections.
- `conn.Read()` — Receives bytes over an accepted connection.
- `conn.Write()` — Writes data back to the client accordingly.

### Goroutines and How They Help

Each client gets its own goroutine, making it so that every client has an independent Connection Handler while still accessing the same underlying data storage.

This allows multiple clients to communicate with the server concurrently.

## RESP Parser

TCP is only responsible for sending and receiving data bytes. These data bytes are encoded using RESP when communicating with Redis clients.

The RESP Parser makes sure that the actual data/Redis commands are retrieved from these encoded data bytes.

TCP can deliver the data in complete chunks or partial chunks, depending on how the bytes arrive.

This is why the RESP protocol and parser are important, as they make sure that complete commands are correctly identified and decoded, regardless of how the bytes arrive.

## Command Dispatcher

The Command Dispatcher makes sure that the extracted commands are executed accordingly.

The RESP Parser receives encoded data and outputs Redis commands from it. The Command Dispatcher then identifies these commands and makes sure the corresponding operations are executed.

## Storage Engine

This is how the program actually stores data.

Currently, there is no data persistence. That can be added later as per necessity.

The Storage Engine guides the data from the commands to their respective storage locations using Go data structures, as per requirement.

It also uses mutexes to ensure safe concurrent access to the stored data.

---

## Controllable Clock

A clock that can be controlled to manipulate the flow of time within the project, allowing us to simulate key expirations and test time-dependent behavior without actually waiting.

This is particularly useful for unit testing.

## Transactions

By using the `MULTI` and `EXEC` commands, you can define a set of Redis commands that will be executed like a transaction (atomically), without allowing commands from other clients to slip in between.

This makes use of mutexes, and every client gets its own transaction queue.

Commands are queued after `MULTI` and executed together when `EXEC` is called.

## Publisher/Subscriber Messaging

Clients can connect or `SUBSCRIBE` to particular channels maintained by the Redis server.

Publisher clients can then send messages to these channels using the `PUBLISH` command.

The server maintains a **Subscriber Registry** (stored using Go maps), which keeps track of the clients subscribed to each channel.

When a message is published, it is forwarded to the relevant subscribed clients.

Concurrency control is used here to make sure that messages being sent don't interfere with the data bytes and responses being sent to each client.

Unlike stored Redis data, Pub/Sub messages are not saved for later use. They are delivered to the clients currently subscribed to the channel.

---

## Project Goal

The goal is to build a lightweight Redis-compatible testing server in Go, while understanding the underlying networking, concurrency, command processing, and data storage mechanisms.

The project is being developed incrementally, with more Redis commands, data structures, and testing features being added along the way.