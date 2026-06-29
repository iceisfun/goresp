// Beginner: subscribe to a channel and print every message.
//
// The simplest possible consumer. A HandlerFunc adapts a plain function to the
// connection.Handler interface; Subscribe remembers the channel so it is
// restored automatically across reconnects.
package main

import (
	"flag"
	"log"

	"github.com/iceisfun/goresp/connection"
)

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	channel := flag.String("channel", "news", "channel to subscribe to")
	flag.Parse()

	conn := connection.New(*addr, connection.HandlerFunc(func(m connection.Message) {
		log.Printf("%s: %s", m.Channel, m.Payload)
	}))
	defer conn.Close()

	conn.Subscribe(*channel)
	log.Printf("subscribed to %q on %s — Ctrl+C to quit", *channel, *addr)

	select {} // block forever
}
