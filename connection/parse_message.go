package connection

import (
	"strings"

	"github.com/iceisfun/goresp/resp"
)

// ParseMessage interprets a decoded RESP value as a pub/sub frame. It returns
// the message and true for "message"/"pmessage" deliveries and for "pong"
// replies (Kind == "pong", no payload). Everything else — subscribe/unsubscribe
// confirmations, out-of-band replies — returns ok=false and is ignored by the
// connection. The payload bytes are taken directly from the decoded bulk string
// (no extra copy).
func ParseMessage(value resp.RESPValue) (Message, bool) {
	array, ok := value.(*resp.RESPArray)
	if !ok || len(array.Items) < 2 {
		return Message{}, false
	}

	kind, ok := array.Items[0].(*resp.RESPBulkString)
	if !ok {
		return Message{}, false
	}
	kindStr := string(kind.Value)

	if strings.EqualFold(kindStr, "pong") {
		return Message{Kind: "pong"}, true
	}

	switch kindStr {
	case "message":
		if len(array.Items) != 3 {
			return Message{}, false
		}
		channel, ok1 := array.Items[1].(*resp.RESPBulkString)
		data, ok2 := array.Items[2].(*resp.RESPBulkString)
		if !ok1 || !ok2 {
			return Message{}, false
		}
		return Message{
			Kind:    "message",
			Channel: string(channel.Value),
			Payload: data.Value,
		}, true

	case "pmessage":
		if len(array.Items) != 4 {
			return Message{}, false
		}
		pattern, ok1 := array.Items[1].(*resp.RESPBulkString)
		channel, ok2 := array.Items[2].(*resp.RESPBulkString)
		data, ok3 := array.Items[3].(*resp.RESPBulkString)
		if !ok1 || !ok2 || !ok3 {
			return Message{}, false
		}
		return Message{
			Kind:    "pmessage",
			Pattern: string(pattern.Value),
			Channel: string(channel.Value),
			Payload: data.Value,
		}, true

	default:
		return Message{}, false
	}
}
