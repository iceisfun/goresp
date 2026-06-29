// Advanced: decouple processing with a worker pool for high throughput.
//
// OnMessage is called synchronously from the read loop, so heavy per-message
// work would stall parsing. Here OnMessage only hands the message to a buffered
// channel drained by a pool of workers. The handoff is a blocking send: if the
// pool falls behind, backpressure propagates to the socket (TCP flow control)
// rather than silently dropping messages — appropriate for 5–25k msg/s.
package main

import (
	"flag"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/iceisfun/goresp/connection"
)

type pool struct {
	jobs chan connection.Message
	done atomic.Uint64
}

func (p *pool) OnMessage(m connection.Message) {
	p.jobs <- m // blocks when saturated -> backpressure, never drops
}

func (p *pool) worker() {
	for m := range p.jobs {
		// Real work goes here: parse JSON, write to a DB, fan out, ...
		_ = m
		p.done.Add(1)
	}
}

// subscriber re-establishes the subscription on every connect.
type subscriber struct{ pattern string }

func (s subscriber) OnConnect(sub connection.Subscriber) { sub.PSubscribe(s.pattern) }
func (subscriber) OnDisconnect(error)                    {}
func (subscriber) OnError(error)                         {}

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	pattern := flag.String("pattern", "*", "pattern to psubscribe")
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	queue := flag.Int("queue", 8192, "job queue depth")
	flag.Parse()

	p := &pool{jobs: make(chan connection.Message, *queue)}
	for range *workers {
		go p.worker()
	}

	conn := connection.New(*addr, p, connection.WithEvents(subscriber{*pattern}))
	defer conn.Close()

	log.Printf("%d workers draining %q (queue depth %d)", *workers, *pattern, *queue)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var last uint64
	for range ticker.C {
		now := p.done.Load()
		log.Printf("%6d msg/s   queue %d/%d", now-last, len(p.jobs), cap(p.jobs))
		last = now
	}
}
