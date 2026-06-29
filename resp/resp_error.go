package resp

import (
	"bytes"
)

type RESPError struct {
	Value string
}

func (e *RESPError) Type() string {
	return "Error"
}

func (e *RESPError) String() string {
	return e.Value
}

func (s *RESPError) Equal(other RESPValue) bool {
	otherError, ok := other.(*RESPError)
	if !ok {
		return false
	}
	return s.Value == otherError.Value
}

func (e *RESPError) Encode(buf *bytes.Buffer) error {
	buf.WriteByte(byte(ERROR))
	buf.WriteString(e.Value)
	buf.Write(PROTOCOL_SEPARATOR)
	return nil
}

func (e *RESPError) Decode(buf *bytes.Buffer, start int) (int, error) {
	if buf.Len() <= start {
		return 0, errIncompleteData
	}

	if buf.Bytes()[start] != byte(ERROR) {
		return 0, errUnrecoverableProtocol
	}

	end := bytes.Index(buf.Bytes()[start:], PROTOCOL_SEPARATOR)
	if end == -1 {
		return 0, errIncompleteData
	}

	e.Value = string(buf.Bytes()[start+1 : start+end])
	return end + len(PROTOCOL_SEPARATOR), nil
}
