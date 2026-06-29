// Beginner+: decode JSON payloads into a struct.
//
// Identical in shape to example 01, but uses Message.JSON to unmarshal each
// payload. The library imposes no schema — you decide how to interpret bytes.
package main

import (
	"flag"
	"log"

	"github.com/iceisfun/goresp/connection"
)

type event struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	channel := flag.String("channel", "events", "channel to subscribe to")
	flag.Parse()

	conn := connection.New(*addr, connection.HandlerFunc(func(m connection.Message) {
		var e event
		if err := m.JSON(&e); err != nil {
			log.Printf("bad payload on %s: %v", m.Channel, err)
			return
		}
		log.Printf("event #%d %q", e.ID, e.Name)
	}))
	defer conn.Close()

	conn.Subscribe(*channel)
	log.Printf("decoding JSON from %q — Ctrl+C to quit", *channel)

	select {}
}
