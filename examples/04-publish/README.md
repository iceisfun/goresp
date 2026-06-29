# 04 · publish

Publish messages over a dedicated, self-owned connection.

```bash
# subscribe in one terminal (use example 01 or redis-cli):
redis-cli subscribe events
# publish in another:
go run ./examples/04-publish -channel events -every 500ms
```

Key points:

- **Publishing needs its own connection.** Once a connection issues `SUBSCRIBE`,
  Redis rejects `PUBLISH` on it. This is a Redis rule, not a goresp limitation —
  no extra dependency required, just a second connector instance.
- `publish.New(addr, opts...)` creates and owns that connection (auto-reconnects
  like any other) and accepts the same `connection.With…` options. `Close` shuts
  it down.
- `PublishJSON` marshals and publishes; `Publish` takes raw bytes. Both are
  non-blocking and return `publish.ErrQueueFull` if the outbound queue is
  saturated.
- To drive publishing through a connection you manage yourself, use
  `publish.Wrap(sender)`.
