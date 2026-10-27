package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tusharsanka4/mini-miniredis/internal/api"
	"github.com/tusharsanka4/mini-miniredis/internal/commands"
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func subscriptionAck(kind, channel string, count int) string {
	return fmt.Sprintf("*3\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n:%d\r\n", len(kind), kind, len(channel), channel, count)
}

func readExpected(t *testing.T, conn net.Conn, want string) {
	t.Helper()
	data := make([]byte, len(want))
	if _, err := io.ReadFull(conn, data); err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("got %q, want %q", data, want)
	}
}

func TestPubSubFanoutAndUnsubscribe(t *testing.T) {
	s := &server{store: store.New()}
	first := connect(t, s)
	second := connect(t, s)
	publisher := connect(t, s)
	checkReply(t, first, subscriptionAck("subscribe", "news", 1)+subscriptionAck("subscribe", "other", 2), "SUBSCRIBE", "news", "other")
	checkReply(t, second, subscriptionAck("subscribe", "news", 1), "SUBSCRIBE", "news")
	// Subscribing twice acknowledges twice but does not duplicate delivery.
	checkReply(t, first, subscriptionAck("subscribe", "news", 2), "SUBSCRIBE", "news")
	checkReply(t, publisher, ":0\r\n", "PUBLISH", "nobody", "hello")
	checkReply(t, publisher, ":2\r\n", "PUBLISH", "news", "a\r\n\x00b")
	message := resp.Array([]string{"message", "news", "a\r\n\x00b"})
	readExpected(t, first, message)
	readExpected(t, second, message)
	checkReply(t, publisher, ":1\r\n", "PUBLISH", "other", "only-first")
	readExpected(t, first, resp.Array([]string{"message", "other", "only-first"}))

	checkReply(t, first, resp.Array([]string{"pong", ""}), "PING")
	checkReply(t, first, resp.Array([]string{"pong", "hi"}), "PING", "hi")
	checkReply(t, first, "-ERR only SUBSCRIBE, UNSUBSCRIBE and PING are allowed while subscribed\r\n", "GET", "key")
	checkReply(t, first, subscriptionAck("unsubscribe", "missing", 2), "UNSUBSCRIBE", "missing")
	checkReply(t, first, subscriptionAck("unsubscribe", "news", 1), "UNSUBSCRIBE", "news")
	checkReply(t, publisher, ":1\r\n", "PUBLISH", "news", "only-second")
	readExpected(t, second, resp.Array([]string{"message", "news", "only-second"}))
	checkReply(t, first, subscriptionAck("unsubscribe", "other", 0), "UNSUBSCRIBE")
	checkReply(t, first, "+PONG\r\n", "PING")
	checkReply(t, first, "*3\r\n$11\r\nunsubscribe\r\n$-1\r\n:0\r\n", "UNSUBSCRIBE")
	checkReply(t, second, subscriptionAck("unsubscribe", "news", 0), "UNSUBSCRIBE")
	checkReply(t, publisher, ":0\r\n", "PUBLISH", "news", "gone")
}

