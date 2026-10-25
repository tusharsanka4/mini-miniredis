package main

import (
	"log"

	"github.com/tusharsanka4/mini-miniredis/internal/server"
)

func main() {
	if err := server.Start("127.0.0.1:6379"); err != nil {
		log.Fatal(err)
	}
}
