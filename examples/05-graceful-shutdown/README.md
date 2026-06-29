# 05 · graceful-shutdown

Bind the connection to a context and shut down cleanly on a signal.

```bash
go run ./examples/05-graceful-shutdown -channel events
# press Ctrl+C — the connection tears down via context cancellation
```

Key points:

- `signal.NotifyContext` cancels the context on `SIGINT`/`SIGTERM`.
- `connection.WithContext(ctx)` shuts the connection down when the context is
  cancelled — equivalent to calling `Close()`. Both are safe; whichever happens
  first wins.
- A dial in flight at cancel time is dropped rather than left to linger, so
  shutdown is prompt.
