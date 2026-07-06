package resp

import (
	"bytes"
	"fmt"
	"strconv"
)

// RESPArray is a RESP array (prefix '*'): an ordered, possibly nested sequence
// of RESP values such as "*3\r\n...". Pub/sub deliveries arrive as arrays (for
// example ["message", channel, payload]). A nil Items represents the RESP null
// array ("*-1"), which is distinct from an empty (non-nil, zero-length) array.
type RESPArray struct {
	Items []RESPValue
}

func (a *RESPArray) Type() string {
	return "Array"
}

func (a *RESPArray) String() string {
	return fmt.Sprintf("%v", a.Items)
}

func (s *RESPArray) Equal(other RESPValue) bool {
	otherString, ok := other.(*RESPArray)
	if !ok {
		return false
	}
	if len(s.Items) != len(otherString.Items) {
		return false
	}

	for i, item := range s.Items {
		if !item.Equal(otherString.Items[i]) {
			return false
		}
	}

	return true
}

func (a *RESPArray) Encode(buf *bytes.Buffer) error {
	buf.WriteByte(byte(ARRAY))
	if a.Items == nil {
		buf.WriteString("-1")
		buf.Write(PROTOCOL_SEPARATOR)
		return nil
	}
	buf.WriteString(strconv.Itoa(len(a.Items)))
	buf.Write(PROTOCOL_SEPARATOR)
	for _, item := range a.Items {
		if err := item.Encode(buf); err != nil {
			return err
		}
	}
	return nil
}

func (a *RESPArray) Decode(buf *bytes.Buffer, start int) (int, error) {
	if buf.Len() <= start {
		return 0, errIncompleteData
	}

	if buf.Bytes()[start] != byte(ARRAY) {
		return 0, errUnrecoverableProtocol
	}

	end := bytes.Index(buf.Bytes()[start:], PROTOCOL_SEPARATOR)
	if end == -1 {
		return 0, errIncompleteData
	}

	count, err := strconv.Atoi(string(buf.Bytes()[start+1 : start+end]))
	if err != nil {
		return 0, err
	}

	consumed := end + len(PROTOCOL_SEPARATOR)

	if count < 0 {
		if count == -1 {
			// Null array.
			a.Items = nil
			return consumed, nil
		}
		// Any other negative count is a protocol violation; guard against the
		// make([]RESPValue, 0, count) panic on corrupt or hostile input.
		return 0, errUnrecoverableProtocol
	}

	a.Items = make([]RESPValue, 0, count)

	for i := 0; i < count; i++ {
		value, n, err := DecodeValue(buf, start+consumed)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, errIncompleteData
		}
		a.Items = append(a.Items, value)
		consumed += n
	}

	return consumed, nil
}
