// Package publish is the producer side of goresp. It reuses the very same
// connection.Reconnecting connector used for consuming — just a separate
// instance that never subscribes.
//
// Redis requires this: once a connection issues SUBSCRIBE it enters subscribe
// mode and rejects PUBLISH, so publishing and subscribing always live on
// different connections. That is a Redis rule, not a goresp limitation, and it
// needs no extra dependency — a second Reconnecting is all it takes.
//
// Two publish flavors are offered:
//
//   - Fire-and-forget (Publish, PublishJSON): non-blocking, best-effort. The
//     command is queued on the async connection and the Redis reply (the
//     subscriber count) is not surfaced. Best for high-rate, don't-care-who-got-it
//     publishing.
//   - Confirmed (PublishCtx, PublishJSONCtx): synchronous request/reply over a
//     dedicated connection. Returns the number of clients that received the
//     message and any Redis error reply, bounded by the supplied context.
package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/iceisfun/goresp/command"
	"github.com/iceisfun/goresp/connection"
	"github.com/iceisfun/goresp/resp"
)

// ErrQueueFull is returned by the fire-and-forget methods when the underlying
// connection's outbound queue is full (the publish is dropped rather than
// blocking).
var ErrQueueFull = errors.New("publish: outbound queue full")

// ErrConfirmUnavailable is returned by the confirmed (Ctx) methods on a
// Publisher created with Wrap, which has no address to open a request/reply
// connection. Use New for confirmed publishing.
var ErrConfirmUnavailable = errors.New("publish: confirmed publish requires publish.New")

// RedisError wraps an error reply returned by Redis (e.g. wrong arity, or a
// cluster CROSSSLOT).
type RedisError struct{ Msg string }

func (e *RedisError) Error() string { return "redis: " + e.Msg }

// Sender is the subset of connection.Reconnecting that fire-and-forget
// publishing needs. It lets callers wrap an existing connection or a fake in
// tests.
type Sender interface {
	Send(cmd []byte) bool
}

// Publisher publishes messages over (non-subscribed) connections. Fire-and-forget
// publishes go through an async connection; confirmed publishes use a separate,
// lazily-opened synchronous connection.
type Publisher struct {
	sender Sender
	owned  *connection.Reconnecting // non-nil when this Publisher created the async connection

	addr        string // empty for a Wrap'd Publisher; disables confirmed publishing
	dialTimeout time.Duration

	scMu sync.Mutex
	sc   *syncConn
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
	return &Publisher{sender: conn, owned: conn, addr: addr, dialTimeout: 10 * time.Second}
}

// Wrap returns a Publisher over an existing Sender (typically a second
// *connection.Reconnecting you manage yourself, or a fake in tests). Confirmed
// publishing is unavailable on a wrapped Publisher (see ErrConfirmUnavailable);
// Close does not affect a wrapped sender's lifecycle.
func Wrap(sender Sender) *Publisher {
	return &Publisher{sender: sender}
}

// Conn returns the owned async connection, or nil when the Publisher wraps an
// external Sender.
func (p *Publisher) Conn() *connection.Reconnecting {
	return p.owned
}

// Publish sends a raw payload on channel, fire-and-forget. It does not block; it
// returns ErrQueueFull if the connection's outbound queue is saturated. The
// number of subscribers that received it is not reported — use PublishCtx for
// that.
func (p *Publisher) Publish(channel string, payload []byte) error {
	if !p.sender.Send(command.FormatCommand("PUBLISH", channel, string(payload))) {
		return ErrQueueFull
	}
	return nil
}

// PublishJSON marshals v to JSON and publishes it on channel, fire-and-forget.
func (p *Publisher) PublishJSON(channel string, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return p.Publish(channel, payload)
}

// PublishCtx publishes a raw payload on channel and waits for Redis's reply,
// returning the number of clients that received the message. The context bounds
// the call. A Redis error reply is returned as *RedisError.
func (p *Publisher) PublishCtx(ctx context.Context, channel string, payload []byte) (int64, error) {
	sc, err := p.syncConn()
	if err != nil {
		return 0, err
	}
	v, err := sc.do(ctx, "PUBLISH", channel, string(payload))
	if err != nil {
		return 0, err
	}
	switch t := v.(type) {
	case *resp.RESPInteger:
		return t.Value, nil
	case *resp.RESPError:
		return 0, &RedisError{Msg: t.Value}
	default:
		return 0, fmt.Errorf("publish: unexpected reply type %s", v.Type())
	}
}

// PublishJSONCtx marshals v to JSON, publishes it on channel, and returns the
// subscriber count (see PublishCtx).
func (p *Publisher) PublishJSONCtx(ctx context.Context, channel string, v any) (int64, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	return p.PublishCtx(ctx, channel, payload)
}

func (p *Publisher) syncConn() (*syncConn, error) {
	if p.addr == "" {
		return nil, ErrConfirmUnavailable
	}
	p.scMu.Lock()
	defer p.scMu.Unlock()
	if p.sc == nil {
		p.sc = newSyncConn(p.addr, p.dialTimeout)
	}
	return p.sc, nil
}

// Close shuts down the connections this Publisher owns; it is a no-op for a
// wrapped Sender's async connection.
func (p *Publisher) Close() {
	if p.owned != nil {
		p.owned.Close()
	}
	p.scMu.Lock()
	if p.sc != nil {
		p.sc.close()
	}
	p.scMu.Unlock()
}
