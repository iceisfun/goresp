package connection_test

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iceisfun/goresp/connection"
)

// collector implements connection.Handler and connection.EventHandler. It
// records messages and lifecycle events, and resubscribes on every connect.
type collector struct {
	mu          sync.Mutex
	msgs        chan connection.Message
	connects    atomic.Int32
	disconnects atomic.Int32
	errors      atomic.Int32
}

func newCollector() *collector { return &collector{msgs: make(chan connection.Message, 16)} }

func (c *collector) OnMessage(m connection.Message) { c.msgs <- m }
func (c *collector) OnConnect(s connection.Subscriber) {
	c.connects.Add(1)
	s.Subscribe("foo")
}
func (c *collector) OnDisconnect(error) { c.disconnects.Add(1) }
func (c *collector) OnError(error)      { c.errors.Add(1) }

// TestReconnectClearsPartialState drives the real connection against a fake
// server. The first connection delivers a truncated frame and drops; the second
// delivers a complete frame. The client must discard the partial bytes on
// disconnect (decoder.Reset) so the second frame decodes cleanly.
func TestReconnectClearsPartialState(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var connCount atomic.Int32
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			n := connCount.Add(1)
			go drainReads(conn) // discard the client's SUBSCRIBE/PING writes

			if n == 1 {
				// Truncated frame: declares a 100-byte payload, sends 4, then
				// abruptly closes.
				io.WriteString(conn, "*3\r\n$7\r\nmessage\r\n$3\r\nfoo\r\n$100\r\nhalf")
				time.Sleep(20 * time.Millisecond)
				conn.Close()
				continue
			}
			// Complete frame on the reconnect.
			io.WriteString(conn, "*3\r\n$7\r\nmessage\r\n$3\r\nfoo\r\n$5\r\nhello\r\n")
			// Hold the connection open until the test tears down.
			<-serverHold
		}
	}()

	c := newCollector()
	conn := connection.New(ln.Addr().String(), c,
		connection.WithEvents(c),
		connection.WithKeepAliveInterval(50*time.Millisecond),
		connection.WithTimeouts(time.Second, time.Second, time.Second),
	)
	defer conn.Close()

	select {
	case msg := <-c.msgs:
		if msg.Kind != "message" || msg.Channel != "foo" || string(msg.Payload) != "hello" {
			t.Fatalf("got %+v, want message/foo/hello", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message after reconnect")
	}

	close(serverHold)

	if c.connects.Load() < 2 {
		t.Errorf("expected at least 2 connects, got %d", c.connects.Load())
	}
	if c.disconnects.Load() < 1 {
		t.Errorf("expected at least 1 disconnect, got %d", c.disconnects.Load())
	}
}

// TestContextShutdown verifies that cancelling the bound context tears the
// connection down (an established connection fires OnDisconnect).
func TestContextShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan struct{}, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			select {
			case accepted <- struct{}{}:
			default:
			}
			go drainReads(conn)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	c := newCollector()
	conn := connection.New(ln.Addr().String(), c,
		connection.WithEvents(c),
		connection.WithContext(ctx),
		connection.WithKeepAlive(false),
	)
	defer conn.Close()

	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("server never accepted a connection")
	}

	cancel()

	deadline := time.After(2 * time.Second)
	for c.disconnects.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("context cancel did not disconnect the connection")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

var serverHold = make(chan struct{})

func drainReads(conn net.Conn) {
	buf := make([]byte, 4096)
	for {
		if _, err := conn.Read(buf); err != nil {
			return
		}
	}
}
