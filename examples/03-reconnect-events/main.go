// Intermediate: lifecycle events and reconnect-safe subscriptions.
//
// One struct implements both Handler (OnMessage) and EventHandler
// (OnConnect/OnDisconnect/OnError). Subscribing inside OnConnect guarantees the
// subscriptions are (re)established every time the link comes up — kill the
// Redis connection and watch it heal.
package main

import (
	"flag"
	"log"
	"strings"

	"github.com/iceisfun/goresp/connection"
)

type app struct {
	patterns []string
}

func (a *app) OnMessage(m connection.Message) {
	log.Printf("[%s] %s: %s", m.Kind, m.Channel, m.Payload)
}

func (a *app) OnConnect(s connection.Subscriber) {
	log.Println("connected — (re)subscribing")
	for _, p := range a.patterns {
		s.PSubscribe(p)
	}
}

func (a *app) OnDisconnect(err error) { log.Printf("disconnected: %v", err) }
func (a *app) OnError(err error)      { log.Printf("error: %v", err) }

func main() {
	addr := flag.String("redis", "127.0.0.1:6379", "redis address")
	patterns := flag.String("patterns", "events.*", "comma-separated patterns to psubscribe")
	flag.Parse()

	a := &app{patterns: strings.Split(*patterns, ",")}
	conn := connection.New(*addr, a, connection.WithEvents(a))
	defer conn.Close()

	log.Println("running — drop the Redis connection to watch reconnect + resubscribe")
	select {}
}
