package resp_test

import (
	"bytes"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func TestRESPBulkString(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name     string
			bulkStr  resp.RESPBulkString
			expected string
		}{
			{"Nil value", resp.RESPBulkString{Value: nil}, "<nil>"},
			{"Empty string", resp.RESPBulkString{Value: []byte("")}, ""},
			{"Non-empty string", resp.RESPBulkString{Value: []byte("Hello")}, "Hello"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.bulkStr.String(); got != tt.expected {
					t.Errorf("RESPBulkString.String() = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	t.Run("Type", func(t *testing.T) {
		bs := resp.RESPBulkString{}
		if got := bs.Type(); got != "BulkString" {
			t.Errorf("RESPBulkString.Type() = %v, want %v", got, "BulkString")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		tests := []struct {
			name     string
			bulkStr  resp.RESPBulkString
			expected []byte
		}{
			{"Nil value", resp.RESPBulkString{Value: nil}, []byte("$-1\r\n")},
			{"Empty string", resp.RESPBulkString{Value: []byte("")}, []byte("$0\r\n\r\n")},
			{"Non-empty string", resp.RESPBulkString{Value: []byte("Hello")}, []byte("$5\r\nHello\r\n")},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := &bytes.Buffer{}
				if err := tt.bulkStr.Encode(buf); err != nil {
					t.Errorf("RESPBulkString.Encode() error = %v", err)
				}
				if got := buf.Bytes(); !bytes.Equal(got, tt.expected) {
					t.Errorf("RESPBulkString.Encode() = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	// Add more tests for Decode, Equal, etc.
}
