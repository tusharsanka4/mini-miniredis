package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func connect(t *testing.T, s *server) net.Conn {
	t.Helper()
	client, conn := net.Pipe()
	client.SetDeadline(time.Now().Add(5 * time.Second))
	go s.handleConnection(conn)
	t.Cleanup(func() { client.Close() })
	return client
}

func command(args ...string) string {
	request := fmt.Sprintf("*%d\r\n", len(args))
	for _, arg := range args {
		request += resp.BulkString(arg)
	}
	return request
}

func checkReply(t *testing.T, conn net.Conn, want string, args ...string) {
	t.Helper()
	if _, err := io.WriteString(conn, command(args...)); err != nil {
		t.Error(err)
		return
	}
	reply := make([]byte, len(want))
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Error(err)
		return
	}
	if string(reply) != want {
		t.Errorf("%q: got %q, want %q", args, reply, want)
	}
}

func TestCommandsAndSharedStore(t *testing.T) {
	s := &server{store: store.New()}
	first := connect(t, s)
	second := connect(t, s)
	checkReply(t, first, "+PONG\r\n", "ping")
	checkReply(t, first, "$2\r\nhi\r\n", "PING", "hi")
	checkReply(t, first, "$3\r\na\x00b\r\n", "ECHO", "a\x00b")
	checkReply(t, first, "+OK\r\n", "SET", "key", "value")
	checkReply(t, second, "$5\r\nvalue\r\n", "GET", "key")
	checkReply(t, second, "+OK\r\n", "SET", "key", "")
	checkReply(t, first, "$0\r\n\r\n", "GET", "key")
	checkReply(t, first, ":1\r\n", "DEL", "key", "key", "missing")
	checkReply(t, second, "$-1\r\n", "GET", "key")
	checkReply(t, first, "-ERR wrong number of arguments\r\n", "SET", "key")
	checkReply(t, first, "-ERR unknown command\r\n", "UNKNOWN")
	checkReply(t, first, "+PONG\r\n", "PING")
}

func TestFragmentedPipeline(t *testing.T) {
	s := &server{store: store.New()}
	conn := connect(t, s)
	request := command("SET", "key", "a\r\nb") + command("GET", "key")
	done := make(chan error, 1)
	go func() {
		for i := range request {
			if _, err := conn.Write([]byte{request[i]}); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	want := "+OK\r\n$4\r\na\r\nb\r\n"
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

func TestInvalidRESP(t *testing.T) {
	for _, input := range []string{
		"PING\r\n", "*0\r\n", "*-1\r\n", "*1025\r\n", "*x\r\n", "*1\n",
		"*1\r\n+PING\r\n", "*1\r\n$-1\r\n", "*1\r\n$1048577\r\n",
		"*1\r\n$4\r\nPIN", "*1\r\n$4\r\nPINGxx",
	} {
		if _, err := resp.ReadCommand(bufio.NewReader(strings.NewReader(input))); err == nil {
			t.Errorf("accepted invalid command %q", input)
		}
	}
	s := &server{store: store.New()}
	conn := connect(t, s)
	io.WriteString(conn, "*0\r\n")
	reply, err := io.ReadAll(conn)
	if err != nil || string(reply) != "-ERR invalid RESP command\r\n" {
		t.Fatalf("got %q, %v", reply, err)
	}
}

func TestTruncatedCommand(t *testing.T) {
	for _, input := range []string{"*", "*1\r\n", "*1\r\n$", "*1\r\n$4\r\n", "*1\r\n$4\r\nPI"} {
		_, err := resp.ReadCommand(bufio.NewReader(strings.NewReader(input)))
		if err != io.ErrUnexpectedEOF {
			t.Errorf("%q: got %v, want unexpected EOF", input, err)
		}
	}
	_, err := resp.ReadCommand(bufio.NewReader(strings.NewReader("")))
	if err != io.EOF {
		t.Errorf("empty connection: got %v, want EOF", err)
	}
}

func TestConcurrentClients(t *testing.T) {
	s := &server{store: store.New()}
	var clients sync.WaitGroup
	for i := 0; i < 8; i++ {
		conn := connect(t, s)
		key := fmt.Sprintf("key%d", i)
		clients.Add(1)
		go func() {
			defer clients.Done()
			for j := 0; j < 20; j++ {
				checkReply(t, conn, "+OK\r\n", "SET", key, "value")
				checkReply(t, conn, "$5\r\nvalue\r\n", "GET", key)
			}
		}()
	}
	clients.Wait()
}
