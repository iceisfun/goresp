// Advanced: replay a captured stream from disk (no Redis required).
//
// The decoder is stream-agnostic: it does not care whether bytes come from a
// socket or a file. This example feeds a capture through resp.Decode in tiny
// chunks to exercise the resumable parser — every chunk boundary lands the
// decoder mid-value, and it picks up cleanly on the next read.
//
// Run with no flags to replay a built-in sample, or point -file at a real
// capture (e.g. bytes recorded off a Redis pub/sub socket).
package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"os"

	"github.com/iceisfun/goresp/connection"
	"github.com/iceisfun/goresp/resp"
)

func main() {
	file := flag.String("file", "", "RESP capture file to replay (default: built-in sample)")
	chunk := flag.Int("chunk", 7, "bytes per simulated read; small values stress the resume path")
	flag.Parse()

	var src io.Reader
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		src = f
	} else {
		src = bytes.NewReader(sampleCapture())
	}

	dec := resp.NewDecode()
	buf := make([]byte, *chunk)
	var count int

	for {
		n, err := src.Read(buf)
		if n > 0 {
			dec.Provide(buf[:n])
			for {
				v, perr := dec.Parse()
				if perr != nil {
					log.Fatalf("parse error: %v", perr)
				}
				if v == nil {
					break // need more bytes — read the next chunk
				}
				if m, ok := connection.ParseMessage(v); ok && m.Kind != "pong" {
					count++
					log.Printf("%s: %s", m.Channel, m.Payload)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("replayed %d messages (chunk size %d)", count, *chunk)
}

// sampleCapture hand-encodes a few RESP "message" frames, exactly as a Redis
// server would write them on the wire.
func sampleCapture() []byte {
	var buf bytes.Buffer
	for _, f := range []struct{ channel, payload string }{
		{"events", `{"id":1,"name":"alpha"}`},
		{"events", `{"id":2,"name":"bravo"}`},
		{"alerts", `{"level":"warn","msg":"disk at 80%"}`},
	} {
		frame := &resp.RESPArray{Items: []resp.RESPValue{
			&resp.RESPBulkString{Value: []byte("message")},
			&resp.RESPBulkString{Value: []byte(f.channel)},
			&resp.RESPBulkString{Value: []byte(f.payload)},
		}}
		_ = frame.Encode(&buf)
	}
	return buf.Bytes()
}
