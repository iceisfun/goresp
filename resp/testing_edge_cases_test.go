package resp_test

import (
	"reflect"
	"testing"

	"github.com/iceisfun/goresp/resp"
)

func TestDecodeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []struct {
			value    resp.RESPValue
			err      bool
			wantMore bool
		}
	}{
		{
			name:  "Empty input",
			input: []byte{},
			expected: []struct {
				value    resp.RESPValue
				err      bool
				wantMore bool
			}{
				{nil, false, true},
			},
		},
		{
			name:  "Simple string with extra opcode",
			input: []byte("+OK\r\n+"),
			expected: []struct {
				value    resp.RESPValue
				err      bool
				wantMore bool
			}{
				{&resp.RESPSimpleString{Value: "OK"}, false, false},
				{nil, false, true},
			},
		},
		{
			name:  "Simple string with incomplete bulk string",
			input: []byte("+OK\r\n$5\r\nhe"),
			expected: []struct {
				value    resp.RESPValue
				err      bool
				wantMore bool
			}{
				{&resp.RESPSimpleString{Value: "OK"}, false, false},
				{nil, false, true},
			},
		},
		{
			name:  "Simple string with non-opcode extra byte",
			input: []byte("+OK\r\n%"),
			expected: []struct {
				value    resp.RESPValue
				err      bool
				wantMore bool
			}{
				{&resp.RESPSimpleString{Value: "OK"}, false, false},
				{nil, true, false},
			},
		},
		{
			name:  "Bulk string with \\r\\n\\r\\n in content",
			input: []byte("$14\r\nHello\r\n\r\nWorld\r\n"),
			expected: []struct {
				value    resp.RESPValue
				err      bool
				wantMore bool
			}{
				{&resp.RESPBulkString{Value: []byte("Hello\r\n\r\nWorld")}, false, false},
				{nil, false, true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := resp.NewDecode()
			decoder.Provide(tt.input)

			for i, exp := range tt.expected {
				value, err := decoder.Parse()

				// Check value
				if !reflect.DeepEqual(value, exp.value) {
					t.Errorf("Parse() call %d value = %v, want %v", i+1, value, exp.value)
				}

				// Check error
				if exp.err && err == nil {
					t.Errorf("Parse() call %d error = nil, want error", i+1)
				} else if !exp.err && err != nil {
					t.Errorf("Parse() call %d error = %v, want nil", i+1, err)
				}

				// Check if more data is needed
				if exp.wantMore && (value != nil || err != nil) {
					t.Errorf("Parse() call %d = %v, %v; want nil, nil (more data needed)", i+1, value, err)
				} else if !exp.wantMore && value == nil && err == nil {
					t.Errorf("Parse() call %d = nil, nil; want non-nil value or error", i+1)
				}
			}
		})
	}
}
