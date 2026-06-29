# 01 · subscribe

The smallest possible consumer: subscribe to one channel and print each message.

```bash
go run ./examples/01-subscribe -redis 127.0.0.1:6379 -channel news
# in another terminal:
redis-cli publish news "hello"
```

Key points:

- `connection.HandlerFunc` adapts a plain `func(connection.Message)` to the
  `Handler` interface — no struct needed for trivial cases.
- `conn.Subscribe(channel)` records the channel, so it is **automatically
  re-subscribed after a reconnect**. (See [03](../03-reconnect-events) for the
  `OnConnect` alternative.)
- `m.Payload` is the raw published bytes; this example just prints them.
