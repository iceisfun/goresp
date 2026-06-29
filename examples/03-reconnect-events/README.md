# 03 · reconnect-events

Implement `EventHandler` for lifecycle visibility and subscribe inside
`OnConnect` so subscriptions survive every reconnect.

```bash
go run ./examples/03-reconnect-events -patterns 'events.*,alerts.*'
# publish something that matches:
redis-cli publish events.created '{"id":1}'
# then kill/restart redis (or the connection) and watch it resubscribe
```

Key points:

- A single type implements both `Handler` and `EventHandler`; pass it as the
  handler and via `WithEvents`.
- `OnConnect(connection.Subscriber)` is the idiomatic place to (re)subscribe.
  Calls are idempotent across reconnects.
- `OnDisconnect` receives the triggering error (`nil` on a clean `Close`);
  `OnError` receives read/write/parse failures. The library never logs on its
  own — these callbacks are your hook.
- Embed `connection.NoopEvents` if you only want to implement one or two of the
  three methods.
