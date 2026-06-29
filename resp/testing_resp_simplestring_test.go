package resp_test

import (
	"bytes"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func TestRESPSimpleString(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		ss := resp.RESPSimpleString{Value: "OK"}
		if got := ss.String(); got != "OK" {
			t.Errorf("RESPSimpleString.String() = %v, want %v", got, "OK")
		}
	})

	t.Run("Type", func(t *testing.T) {
		ss := resp.RESPSimpleString{}
		if got := ss.Type(); got != "SimpleString" {
			t.Errorf("RESPSimpleString.Type() = %v, want %v", got, "SimpleString")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		ss := resp.RESPSimpleString{Value: "OK"}
		buf := &bytes.Buffer{}
		if err := ss.Encode(buf); err != nil {
			t.Errorf("RESPSimpleString.Encode() error = %v", err)
		}
		expected := []byte("+OK\r\n")
		if got := buf.Bytes(); !bytes.Equal(got, expected) {
			t.Errorf("RESPSimpleString.Encode() = %v, want %v", got, expected)
		}
	})

	// Add more tests for Decode, Equal, etc.
}
