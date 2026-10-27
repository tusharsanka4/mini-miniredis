package server

import (
	"bufio"
	"io"
	"net"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
)

func (s *server) handleConnection(conn net.Conn) {
	c := newClient(conn)
	defer c.close()
	defer s.store.PubSub.Remove(c)

	// The buffered reader calls conn.Read and keeps bytes for the next command.
	reader := bufio.NewReader(conn)
	// Only this connection's goroutine can access its transaction queue.
	var tx transaction
	for {
		args, err := resp.ReadCommand(reader)
		if err != nil {
			if err != io.EOF {
				c.writeReply(resp.Error("invalid RESP command"))
			}
			return
		}

		reply := c.handle(s.store, &tx, args)
		if reply != "" && !c.writeReply(reply) {
			return
		}
	}
}
