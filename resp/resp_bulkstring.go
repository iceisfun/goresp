package resp

import (
	"bytes"
	"strconv"
)

type RESPBulkString struct {
	Value []byte
}

func (b *RESPBulkString) Type() string {
	return "BulkString"
}

func (b *RESPBulkString) String() string {
	if b.Value == nil {
		return "<nil>"
	}
	return string(b.Value)
}

func (s *RESPBulkString) Equal(other RESPValue) bool {
	otherString, ok := other.(*RESPBulkString)
	if !ok {
		return false
	}

	// If both are nil, they're equal
	if s.Value == nil && otherString.Value == nil {
		return true
	}

	// If only one is nil, they're not equal
	if (s.Value == nil) != (otherString.Value == nil) {
		return false
	}

	// Both are non-nil, compare contents
	return bytes.Equal(s.Value, otherString.Value)
}

func (bs *RESPBulkString) Encode(buf *bytes.Buffer) error {
	buf.WriteByte(byte(BULK_STRING))
	if bs.Value == nil {
		buf.WriteString("-1")
		buf.Write(PROTOCOL_SEPARATOR)
	} else {
		buf.WriteString(strconv.Itoa(len(bs.Value)))
		buf.Write(PROTOCOL_SEPARATOR)
		buf.Write(bs.Value)
		buf.Write(PROTOCOL_SEPARATOR)
	}
	return nil
}

func (bs *RESPBulkString) Decode(buf *bytes.Buffer, start int) (int, error) {
	if buf.Len() <= start {
		return 0, errIncompleteData
	}

	if buf.Bytes()[start] != byte(BULK_STRING) {
		return 0, errUnrecoverableProtocol
	}

	end := bytes.Index(buf.Bytes()[start:], PROTOCOL_SEPARATOR)
	if end == -1 {
		return 0, errIncompleteData
	}

	length, err := strconv.Atoi(string(buf.Bytes()[start+1 : start+end]))
	if err != nil {
		return 0, err
	}

	consumed := end + len(PROTOCOL_SEPARATOR)

	if length == -1 {
		// Null bulk string
		bs.Value = nil
		return consumed, nil
	}

	if start+consumed+length+len(PROTOCOL_SEPARATOR) > buf.Len() {
		return 0, errIncompleteData
	}

	bs.Value = make([]byte, length)
	copy(bs.Value, buf.Bytes()[start+consumed:start+consumed+length])

	return consumed + length + len(PROTOCOL_SEPARATOR), nil
}
