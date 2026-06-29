# GoRESP

`github.com/iceisfun/goresp`

A stream-agnostic RESP (Redis Serialization Protocol) parser plus a self-healing
Redis pub/sub connector. The decoder turns any byte stream — a socket, a file,
a test fixture — into a sequence of complete RESP values, resuming cleanly
whenever the stream starves at a packet boundary. The connector hands decoded
messages to a handler you implement and inject. The library does no logging and
imposes no message schema: you own the payload.

## Design

- **Stream-agnostic decoder.** `resp.Decode` consumes bytes via `Provide` and
  emits complete values via `Parse`. It does not care where the bytes come from,
  so the same code path drives a live socket or replays a captured stream from
  disk.
- **Resumable / rewind on starvation.** When the buffered data ends mid-value,
  `Parse` returns `(nil, nil)` and retains its state; the next `Provide` resumes
  exactly where it left off. A non-allocating look-ahead remembers how many bytes
  the in-flight value still needs, so a starved stream stops re-scanning the same
  partial value on every read.
- **You own the payload.** The connector delivers a minimal `Message`
  (`Kind`, `Pattern`, `Channel`, `Payload []byte`); interpreting the bytes is up
  to you. A JSON convenience is provided.
- **Dependency injection.** Provide a `Handler` for messages and an optional
  `EventHandler` for connect/disconnect/error. `EventHandler.OnConnect` is handed
  a `Subscriber` so you can (re)subscribe on every connect.
- **Self-healing.** Automatic reconnect with backoff, automatic resubscription of
  remembered channels, and an optional idle keepalive.

## Decoding a stream

```go
d := resp.NewDecode()
for {
    d.Provide(readSomeBytes()) // from a socket, file, anywhere
    for {
        v, err := d.Parse()
        if err != nil {
            // unrecoverable protocol error
        }
        if v == nil {
            break // need more data; loop back to Provide
        }
        handle(v) // a resp.RESPValue
    }
}
```

Replaying a captured stream from disk uses the identical loop — just `Provide`
the file's bytes (in whatever chunk sizes you like).

## Consuming pub/sub

```go
type myHandler struct{}

func (myHandler) OnMessage(m connection.Message) {
    var v map[string]any
    if err := m.JSON(&v); err != nil {
        return
    }
    // ... use v, m.Channel, m.Pattern
}

// Subscribe on every connect so subscriptions survive reconnects.
type myEvents struct{ connection.NoopEvents }

func (myEvents) OnConnect(s connection.Subscriber) { s.PSubscribe("events.*") }

conn := connection.New("127.0.0.1:6379", myHandler{},
    connection.WithEvents(myEvents{}),
)
defer conn.Close()
```

`Handler.OnMessage` is called synchronously from the read loop: a slow handler
applies backpressure to the stream rather than silently dropping messages.
Offload to your own queue if you need to decouple processing from the socket.

### Options

```go
connection.WithEvents(EventHandler)             // lifecycle + errors (default: noop)
connection.WithContext(ctx)                     // cancel ctx to shut down (like Close)
connection.WithKeepAlive(bool)                  // idle PING/PONG keepalive (default: on)
connection.WithKeepAliveInterval(time.Duration) // idle threshold/period (default: 5s)
connection.WithTimeouts(read, write, dial time.Duration)
connection.WithMaxReconnectDelay(time.Duration)
```

`WithContext` binds the connection to a context, so it shuts down on cancel
(`Close` still works too, whichever fires first):

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
conn := connection.New(addr, handler, connection.WithContext(ctx))
```

The keepalive only PINGs when **no** data has arrived within the interval —
active traffic suppresses it — and forces a reconnect if the stream goes silent
for several intervals (catching stalls that produce no socket error). With
keepalive disabled, liveness relies solely on the read timeout.

## Publishing

Publishing reuses the same connector — just a **separate instance** that never
subscribes. Redis requires this: a subscribed connection rejects `PUBLISH`, so
publish and subscribe always live on different connections. No extra dependency
needed. `publish.New` creates and owns that dedicated connection for you:

```go
pub := publish.New("127.0.0.1:6379") // owns a dedicated, auto-reconnecting publish connection
defer pub.Close()

pub.PublishJSON("events.created", map[string]any{"id": 42})
pub.Publish("raw.channel", []byte("bytes"))
```

It takes the same connection options as `connection.New`
(`publish.New(addr, connection.WithTimeouts(...))`). To drive publishing through
a connection you manage yourself, use `publish.Wrap(sender)`.

## Encoding & commands

```go
ss := resp.RESPSimpleString{Value: "hello"}
var buf bytes.Buffer
ss.Encode(&buf)

cmd := command.FormatCommand("SUBSCRIBE", "chan") // RESP-encoded []byte
```

## Testing

```
go test ./...
go test -race ./...
```

The suite covers per-type encode/decode, exhaustive fragmentation (byte-at-a-time
and split-at-every-offset), large-payload reassembly, decoder reset after a
partial frame, and an end-to-end reconnect test proving partial state is
discarded so the post-reconnect stream parses cleanly.

### Manual fault injection

Use `socat` to sit between client and Redis and manipulate the stream:

```bash
socat -v -x TCP4-LISTEN:6379,fork,reuseaddr TCP4:127.0.0.1:6380
kill -SIGSTOP <socat child pid>  # pause data flow (exercise keepalive stall detection)
kill -SIGCONT <socat child pid>  # resume
kill -SIGKILL <socat child pid>  # force a reconnect
```

## Examples

Runnable, beginner-to-advanced examples live in [`examples/`](./examples), from
a minimal subscriber to a high-throughput worker pool.

## License

MIT — see [LICENSE](./LICENSE).
