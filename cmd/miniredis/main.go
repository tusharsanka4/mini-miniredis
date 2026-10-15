package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:6379")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println("mini-miniredis listening on ", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept failed: ", err)
			continue
		}

		fmt.Println("Client Connected: ", conn.RemoteAddr())
		conn.Close()
	}

}
