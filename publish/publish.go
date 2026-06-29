// Package publish is the producer side of goresp. It reuses the very same
// connection.Reconnecting connector used for consuming — just a separate
// instance that never subscribes.
//
// Redis requires this: once a connection issues SUBSCRIBE it enters subscribe
// mode and rejects PUBLISH, so publishing and subscribing always live on
// different connections. That is a Redis rule, not a goresp limitation, and it
// needs no extra dependency — a second Reconnecting is all it takes.
package publish

import (
	"encoding/json"
	"errors"

	"github.com/iceisfun/goresp/command"
	"github.com/iceisfun/goresp/connection"
)

// ErrQueueFull is returned when the underlying connection's outbound queue is
// full (the publish is dropped rather than blocking).
var ErrQueueFull = errors.New("publish: outbound queue full")

// Sender is the subset of connection.Reconnecting that Publisher needs. It lets
// callers wrap an existing connection or a fake in tests.
type Sender interface {
	Send(cmd []byte) bool
}

// Publisher publishes messages over a (non-subscribed) Reconnecting connection.
// It adds no buffering of its own: Reconnecting.Send already queues commands and
// writes them from its own goroutine.
type Publisher struct {
	sender Sender
	owned  *connection.Reconnecting // non-nil when this Publisher created the connection
}

// noopHandler satisfies connection.Handler for a publish-only connection, which
// never receives pub/sub deliveries.
type noopHandler struct{}

func (noopHandler) OnMessage(connection.Message) {}

// New creates a Publisher backed by its own dedicated publish connection to
// addr. The connection auto-reconnects like any other; pass connection options
// (WithEvents, WithTimeouts, ...) to configure it. Call Close to shut it down.
func New(addr string, opts ...connection.Option) *Publisher {
	conn := connection.New(addr, noopHandler{}, opts...)
	return &Publisher{sender: conn, owned: conn}
}

// Wrap returns a Publisher over an existing Sender (typically a second
// *connection.Reconnecting you manage yourself, or a fake in tests). Close does
// not affect a wrapped sender's lifecycle.
func Wrap(sender Sender) *Publisher {
	return &Publisher{sender: sender}
}

// Conn returns the owned connection, or nil when the Publisher wraps an external
// Sender. Useful for inspecting connection state or attaching event handlers.
func (p *Publisher) Conn() *connection.Reconnecting {
	return p.owned
}

// Publish sends a raw payload on channel. It does not block; it returns
// ErrQueueFull if the connection's outbound queue is saturated.
func (p *Publisher) Publish(channel string, payload []byte) error {
	if !p.sender.Send(command.FormatCommand("PUBLISH", channel, string(payload))) {
		return ErrQueueFull
	}
	return nil
}

// PublishJSON marshals v to JSON and publishes it on channel.
func (p *Publisher) PublishJSON(channel string, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return p.Publish(channel, payload)
}

// Close shuts down the connection if this Publisher owns it; it is a no-op for a
// wrapped Sender.
func (p *Publisher) Close() {
	if p.owned != nil {
		p.owned.Close()
	}
}