func TestHTTPAndTransactionPublish(t *testing.T) {
	s := &server{store: store.New()}
	subscriber := connect(t, s)
	publisher := connect(t, s)
	checkReply(t, subscriber, subscriptionAck("subscribe", "news", 1), "SUBSCRIBE", "news")
	r := httptest.NewRequest(http.MethodPost, "/command", strings.NewReader(`{"command":["PUBLISH","news","http"]}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.NewHandler(s.store).ServeHTTP(w, r)
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"result":1}` {
		t.Fatalf("HTTP publish: %d %s", w.Code, w.Body)
	}
	readExpected(t, subscriber, resp.Array([]string{"message", "news", "http"}))
	checkReply(t, publisher, "+OK\r\n", "MULTI")
	checkReply(t, publisher, "+QUEUED\r\n", "PUBLISH", "news", "first")
	checkReply(t, publisher, "+QUEUED\r\n", "PUBLISH", "news", "second")
	checkReply(t, publisher, "*2\r\n:1\r\n:1\r\n", "EXEC")
	readExpected(t, subscriber, resp.Array([]string{"message", "news", "first"}))
	readExpected(t, subscriber, resp.Array([]string{"message", "news", "second"}))
	checkReply(t, publisher, "+OK\r\n", "MULTI")
	checkReply(t, publisher, "-ERR SUBSCRIBE and UNSUBSCRIBE are not supported inside MULTI\r\n", "SUBSCRIBE", "news")
	checkReply(t, publisher, "-EXECABORT Transaction discarded because of previous errors.\r\n", "EXEC")
}

func TestManySubscriptionsInOneCommand(t *testing.T) {
	s := &server{store: store.New()}
	subscriber := connect(t, s)
	args := []string{"SUBSCRIBE"}
	want := ""
	for i := 0; i < outgoingQueueSize+10; i++ {
		channel := fmt.Sprintf("channel%03d", i)
		args = append(args, channel)
		want += subscriptionAck("subscribe", channel, i+1)
	}
	checkReply(t, subscriber, want, args...)
	want = ""
	for i, channel := range args[1:] {
		want += subscriptionAck("unsubscribe", channel, len(args)-2-i)
	}
	checkReply(t, subscriber, want, "UNSUBSCRIBE")
	// An empty channel name is a valid string, different from the null channel
	// in the acknowledgement for UNSUBSCRIBE with no current subscriptions.
	checkReply(t, subscriber, subscriptionAck("subscribe", "", 1), "SUBSCRIBE", "")
	checkReply(t, subscriber, subscriptionAck("unsubscribe", "", 0), "UNSUBSCRIBE", "")
	checkReply(t, subscriber, "+PONG\r\n", "PING")
}

func TestSlowSubscriberDoesNotBlockPublishers(t *testing.T) {
	s := &server{store: store.New()}
	subscriber, conn := net.Pipe()
	subscriber.SetDeadline(time.Now().Add(5 * time.Second))
	done := make(chan struct{})
	go func() {
		s.handleConnection(conn)
		close(done)
	}()
	t.Cleanup(func() { subscriber.Close() })
	checkReply(t, subscriber, subscriptionAck("subscribe", "news", 1), "SUBSCRIBE", "news")
	// Stop reading. More messages than the bounded queue can hold closes this
	// subscriber instead of making a publisher wait for its network connection.
	for i := 0; i < outgoingQueueSize+5; i++ {
		commands.Dispatch(s.store, []string{"PUBLISH", "news", "unread"})
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("slow subscriber was not disconnected")
	}
	if got := commands.Dispatch(s.store, []string{"PUBLISH", "news", "gone"}); got != ":0\r\n" {
		t.Fatalf("disconnected subscriber remained in registry: %q", got)
	}
	checkReply(t, connect(t, s), "+PONG\r\n", "PING")
}

func TestDisconnectRemovesSubscriptions(t *testing.T) {
	s := &server{store: store.New()}
	subscriber, conn := net.Pipe()
	subscriber.SetDeadline(time.Now().Add(5 * time.Second))
	done := make(chan struct{})
	go func() {
		s.handleConnection(conn)
		close(done)
	}()
	t.Cleanup(func() { subscriber.Close() })
	checkReply(t, subscriber, subscriptionAck("subscribe", "a", 1)+subscriptionAck("subscribe", "b", 2), "SUBSCRIBE", "a", "b")
	subscriber.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("connection handler did not stop")
	}
	for _, channel := range []string{"a", "b"} {
		if got := commands.Dispatch(s.store, []string{"PUBLISH", channel, "gone"}); got != ":0\r\n" {
			t.Errorf("subscription to %s survived disconnect: %q", channel, got)
		}
	}
}

func TestMessagesAndCommandRepliesShareOneWriter(t *testing.T) {
	s := &server{store: store.New()}
	subscriber := connect(t, s)
	checkReply(t, subscriber, subscriptionAck("subscribe", "news", 1), "SUBSCRIBE", "news")
	const count = 30
	want := make(map[string]bool)
	for i := 0; i < count; i++ {
		text := fmt.Sprintf("item%d\r\n", i)
		want[resp.Array([]string{"message", "news", text})] = true
		want[resp.Array([]string{"pong", text})] = true
	}
	published := make(chan error, 1)
	go func() {
		for i := 0; i < count; i++ {
			text := fmt.Sprintf("item%d\r\n", i)
			if got := commands.Dispatch(s.store, []string{"PUBLISH", "news", text}); got != ":1\r\n" {
				published <- fmt.Errorf("publish failed: %q", got)
				return
			}
		}
		published <- nil
	}()
	pinged := make(chan error, 1)
	go func() {
		for i := 0; i < count; i++ {
			if _, err := io.WriteString(subscriber, command("PING", fmt.Sprintf("item%d\r\n", i))); err != nil {
				pinged <- err
				return
			}
		}
		pinged <- nil
	}()
	reader := bufio.NewReader(subscriber)
	for i := 0; i < count*2; i++ {
		reply, err := readFrame(reader)
		if err != nil {
			t.Fatal(err)
		}
		if !want[reply] {
			t.Fatalf("unexpected, duplicated, or interleaved frame: %q", reply)
		}
		delete(want, reply)
	}
	if err := <-published; err != nil {
		t.Fatal(err)
	}
	if err := <-pinged; err != nil {
		t.Fatal(err)
	}
}

// Read whole frames to detect byte-level corruption between concurrent replies.
func readFrame(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil || len(line) < 3 || !strings.HasSuffix(line, "\r\n") {
		return "", fmt.Errorf("invalid frame header %q: %v", line, err)
	}
	if line[0] != '$' && line[0] != '*' {
		return line, nil
	}
	length, err := strconv.Atoi(line[1 : len(line)-2])
	if err != nil || length < -1 {
		return "", fmt.Errorf("invalid frame length %q", line)
	}
	if length == -1 {
		return line, nil
	}
	if line[0] == '$' {
		data := make([]byte, length+2)
		_, err := io.ReadFull(reader, data)
		return line + string(data), err
	}
	for i := 0; i < length; i++ {
		part, err := readFrame(reader)
		if err != nil {
			return "", err
		}
		line += part
	}
	return line, nil
}
