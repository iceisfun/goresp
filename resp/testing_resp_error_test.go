package resp_test

import (
	"bytes"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func TestRESPError(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		err := resp.RESPError{Value: "Test error"}
		if got := err.String(); got != "Test error" {
			t.Errorf("RESPError.String() = %v, want %v", got, "Test error")
		}
	})

	t.Run("Type", func(t *testing.T) {
		err := resp.RESPError{}
		if got := err.Type(); got != "Error" {
			t.Errorf("RESPError.Type() = %v, want %v", got, "Error")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		err := resp.RESPError{Value: "Test error"}
		buf := &bytes.Buffer{}
		if err := err.Encode(buf); err != nil {
			t.Errorf("RESPError.Encode() error = %v", err)
		}
		expected := []byte("-Test error\r\n")
		if got := buf.Bytes(); !bytes.Equal(got, expected) {
			t.Errorf("RESPError.Encode() = %v, want %v", got, expected)
		}
	})

	// Add more tests for Decode, Equal, etc.
}
