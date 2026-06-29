package resp_test

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

// completeInputs returns encoded inputs paired with their expected decoded
// value, drawn from the shared TestCases table (only the complete ones).
func completeInputs() []TestCase {
	var out []TestCase
	for _, tc := range TestCases {
		if tc.Expected != nil && !tc.WantsErr {
			out = append(out, tc)
		}
	}
	return out
}

// drainAll provides nothing new and pulls every value currently decodable.
func drainAll(t *testing.T, d *resp.Decode) []resp.RESPValue {
	t.Helper()
	var got []resp.RESPValue
	for {
		v, err := d.Parse()
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if v == nil {
			return got
		}
		got = append(got, v)
	}
}

// TestByteAtATime feeds each input one byte at a time. Parse must return
// (nil,nil) on every starved step and yield exactly the expected value once the
// final byte lands — proving resume/rewind across the worst-case fragmentation.
func TestByteAtATime(t *testing.T) {
	for _, tc := range completeInputs() {
		t.Run(tc.Name, func(t *testing.T) {
			d := resp.NewDecode()
			for i := 0; i < len(tc.Input); i++ {
				d.Provide(tc.Input[i : i+1])
				v, err := d.Parse()
				if err != nil {
					t.Fatalf("byte %d: unexpected error: %v", i, err)
				}
				if i < len(tc.Input)-1 && v != nil {
					t.Fatalf("byte %d: got value before stream complete: %v", i, v)
				}
				if i == len(tc.Input)-1 {
					if !reflect.DeepEqual(v, tc.Expected) {
						t.Fatalf("final value = %v, want %v", v, tc.Expected)
					}
				}
			}
		})
	}
}

// TestEverySplit feeds each input as two chunks split at every offset, ensuring
// the decoder resumes correctly regardless of where the packet boundary falls.
func TestEverySplit(t *testing.T) {
	for _, tc := range completeInputs() {
		t.Run(tc.Name, func(t *testing.T) {
			for split := 0; split <= len(tc.Input); split++ {
				d := resp.NewDecode()
				d.Provide(tc.Input[:split])
				got := drainAll(t, d)
				if len(got) != 0 && split != len(tc.Input) {
					t.Fatalf("split %d: decoded a value from partial input", split)
				}
				d.Provide(tc.Input[split:])
				got = append(got, drainAll(t, d)...)
				if len(got) != 1 || !reflect.DeepEqual(got[0], tc.Expected) {
					t.Fatalf("split %d: got %v, want [%v]", split, got, tc.Expected)
				}
			}
		})
	}
}

// TestResetAfterPartial models a disconnect mid-value: a partial frame is
// buffered, Reset is called (as the connection does on disconnect/reconnect),
// and a fresh frame must then decode cleanly with no contamination.
func TestResetAfterPartial(t *testing.T) {
	d := resp.NewDecode()

	// Declares a 100-byte payload but only delivers part of it.
	d.Provide([]byte("$100\r\nonly-a-little"))
	if v, err := d.Parse(); v != nil || err != nil {
		t.Fatalf("partial frame: got (%v,%v), want (nil,nil)", v, err)
	}

	d.Reset()

	d.Provide([]byte("+OK\r\n"))
	v, err := d.Parse()
	if err != nil {
		t.Fatalf("after reset: %v", err)
	}
	want := &resp.RESPSimpleString{Value: "OK"}
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("after reset: got %v, want %v", v, want)
	}
}

// TestLargePayloadFragmented exercises the 5-25k msg/s JSON case: a big bulk
// string inside a pub/sub array delivered in many small chunks. It verifies the
// payload reassembles byte-for-byte and that starvation never produces a value
// early.
func TestLargePayloadFragmented(t *testing.T) {
	payload := bytes.Repeat([]byte(`{"k":"v","n":1234567890},`), 5000) // ~120 KB
	frame := encodeMessageFrame("events", payload)

	d := resp.NewDecode()
	const chunk = 1500 // mimic MTU-sized reads
	var got resp.RESPValue
	for off := 0; off < len(frame); off += chunk {
		end := min(off+chunk, len(frame))
		d.Provide(frame[off:end])
		v, err := d.Parse()
		if err != nil {
			t.Fatalf("offset %d: %v", off, err)
		}
		if v != nil && end != len(frame) {
			t.Fatalf("offset %d: value produced before frame complete", off)
		}
		if v != nil {
			got = v
		}
	}

	arr, ok := got.(*resp.RESPArray)
	if !ok || len(arr.Items) != 3 {
		t.Fatalf("decoded value is not a 3-item array: %v", got)
	}
	body, ok := arr.Items[2].(*resp.RESPBulkString)
	if !ok || !bytes.Equal(body.Value, payload) {
		t.Fatal("payload did not round-trip intact")
	}
}

// encodeMessageFrame builds a RESP "message" pub/sub frame by hand.
func encodeMessageFrame(channel string, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString("*3\r\n")
	b.WriteString("$7\r\nmessage\r\n")
	b.WriteString("$" + strconv.Itoa(len(channel)) + "\r\n" + channel + "\r\n")
	b.WriteString("$" + strconv.Itoa(len(payload)) + "\r\n")
	b.Write(payload)
	b.WriteString("\r\n")
	return b.Bytes()
}
