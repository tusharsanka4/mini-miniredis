package server

import (
	"io"
	"net"
	"sync"
	"time"
)

const outgoingQueueSize = 64

type outgoingReply struct {
	data     string
	finished chan error // Only command replies need to wait for the write.
}

type client struct {
	conn      net.Conn
	outgoing  chan outgoingReply
	done      chan struct{}
	closeOnce sync.Once
}

func newClient(conn net.Conn) *client {
	c := &client{
		conn:     conn,
		outgoing: make(chan outgoingReply, outgoingQueueSize),
		done:     make(chan struct{}),
	}
	go c.writeLoop()
	return c
}

// Send is used by the subscriber registry. It never waits on network I/O.
func (c *client) Send(reply string) bool {
	return c.enqueue(outgoingReply{data: reply})
}

func (c *client) enqueue(reply outgoingReply) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.outgoing <- reply:
		return true
	case <-c.done:
		return false
	default:
		// A slow subscriber must not hold up publishers or other subscribers.
		c.close()
		return false
	}
}

func (c *client) writeReply(reply string) bool {
	finished := make(chan error, 1)
	if !c.enqueue(outgoingReply{data: reply, finished: finished}) {
		return false
	}
	select {
	case err := <-finished:
		return err == nil
	case <-c.done:
		return false
	}
}

// This is the only goroutine that writes to this connection. A queued frame
// is written completely before the next command reply or message can begin.
func (c *client) writeLoop() {
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		case reply := <-c.outgoing:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			_, err := io.WriteString(c.conn, reply.data)
			if reply.finished != nil {
				reply.finished <- err
			}
			if err != nil {
				return
			}
		}
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}
