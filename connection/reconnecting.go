package connection

import (
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/iceisfun/goresp/command"
	"github.com/iceisfun/goresp/resp"
)

// Defaults applied unless overridden by an Option.
const (
	defaultKeepAliveInterval = 5 * time.Second
	defaultMaxReconnectDelay = 30 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultDialTimeout       = 10 * time.Second
	commandQueueSize         = 4096
	readChunkSize            = 32768
)

type reconnectingChannel struct {
	Channel string
	Kind    string // "SUBSCRIBE" or "PSUBSCRIBE"
}

// Reconnecting is a self-healing Redis pub/sub client. It dials addr, optionally
// keeps the connection alive with idle PINGs, automatically reconnects with
// backoff, and resubscribes to every remembered channel/pattern after a
// reconnect. Decoded messages are delivered to the injected Handler; lifecycle
// and error events go to the injected EventHandler. It performs no logging of
// its own.
type Reconnecting struct {
	addr    string
	handler Handler
	events  EventHandler

	keepAlive         bool
	keepAliveInterval time.Duration
	maxReconnectDelay time.Duration
	dialTimeout       time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration

	decoder *resp.Decode

	mutex     sync.Mutex
	conn      net.Conn
	connected bool
	lastData  time.Time

	done           chan struct{}
	commands       chan []byte
	reconnectDelay time.Duration
	channels       sync.Map // string -> reconnectingChannel
}

// Option configures a Reconnecting connection.
type Option func(*Reconnecting)

// WithEvents sets the lifecycle/error handler. Defaults to NoopEvents.
func WithEvents(e EventHandler) Option {
	return func(r *Reconnecting) {
		if e != nil {
			r.events = e
		}
	}
}

// WithKeepAlive enables or disables the background idle PING/PONG keepalive.
// When enabled (the default), the connection sends a PING only when no data has
// arrived within the keepalive interval — active traffic suppresses it — and
// forces a reconnect if the stream goes silent for several intervals. When
// disabled, liveness relies solely on the read timeout.
func WithKeepAlive(enabled bool) Option {
	return func(r *Reconnecting) { r.keepAlive = enabled }
}

// WithKeepAliveInterval sets the idle threshold/period for the keepalive.
func WithKeepAliveInterval(d time.Duration) Option {
	return func(r *Reconnecting) {
		if d > 0 {
			r.keepAliveInterval = d
		}
	}
}

// WithTimeouts overrides the read, write, and dial timeouts. A zero value for
// any of them leaves that timeout at its default.
func WithTimeouts(read, write, dial time.Duration) Option {
	return func(r *Reconnecting) {
		if read > 0 {
			r.readTimeout = read
		}
		if write > 0 {
			r.writeTimeout = write
		}
		if dial > 0 {
			r.dialTimeout = dial
		}
	}
}

// WithMaxReconnectDelay caps the exponential reconnect backoff.
func WithMaxReconnectDelay(d time.Duration) Option {
	return func(r *Reconnecting) {
		if d > 0 {
			r.maxReconnectDelay = d
		}
	}
}

// New starts a Reconnecting connection to addr, delivering messages to handler.
// It begins dialing immediately in the background and returns right away. Call
// Close to stop it.
func New(addr string, handler Handler, opts ...Option) *Reconnecting {
	if addr == "" {
		panic("connection: addr is required")
	}
	if handler == nil {
		panic("connection: handler is required")
	}

	r := &Reconnecting{
		addr:              addr,
		handler:           handler,
		events:            NoopEvents{},
		keepAlive:         true,
		keepAliveInterval: defaultKeepAliveInterval,
		maxReconnectDelay: defaultMaxReconnectDelay,
		dialTimeout:       defaultDialTimeout,
		readTimeout:       defaultReadTimeout,
		writeTimeout:      defaultWriteTimeout,
		decoder:           resp.NewDecode(),
		done:              make(chan struct{}),
		commands:          make(chan []byte, commandQueueSize),
		reconnectDelay:    time.Second,
	}

	for _, opt := range opts {
		opt(r)
	}

	go r.handleReconnect()
	go r.handleSend()
	if r.keepAlive {
		go r.handleHealthCheck()
	}

	return r
}

// Close stops the connection and all background goroutines.
func (r *Reconnecting) Close() {
	select {
	case <-r.done:
		// already closed
	default:
		close(r.done)
	}
	r.disconnect(nil)
}

// Subscribe subscribes to the given channels and remembers them so they are
// restored after a reconnect.
func (r *Reconnecting) Subscribe(channels ...string) {
	for _, channel := range channels {
		if _, loaded := r.channels.LoadOrStore(channel, reconnectingChannel{Channel: channel, Kind: "SUBSCRIBE"}); !loaded {
			r.Send(command.FormatCommand("SUBSCRIBE", channel))
		}
	}
}

// PSubscribe subscribes to the given glob patterns and remembers them so they
// are restored after a reconnect.
func (r *Reconnecting) PSubscribe(patterns ...string) {
	for _, pattern := range patterns {
		if _, loaded := r.channels.LoadOrStore(pattern, reconnectingChannel{Channel: pattern, Kind: "PSUBSCRIBE"}); !loaded {
			r.Send(command.FormatCommand("PSUBSCRIBE", pattern))
		}
	}
}

// Unsubscribe removes channel subscriptions.
func (r *Reconnecting) Unsubscribe(channels ...string) {
	for _, channel := range channels {
		if _, loaded := r.channels.LoadAndDelete(channel); loaded {
			r.Send(command.FormatCommand("UNSUBSCRIBE", channel))
		}
	}
}

