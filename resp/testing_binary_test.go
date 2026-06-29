package resp_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

// fakeJPEG builds a deterministic binary blob that exercises every byte value
// and deliberately embeds RESP-looking sequences (CRLF, type markers, a nested
// "$5\r\n...") so a decoder that mistakenly scanned the body instead of honoring
// the length prefix would break. It is framed like a real JPEG (SOI ... EOI).
func fakeJPEG(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte((i*131 + 7) % 256) // cycles all 256 values, deterministic
	}
	copy(b[0:], []byte{0xFF, 0xD8, 0xFF, 0xE0}) // JPEG SOI + APP0
	if n > 32 {
		copy(b[8:], []byte("\r\n\r\n\x00message\r\n$5\r\nhello\r\n\x00\xff"))
	}
	if n >= 2 {
		b[n-2], b[n-1] = 0xFF, 0xD9 // JPEG EOI
	}
	return b
}

// TestBinaryBulkRoundTrip encodes a binary bulk string and decodes it back,
// proving both Encode and Decode are byte-for-byte binary safe (no CRLF
// scanning of the body).
func TestBinaryBulkRoundTrip(t *testing.T) {
	payload := fakeJPEG(50000)
	var buf bytes.Buffer
	if err := (&resp.RESPBulkString{Value: payload}).Encode(&buf); err != nil {
		t.Fatal(err)
	}

	d := resp.NewDecode()
	d.Provide(buf.Bytes())
	v, err := d.Parse()
	if err != nil {
		t.Fatal(err)
	}
	bs, ok := v.(*resp.RESPBulkString)
	if !ok || !bytes.Equal(bs.Value, payload) {
		t.Fatal("binary bulk string did not round-trip intact")
	}
}

// TestBinaryMessageByteByByte is the extreme stall case: a ~100 KB binary
// pub/sub frame is fed one byte at a time. Parse must return (nil,nil) on every
// starved step and yield exactly one intact value on the final byte.
func TestBinaryMessageByteByByte(t *testing.T) {
	payload := fakeJPEG(100000)
	frame := encodeMessageFrame("camera.7", payload)

	d := resp.NewDecode()
	var got resp.RESPValue
	for i := 0; i < len(frame); i++ {
		d.Provide(frame[i : i+1])
		v, err := d.Parse()
		if err != nil {
			t.Fatalf("byte %d: %v", i, err)
		}
		if v != nil && i != len(frame)-1 {
			t.Fatalf("byte %d: value produced before final byte", i)
		}
		if v != nil {
			got = v
		}
	}

	assertMessagePayload(t, got, payload)
}

// TestBinaryMessageRandomChunks pushes a multi-megabyte binary payload through
// the decoder in randomly sized reads, mimicking real socket delivery.
func TestBinaryMessageRandomChunks(t *testing.T) {
	payload := fakeJPEG(3 << 20) // 3 MiB
	frame := encodeMessageFrame("blob", payload)

	rng := rand.New(rand.NewSource(1))
	d := resp.NewDecode()
	var got resp.RESPValue
	for off := 0; off < len(frame); {
		n := 1 + rng.Intn(9000)
		end := min(off+n, len(frame))
		d.Provide(frame[off:end])
		off = end
		for {
			v, err := d.Parse()
			if err != nil {
				t.Fatalf("offset %d: %v", off, err)
			}
			if v == nil {
				break
			}
			if got != nil {
				t.Fatal("decoded more than one value")
			}
			got = v
		}
	}

	assertMessagePayload(t, got, payload)
}

// TestBinaryEmbeddedFraming proves a body that literally contains a valid-looking
// RESP frame is treated as opaque bytes, not re-parsed.
func TestBinaryEmbeddedFraming(t *testing.T) {
	payload := []byte("*3\r\n$7\r\nmessage\r\n$3\r\nfoo\r\n$5\r\nhello\r\n\x00\xff\r\n")
	frame := encodeMessageFrame("trick", payload)

	d := resp.NewDecode()
	d.Provide(frame)
	v, err := d.Parse()
	if err != nil {
		t.Fatal(err)
	}
	assertMessagePayload(t, v, payload)

	// And nothing else should remain to decode.
	if extra, _ := d.Parse(); extra != nil {
		t.Fatalf("unexpected trailing value: %v", extra)
	}
}

// TestNegativeLengthIsErrorNotPanic ensures corrupt length prefixes are rejected
// cleanly instead of panicking on a negative make().
func TestNegativeLengthIsErrorNotPanic(t *testing.T) {
	for _, in := range []string{"$-2\r\n", "$-100\r\n", "*-2\r\nx", "*-7\r\n"} {
		d := resp.NewDecode()
		d.Provide([]byte(in))
		v, err := d.Parse()
		if err == nil {
			t.Errorf("%q: expected protocol error, got value=%v err=nil", in, v)
		}
	}
}

func assertMessagePayload(t *testing.T, v resp.RESPValue, want []byte) {
	t.Helper()
	arr, ok := v.(*resp.RESPArray)
	if !ok || len(arr.Items) != 3 {
		t.Fatalf("not a 3-item array: %v", v)
	}
	body, ok := arr.Items[2].(*resp.RESPBulkString)
	if !ok {
		t.Fatalf("third item is not a bulk string: %v", arr.Items[2])
	}
	if !bytes.Equal(body.Value, want) {
		t.Fatalf("payload mismatch: got %d bytes, want %d bytes", len(body.Value), len(want))
	}
}
