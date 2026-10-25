package server

import (
	"bufio"
	"io"
	"net"

	"github.com/tusharsanka4/mini-miniredis/internal/commands"
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
)

func (s *server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// The buffered reader calls conn.Read and keeps bytes for the next command.
	reader := bufio.NewReader(conn)
	for {
		args, err := resp.ReadCommand(reader)
		if err != nil {
			if err != io.EOF {
				conn.Write([]byte(resp.Error("invalid RESP command")))
			}
			return
		}

		if _, err := conn.Write([]byte(commands.Dispatch(s.store, args))); err != nil {
			return
		}
	}
}