// PUnsubscribe removes pattern subscriptions.
func (r *Reconnecting) PUnsubscribe(patterns ...string) {
	for _, pattern := range patterns {
		if _, loaded := r.channels.LoadAndDelete(pattern); loaded {
			r.Send(command.FormatCommand("PUNSUBSCRIBE", pattern))
		}
	}
}

// Send queues a raw, RESP-encoded command for the send goroutine. It returns
// false (and reports ErrCommandQueueFull via Events.OnError) if the queue is
// full.
func (r *Reconnecting) Send(cmd []byte) bool {
	select {
	case r.commands <- cmd:
		return true
	default:
		r.events.OnError(ErrCommandQueueFull)
		return false
	}
}

func (r *Reconnecting) onConnect() {
	// Reset the inactivity timer up front.
	r.markData()

	// Restore all remembered subscriptions.
	r.channels.Range(func(_, value any) bool {
		item := value.(reconnectingChannel)
		switch item.Kind {
		case "SUBSCRIBE":
			r.Send(command.FormatCommand("SUBSCRIBE", item.Channel))
		case "PSUBSCRIBE":
			r.Send(command.FormatCommand("PSUBSCRIBE", item.Channel))
		}
		return true
	})

	// Let the user (re)subscribe through the supplied Subscriber.
	r.events.OnConnect(r)
}

func (r *Reconnecting) handleReconnect() {
	for {
		select {
		case <-r.done:
			return
		default:
			if err := r.connect(); err != nil {
				r.events.OnError(err)
				if !r.sleep(r.reconnectDelay) {
					return
				}
				r.reconnectDelay = min(r.reconnectDelay*2, r.maxReconnectDelay)
				continue
			}
			r.reconnectDelay = time.Second
			r.readLoop()
		}
	}
}

// sleep waits for d or until Close, returning false if Close fired.
func (r *Reconnecting) sleep(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-r.done:
		return false
	case <-timer.C:
		return true
	}
}

func (r *Reconnecting) connect() error {
	conn, err := net.DialTimeout("tcp", r.addr, r.dialTimeout)
	if err != nil {
		return err
	}

	r.mutex.Lock()
	r.conn = conn
	r.connected = true
	r.lastData = time.Now()
	// Start every connection with a clean decoder so any partial value left
	// over from a previous, abruptly-dropped connection cannot corrupt parsing
	// of the new stream.
	r.decoder.Reset()
	r.mutex.Unlock()

	r.onConnect()
	return nil
}

func (r *Reconnecting) readLoop() {
	defer r.disconnect(nil)

	buf := make([]byte, readChunkSize)
	for {
		conn := r.currentConn()
		if conn == nil {
			return
		}

		conn.SetReadDeadline(time.Now().Add(r.readTimeout))
		n, err := conn.Read(buf)
		if err != nil {
			r.events.OnError(err)
			r.disconnect(err)
			return
		}

		r.markData()
		r.decoder.Provide(buf[:n])
		if err := r.drain(); err != nil {
			r.events.OnError(err)
			r.disconnect(err)
			return
		}
	}
}

// drain pulls every complete value currently buffered and dispatches pub/sub
// messages to the handler. It returns on a parse error (unrecoverable) or once
// the buffer is exhausted.
func (r *Reconnecting) drain() error {
	for {
		value, err := r.decoder.Parse()
		if err != nil {
			return err
		}
		if value == nil {
			return nil // need more data
		}

		msg, ok := ParseMessage(value)
		if !ok {
			continue
		}
		if msg.Kind == "pong" {
			r.markData()
			continue
		}
		r.handler.OnMessage(msg)
	}
}

func (r *Reconnecting) handleHealthCheck() {
	ticker := time.NewTicker(r.keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			r.healthCheck()
		}
	}
}

func (r *Reconnecting) healthCheck() {
	r.mutex.Lock()
	if !r.connected || r.conn == nil {
		r.mutex.Unlock()
		return
	}
	idle := time.Since(r.lastData)
	r.mutex.Unlock()

	switch {
	case idle > 4*r.keepAliveInterval:
		// The stream went silent without a syscall error; force a reconnect.
		r.disconnect(ErrStreamStalled)
	case idle >= r.keepAliveInterval:
		// Idle but not yet stale: probe with a unique-nonce PING so we can
		// recognize our own PONG. Active traffic keeps idle small and skips this.
		r.Send(command.FormatCommand("PING", strconv.Itoa(rand.Int())))
	}
}

func (r *Reconnecting) handleSend() {
	for {
		select {
		case <-r.done:
			return
		case cmd := <-r.commands:
			conn := r.currentConn()
			if conn == nil {
				continue
			}
			conn.SetWriteDeadline(time.Now().Add(r.writeTimeout))
			if _, err := conn.Write(cmd); err != nil {
				r.events.OnError(err)
				r.disconnect(err)
			}
		}
	}
}

func (r *Reconnecting) disconnect(cause error) {
	r.mutex.Lock()
	if !r.connected || r.conn == nil {
		r.mutex.Unlock()
		return
	}
	r.connected = false
	conn := r.conn
	r.conn = nil
	// Drop any partial value: the next connection starts from a clean slate.
	r.decoder.Reset()
	r.mutex.Unlock()

	conn.Close()
	r.events.OnDisconnect(cause)
}

func (r *Reconnecting) currentConn() net.Conn {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if !r.connected {
		return nil
	}
	return r.conn
}

func (r *Reconnecting) markData() {
	r.mutex.Lock()
	r.lastData = time.Now()
	r.mutex.Unlock()
}
