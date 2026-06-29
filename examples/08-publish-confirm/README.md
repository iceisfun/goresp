# 08 · publish-confirm

Publish and get Redis's reply back — the number of subscribers that received the
message — with a per-call timeout.

```bash
# with no subscriber, expect "no subscribers":
go run ./examples/08-publish-confirm -channel events -every 1s

# now subscribe elsewhere and watch the count rise:
redis-cli subscribe events
```

Key points:

- `PublishCtx` / `PublishJSONCtx` do a **synchronous request/reply** and return
  `(int64, error)`: the subscriber count, or a Redis error reply as
  `*publish.RedisError`. The context bounds the wait.
- `received == 0` means the publish succeeded but nobody was listening — often
  worth knowing.
- These use a dedicated connection separate from fire-and-forget `Publish`
  (Redis request/reply must not share the async path). They are unavailable on a
  `publish.Wrap` Publisher (see `ErrConfirmUnavailable`); use `publish.New`.
- For high-rate publishing where you don't care who received it, prefer the
  non-blocking `Publish` / `PublishJSON` (see [04](../04-publish)).
