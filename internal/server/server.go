package server

import (
	"fmt"
	"net"

	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

type server struct {
	store *store.Store
}

func Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	fmt.Println("MiniRedis listening on", listener.Addr())
	s := &server{store: store.New()}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}
		go s.handleConnection(conn)
	}
}
