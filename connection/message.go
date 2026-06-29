package connection

import "encoding/json"

// Message is a decoded Redis pub/sub delivery. The payload is handed to the
// user verbatim; interpreting it (as JSON, protobuf, plain text, ...) is the
// user's responsibility. Use JSON for the common case.
type Message struct {
	// Kind is the pub/sub frame type: "message" or "pmessage".
	Kind string
	// Pattern is the originating glob pattern, set only for "pmessage".
	Pattern string
	// Channel is the channel the payload was published on.
	Channel string
	// Payload is the published bytes. It is owned by the Message and safe to
	// retain after the handler returns.
	Payload []byte
}

// JSON unmarshals the payload into v. It is a convenience for the common case
// of JSON-encoded payloads and does not alter how the payload is stored.
func (m Message) JSON(v any) error {
	return json.Unmarshal(m.Payload, v)
}

// Handler consumes decoded pub/sub messages. It is called synchronously from
// the connection's read loop: a slow handler applies natural backpressure to
// the stream rather than dropping messages. Offload to your own queue if you
// need to decouple processing from the socket.
type Handler interface {
	OnMessage(Message)
}

// HandlerFunc adapts a plain function to the Handler interface.
type HandlerFunc func(Message)

func (f HandlerFunc) OnMessage(m Message) { f(m) }

// Subscriber is the subscription surface handed to EventHandler.OnConnect so a
// user can (re)establish subscriptions every time the link comes up. It is
// satisfied by *Reconnecting.
type Subscriber interface {
	Subscribe(channels ...string)
	PSubscribe(patterns ...string)
	Unsubscribe(channels ...string)
	PUnsubscribe(patterns ...string)
}

// EventHandler is notified of connection lifecycle transitions and errors. All
// methods are optional in spirit; pass a NoopEvents (or embed it) to ignore the
// ones you don't care about. Methods are called synchronously and should not
// block for long.
type EventHandler interface {
	// OnConnect fires after each successful (re)connect, after remembered
	// subscriptions have been restored. Use the supplied Subscriber to add or
	// refresh subscriptions; calls are idempotent across reconnects.
	OnConnect(Subscriber)
	// OnDisconnect fires when an established connection is torn down. err is the
	// triggering error, or nil for a clean Close.
	OnDisconnect(err error)
	// OnError fires for non-fatal and fatal errors (read/write failures, parse
	// errors, dropped commands). A fatal error is typically followed by
	// OnDisconnect.
	OnError(err error)
}

// NoopEvents is an EventHandler that ignores every event. Embed it to
// implement only the callbacks you need.
type NoopEvents struct{}

func (NoopEvents) OnConnect(Subscriber) {}
func (NoopEvents) OnDisconnect(error)   {}
func (NoopEvents) OnError(error)        {}
