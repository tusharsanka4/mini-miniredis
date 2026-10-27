package main

import (
	"flag"
	"log"

	"github.com/tusharsanka4/mini-miniredis/internal/api"
	"github.com/tusharsanka4/mini-miniredis/internal/server"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func main() {
	redisAddress := flag.String("redis-addr", "127.0.0.1:6379", "Redis TCP listen address")
	httpAddress := flag.String("http-addr", "127.0.0.1:8080", "HTTP API listen address")
	flag.Parse()

	db := store.New()
	errors := make(chan error, 2)
	go func() { errors <- server.Start(*redisAddress, db) }()
	go func() { errors <- api.Start(*httpAddress, db) }()
	log.Fatal(<-errors)
}
