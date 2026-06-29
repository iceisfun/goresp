package resp_test

import (
	"bytes"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func TestRESPInteger(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		integer := resp.RESPInteger{Value: 42}
		if got := integer.String(); got != "42" {
			t.Errorf("RESPInteger.String() = %v, want %v", got, "42")
		}
	})

	t.Run("Type", func(t *testing.T) {
		integer := resp.RESPInteger{}
		if got := integer.Type(); got != "Integer" {
			t.Errorf("RESPInteger.Type() = %v, want %v", got, "Integer")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		integer := resp.RESPInteger{Value: 42}
		buf := &bytes.Buffer{}
		if err := integer.Encode(buf); err != nil {
			t.Errorf("RESPInteger.Encode() error = %v", err)
		}
		expected := []byte(":42\r\n")
		if got := buf.Bytes(); !bytes.Equal(got, expected) {
			t.Errorf("RESPInteger.Encode() = %v, want %v", got, expected)
		}
	})

	// Add more tests for Decode, Equal, etc.
}
