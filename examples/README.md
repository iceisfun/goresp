# Examples

Runnable examples, beginner to advanced. Each is a standalone `main` package.

| # | Example | Shows |
|---|---------|-------|
| 01 | [subscribe](./01-subscribe) | Minimal consumer with `HandlerFunc` |
| 02 | [json](./02-json) | Decode payloads with `Message.JSON` |
| 03 | [reconnect-events](./03-reconnect-events) | `EventHandler`; subscribe in `OnConnect`; reconnect-safe |
| 04 | [publish](./04-publish) | Producing with `publish.New` (a separate connection) |
| 05 | [graceful-shutdown](./05-graceful-shutdown) | `WithContext` + `signal.NotifyContext` |
| 06 | [replay-from-disk](./06-replay-from-disk) | Driving `resp.Decode` from a file; no Redis |
| 07 | [worker-pool](./07-worker-pool) | Decoupled, high-throughput processing |
| 08 | [publish-confirm](./08-publish-confirm) | Confirmed publish with subscriber count via `PublishCtx` |

Most need a Redis (or compatible) server:

```bash
docker run --rm -p 6379:6379 redis:7
```

Example 06 needs nothing — it replays a built-in capture.

```bash
go run ./examples/01-subscribe -channel news
```
