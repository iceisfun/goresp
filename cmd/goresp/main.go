// Command goresp is a small example consumer: it subscribes to one or more
// channels and prints each message. It demonstrates the Handler and
// EventHandler dependency-injection points. The library itself does no logging;
// this example uses the standard library log package.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/iceisfun/goresp/connection"
)

// printHandler implements connection.Handler.
type printHandler struct{}

func (printHandler) OnMessage(m connection.Message) {
	if m.Pattern != "" {
		log.Printf("[%s] (%s) %s: %s", m.Kind, m.Pattern, m.Channel, m.Payload)
		return
	}
	log.Printf("[%s] %s: %s", m.Kind, m.Channel, m.Payload)
}

// logEvents implements connection.EventHandler. It (re)subscribes on every
// connect, so subscriptions survive reconnects automatically.
type logEvents struct {
	channels []string
}

func (e logEvents) OnConnect(s connection.Subscriber) {
	log.Println("connected")
	for _, channel := range e.channels {
		if strings.Contains(channel, "*") {
			s.PSubscribe(channel)
		} else {
			s.Subscribe(channel)
		}
	}
}
func (logEvents) OnDisconnect(err error) { log.Printf("disconnected: %v", err) }
func (logEvents) OnError(err error)      { log.Printf("error: %v", err) }

func main() {
	redisAddr := flag.String("redis", "127.0.0.1:6379", "Redis server address")
	channelsFlag := flag.String("channels", "*", "Comma-separated channels (a '*' triggers a pattern subscribe)")
	flag.Parse()

	var channels []string
	for channel := range strings.SplitSeq(*channelsFlag, ",") {
		if channel = strings.TrimSpace(channel); channel != "" {
			channels = append(channels, channel)
		}
	}

	conn := connection.New(*redisAddr, printHandler{}, connection.WithEvents(logEvents{channels: channels}))
	defer conn.Close()

	log.Printf("listening on %s; press Ctrl+C to exit", *redisAddr)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
}
