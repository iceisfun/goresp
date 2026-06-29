package connection_test

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/iceisfun/goresp/connection"
)

// binaryBlob is a deterministic payload spanning all byte values with embedded
// CRLFs and RESP markers — the kind of content a "bus full of jpegs" carries.
func binaryBlob(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte((i*131 + 7) % 256)
	}
	copy(b, []byte{0xFF, 0xD8, 0xFF, 0xE0})
	if n > 24 {
		copy(b[8:], []byte("\r\n\x00$5\r\nhello\r\n\xff"))
	}
	return b
}

// messageFrame hand-builds a RESP "message" pub/sub frame with a raw payload.
func messageFrame(channel string, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString("*3\r\n$7\r\nmessage\r\n")
	fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(channel), channel)
	fmt.Fprintf(&b, "$%d\r\n", len(payload))
	b.Write(payload)
	b.WriteString("\r\n")
	return b.Bytes()
}

// TestBinaryPayloadIntegrity sends a large binary message over the real
// connection path (read loop -> decoder -> ParseMessage -> handler) and verifies
// the payload arrives byte-for-byte intact.
func TestBinaryPayloadIntegrity(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	payload := binaryBlob(256 * 1024)
	hold := make(chan struct{})
	defer close(hold)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go drainReads(conn)
			conn.Write(messageFrame("camera.binary", payload))
			<-hold
		}
	}()

	c := newCollector()
	conn := connection.New(ln.Addr().String(), c, connection.WithKeepAlive(false))
	defer conn.Close()

	select {
	case msg := <-c.msgs:
		if msg.Channel != "camera.binary" {
			t.Fatalf("channel = %q", msg.Channel)
		}
		if !bytes.Equal(msg.Payload, payload) {
			t.Fatalf("payload mismatch: got %d bytes, want %d", len(msg.Payload), len(payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for binary message")
	}
}
