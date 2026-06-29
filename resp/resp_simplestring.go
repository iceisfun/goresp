package resp

import (
	"bytes"
)

type RESPSimpleString struct {
	Value string
}

func (s *RESPSimpleString) Type() string {
	return "SimpleString"
}

func (s *RESPSimpleString) String() string {
	return s.Value
}

func (s *RESPSimpleString) Equal(other RESPValue) bool {
	otherString, ok := other.(*RESPSimpleString)
	if !ok {
		return false
	}
	return s.Value == otherString.Value
}

func (ss *RESPSimpleString) Encode(buf *bytes.Buffer) error {
	buf.WriteByte(byte(SIMPLE_STRING))
	buf.WriteString(ss.Value)
	buf.Write(PROTOCOL_SEPARATOR)
	return nil
}

func (ss *RESPSimpleString) Decode(buf *bytes.Buffer, start int) (int, error) {
	if buf.Len() <= start {
		return 0, errIncompleteData
	}

	if buf.Bytes()[start] != byte(SIMPLE_STRING) {
		return 0, errUnrecoverableProtocol
	}

	end := bytes.Index(buf.Bytes()[start:], PROTOCOL_SEPARATOR)
	if end == -1 {
		return 0, errIncompleteData
	}

	ss.Value = string(buf.Bytes()[start+1 : start+end])
	return end + len(PROTOCOL_SEPARATOR), nil
}
