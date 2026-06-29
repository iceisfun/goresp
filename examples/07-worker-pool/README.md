# 07 · worker-pool

Decouple message processing from the socket with a buffered queue and a pool of
workers — the pattern for sustained high throughput (5–25k msg/s).

```bash
go run ./examples/07-worker-pool -pattern '*' -workers 8 -queue 8192
# generate load from another terminal:
redis-cli -r 100000 publish bench '{"x":1}'
```

Key points:

- `Handler.OnMessage` runs **synchronously in the read loop**. Do not do heavy
  work there — hand off to a queue, as shown.
- The handoff (`p.jobs <- m`) is a **blocking send**: when workers fall behind,
  backpressure flows back through the socket via TCP flow control instead of
  dropping messages. Swap to a `select`/`default` if you would rather shed load.
- Tune `-workers` and `-queue` for your workload. The per-second log prints
  throughput and current queue depth so you can see where the bottleneck is.
- Subscriptions are (re)established in `OnConnect`, so the pool keeps working
  across reconnects.
