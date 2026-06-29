package resp_test

import (
	"bytes"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func TestRESPArray(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		array := resp.RESPArray{Items: []resp.RESPValue{
			&resp.RESPSimpleString{Value: "OK"},
			&resp.RESPInteger{Value: 42},
		}}
		expected := "[OK 42]"
		if got := array.String(); got != expected {
			t.Errorf("RESPArray.String() = %v, want %v", got, expected)
		}
	})

	t.Run("Type", func(t *testing.T) {
		array := resp.RESPArray{}
		if got := array.Type(); got != "Array" {
			t.Errorf("RESPArray.Type() = %v, want %v", got, "Array")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		array := resp.RESPArray{Items: []resp.RESPValue{
			&resp.RESPSimpleString{Value: "OK"},
			&resp.RESPInteger{Value: 42},
		}}
		buf := &bytes.Buffer{}
		if err := array.Encode(buf); err != nil {
			t.Errorf("RESPArray.Encode() error = %v", err)
		}
		expected := []byte("*2\r\n+OK\r\n:42\r\n")
		if got := buf.Bytes(); !bytes.Equal(got, expected) {
			t.Errorf("RESPArray.Encode() = %v, want %v", got, expected)
		}
	})
}
