package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/tusharsanka4/mini-miniredis/internal/api"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func TestTransactionQueuesBelongToConnections(t *testing.T) {
	s := &server{store: store.New()}
	first := connect(t, s)
	second := connect(t, s)

	checkReply(t, first, "+OK\r\n", "MULTI")
	checkReply(t, first, "+QUEUED\r\n", "SET", "first", "one")
	// MULTI does not hold the shared lock while waiting for more commands.
	checkReply(t, second, "$-1\r\n", "GET", "first")
	checkReply(t, second, "+OK\r\n", "MULTI")
	checkReply(t, second, "+QUEUED\r\n", "SET", "second", "two")
	checkReply(t, first, "+QUEUED\r\n", "GET", "second")
	checkReply(t, first, "*2\r\n+OK\r\n$-1\r\n", "EXEC")
	checkReply(t, first, "$3\r\none\r\n", "GET", "first")
	checkReply(t, first, "$-1\r\n", "GET", "second")
	checkReply(t, second, "*1\r\n+OK\r\n", "EXEC")
	checkReply(t, first, "$3\r\ntwo\r\n", "GET", "second")

	checkReply(t, first, "+OK\r\n", "MULTI")
	checkReply(t, first, "*0\r\n", "EXEC")
	checkReply(t, first, "+PONG\r\n", "PING")
}

func TestTransactionControlAndErrors(t *testing.T) {
	s := &server{store: store.New()}
	conn := connect(t, s)
	checkReply(t, conn, "-ERR EXEC without MULTI\r\n", "EXEC")
	checkReply(t, conn, "-ERR DISCARD without MULTI\r\n", "DISCARD")
	checkReply(t, conn, "-ERR wrong number of arguments\r\n", "MULTI", "extra")
	checkReply(t, conn, "+OK\r\n", "MULTI")
	checkReply(t, conn, "-ERR MULTI calls can not be nested\r\n", "MULTI")
	checkReply(t, conn, "+QUEUED\r\n", "SET", "discarded", "value")
	checkReply(t, conn, "+OK\r\n", "DISCARD")
	checkReply(t, conn, "$-1\r\n", "GET", "discarded")

	for _, badCommand := range [][]string{{"NOPE"}, {"SET", "key"}, {"EXEC", "extra"}} {
		checkReply(t, conn, "+OK\r\n", "MULTI")
		checkReply(t, conn, "+QUEUED\r\n", "SET", "never-written", "value")
		want := "-ERR wrong number of arguments\r\n"
		if badCommand[0] == "NOPE" {
			want = "-ERR unknown command\r\n"
		}
		checkReply(t, conn, want, badCommand...)
		checkReply(t, conn, "+QUEUED\r\n", "SET", "also-not-written", "value")
		checkReply(t, conn, "-EXECABORT Transaction discarded because of previous errors.\r\n", "EXEC")
		checkReply(t, conn, "$-1\r\n", "GET", "never-written")
		checkReply(t, conn, "$-1\r\n", "GET", "also-not-written")
	}

	// Execution errors are array elements. They do not roll back other commands.
	checkReply(t, conn, "+OK\r\n", "MULTI")
	checkReply(t, conn, "+QUEUED\r\n", "SET", "key", "text")
	checkReply(t, conn, "+QUEUED\r\n", "LPUSH", "key", "item")
	checkReply(t, conn, "+QUEUED\r\n", "ZADD", "scores", "NaN", "member")
	checkReply(t, conn, "+QUEUED\r\n", "SET", "after-error", "done")
	checkReply(t, conn, "*4\r\n+OK\r\n-WRONGTYPE Operation against a key holding the wrong kind of value\r\n-ERR value is not a valid float\r\n+OK\r\n", "EXEC")
	checkReply(t, conn, "$4\r\ntext\r\n", "GET", "key")
	checkReply(t, conn, "$4\r\ndone\r\n", "GET", "after-error")

	// A new transaction works after an aborted or completed one.
	checkReply(t, conn, "+OK\r\n", "multi")
	checkReply(t, conn, "+QUEUED\r\n", "RPUSH", "list", "a", "b")
	checkReply(t, conn, "+QUEUED\r\n", "LRANGE", "list", "0", "-1")
	checkReply(t, conn, "*2\r\n:2\r\n*2\r\n$1\r\na\r\n$1\r\nb\r\n", "exec")
}

func TestDisconnectDropsTransaction(t *testing.T) {
	s := &server{store: store.New()}
	client, conn := net.Pipe()
	done := make(chan struct{})
	go func() {
		s.handleConnection(conn)
		close(done)
	}()
	t.Cleanup(func() { client.Close() })
	checkReply(t, client, "+OK\r\n", "MULTI")
	checkReply(t, client, "+QUEUED\r\n", "SET", "key", "never-written")
	client.Close()
	<-done
	checkReply(t, connect(t, s), "$-1\r\n", "GET", "key")
}

func TestPipelinedTransaction(t *testing.T) {
	s := &server{store: store.New()}
	conn := connect(t, s)
	request := command("MULTI") + command("SET", "key", "value") + command("GET", "key") + command("EXEC") + command("PING")
	done := make(chan error, 1)
	go func() {
		_, err := io.WriteString(conn, request)
		done <- err
	}()
	want := "+OK\r\n+QUEUED\r\n+QUEUED\r\n*2\r\n+OK\r\n$5\r\nvalue\r\n+PONG\r\n"
	reply := make([]byte, len(want))
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if string(reply) != want {
		t.Fatalf("got %q, want %q", reply, want)
	}
}

func TestTransactionsDoNotInterleaveWithTCPOrHTTP(t *testing.T) {
	s := &server{store: store.New()}
	var workers sync.WaitGroup
	// Competing transactions all write and read the same key. Each must see its
	// own write, even while other transactions and ordinary HTTP commands run.
	for i := 0; i < 8; i++ {
		conn := connect(t, s)
		id := fmt.Sprintf("client%d", i)
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 30; j++ {
				checkReply(t, conn, "+OK\r\n", "MULTI")
				checkReply(t, conn, "+QUEUED\r\n", "SET", "shared", id)
				checkReply(t, conn, "+QUEUED\r\n", "GET", "shared")
				want := fmt.Sprintf("*2\r\n+OK\r\n$%d\r\n%s\r\n", len(id), id)
				checkReply(t, conn, want, "EXEC")
			}
		}()
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		handler := api.NewHandler(s.store)
		for i := 0; i < 300; i++ {
			r := httptest.NewRequest(http.MethodPost, "/command", strings.NewReader(`{"command":["SET","shared","http"]}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("HTTP write failed: %s", w.Body)
				return
			}
		}
	}()
	workers.Wait()
}
