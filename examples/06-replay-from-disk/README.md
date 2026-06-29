# 06 · replay-from-disk

Drive the decoder directly from a file instead of a socket — the same
`resp.Decode` powers both. No Redis required.

```bash
# replay the built-in sample, one 7-byte read at a time:
go run ./examples/06-replay-from-disk

# stress the resume path with single-byte reads:
go run ./examples/06-replay-from-disk -chunk 1

# replay a real capture:
go run ./examples/06-replay-from-disk -file capture.bin
```

Key points:

- The decoder is **stream-agnostic**: `Provide` bytes from anywhere, `Parse`
  complete values. This is what makes captured streams replayable in tests.
- `Parse` returning `(nil, nil)` means "need more bytes" — the loop reads the
  next chunk and resumes exactly where it left off, regardless of where the chunk
  boundary fell mid-value.
- `connection.ParseMessage` turns a decoded value into a pub/sub `Message`, so
  the same demux logic the live connection uses works on replayed data.
- Tiny `-chunk` sizes deliberately split frames at the worst possible offsets to
  demonstrate the resumable/rewind behavior.
