// Intermediate: graceful shutdown with context.
//
// signal.NotifyContext turns SIGINT/SIGTERM into context cancellation, and
// WithContext binds the connection's lifetime to it: one Ctrl+C tears everything
// down cleanly.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/iceisfun/goresp/connection"
)

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	channel := flag.String("channel", "events", "channel to subscribe to")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn := connection.New(*addr, connection.HandlerFunc(func(m connection.Message) {
		log.Printf("%s: %s", m.Channel, m.Payload)
	}), connection.WithContext(ctx))
	defer conn.Close()

	conn.Subscribe(*channel)
	log.Printf("listening on %q — Ctrl+C for graceful shutdown", *channel)

	<-ctx.Done()
	log.Println("signal received; shutting down")
}
