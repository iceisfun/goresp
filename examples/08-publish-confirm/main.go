// Advanced: confirmed publish — get the subscriber count back.
//
// Fire-and-forget Publish/PublishJSON are non-blocking and don't report who
// received the message. PublishCtx/PublishJSONCtx instead do a synchronous
// request/reply and return the number of clients Redis delivered to, bounded by
// a context. Useful when you need delivery confirmation or want to detect "no
// subscribers".
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"time"

	"github.com/iceisfun/goresp/publish"
)

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	channel := flag.String("channel", "events", "channel to publish to")
	every := flag.Duration("every", time.Second, "publish interval")
	timeout := flag.Duration("timeout", 2*time.Second, "per-publish timeout")
	flag.Parse()

	pub := publish.New(*addr)
	defer pub.Close()

	log.Printf("confirmed-publishing to %q every %s — Ctrl+C to quit", *channel, *every)
	ticker := time.NewTicker(*every)
	defer ticker.Stop()

	for n := 1; ; n++ {
		<-ticker.C

		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		received, err := pub.PublishJSONCtx(ctx, *channel, map[string]any{"id": n})
		cancel()

		switch {
		case err == nil && received == 0:
			log.Printf("#%d published, but no subscribers were listening", n)
		case err == nil:
			log.Printf("#%d delivered to %d subscriber(s)", n, received)
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("#%d timed out waiting for Redis", n)
		default:
			var re *publish.RedisError
			if errors.As(err, &re) {
				log.Printf("#%d rejected by Redis: %v", n, re)
			} else {
				log.Printf("#%d failed: %v", n, err)
			}
		}
	}
}
