# 02 · json

Decode JSON payloads into a typed struct with `Message.JSON`.

```bash
go run ./examples/02-json -channel events
# in another terminal:
redis-cli publish events '{"id":7,"name":"widget"}'
```

Key points:

- `Message.JSON(&v)` is a thin convenience over `json.Unmarshal(m.Payload, &v)`.
  Use it for JSON payloads; for anything else, read `m.Payload` directly.
- Decoding errors are the handler's concern — bad input does not tear down the
  connection.
