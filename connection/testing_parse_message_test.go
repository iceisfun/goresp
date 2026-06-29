package connection_test

import (
	"reflect"
	"testing"

	"github.com/iceisfun/goresp/connection"
	"github.com/iceisfun/goresp/resp"
)

func bulk(s string) *resp.RESPBulkString { return &resp.RESPBulkString{Value: []byte(s)} }

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   resp.RESPValue
		wantMsg connection.Message
		wantOK  bool
	}{
		{name: "Nil input", input: nil, wantOK: false},
		{name: "Simple String", input: &resp.RESPSimpleString{Value: "OK"}, wantOK: false},
		{name: "Error", input: &resp.RESPError{Value: "boom"}, wantOK: false},
		{name: "Integer", input: &resp.RESPInteger{Value: 42}, wantOK: false},
		{name: "Bulk String", input: bulk("test"), wantOK: false},
		{
			name: "message",
			input: &resp.RESPArray{Items: []resp.RESPValue{
				bulk("message"), bulk("channel1"), bulk("Hello, World!"),
			}},
			wantMsg: connection.Message{Kind: "message", Channel: "channel1", Payload: []byte("Hello, World!")},
			wantOK:  true,
		},
		{
			name: "pmessage",
			input: &resp.RESPArray{Items: []resp.RESPValue{
				bulk("pmessage"), bulk("pattern1"), bulk("channel1"), bulk("Hello, Pattern!"),
			}},
			wantMsg: connection.Message{Kind: "pmessage", Pattern: "pattern1", Channel: "channel1", Payload: []byte("Hello, Pattern!")},
			wantOK:  true,
		},
		{
			name: "pong",
			input: &resp.RESPArray{Items: []resp.RESPValue{
				bulk("pong"), bulk(""),
			}},
			wantMsg: connection.Message{Kind: "pong"},
			wantOK:  true,
		},
		{
			name: "subscribe confirmation ignored",
			input: &resp.RESPArray{Items: []resp.RESPValue{
				bulk("subscribe"), bulk("channel1"), &resp.RESPInteger{Value: 1},
			}},
			wantOK: false,
		},
		{
			name: "message wrong arity",
			input: &resp.RESPArray{Items: []resp.RESPValue{
				bulk("message"), bulk("channel1"),
			}},
			wantOK: false,
		},
		{
			name: "unknown kind",
			input: &resp.RESPArray{Items: []resp.RESPValue{
				bulk("invalid"), bulk("channel1"), bulk("data"),
			}},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, ok := connection.ParseMessage(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ParseMessage() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && !reflect.DeepEqual(msg, tt.wantMsg) {
				t.Errorf("ParseMessage() = %+v, want %+v", msg, tt.wantMsg)
			}
		})
	}
}

func TestMessageJSON(t *testing.T) {
	m := connection.Message{Payload: []byte(`{"a":1,"b":"two"}`)}
	var out struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	if err := m.JSON(&out); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if out.A != 1 || out.B != "two" {
		t.Errorf("got %+v", out)
	}
}
