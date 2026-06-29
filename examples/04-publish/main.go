// Intermediate: publish messages.
//
// Publishing reuses the same connector as subscribing, just a separate instance
// that never subscribes (Redis rejects PUBLISH on a subscribed connection).
// publish.New creates and owns that dedicated connection for you.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/iceisfun/goresp/publish"
)

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	channel := flag.String("channel", "events", "channel to publish to")
	every := flag.Duration("every", time.Second, "publish interval")
	flag.Parse()

	pub := publish.New(*addr)
	defer pub.Close()

	log.Printf("publishing to %q every %s — Ctrl+C to quit", *channel, *every)
	ticker := time.NewTicker(*every)
	defer ticker.Stop()

	for n := 1; ; n++ {
		<-ticker.C
		if err := pub.PublishJSON(*channel, map[string]any{"id": n, "name": "tick"}); err != nil {
			log.Printf("publish failed: %v", err)
		}
	}
}
