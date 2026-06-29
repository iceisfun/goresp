package resp

import (
	"bytes"
	"fmt"
	"strconv"
)

type RESPInteger struct {
	Value int64
}

func (i *RESPInteger) Type() string {
	return "Integer"
}

func (i *RESPInteger) String() string {
	return fmt.Sprintf("%d", i.Value)
}

func (s *RESPInteger) Equal(other RESPValue) bool {
	otherInteger, ok := other.(*RESPInteger)
	if !ok {
		return false
	}
	return s.Value == otherInteger.Value
}

func (i *RESPInteger) Encode(buf *bytes.Buffer) error {
	buf.WriteByte(byte(INTEGER))
	buf.WriteString(strconv.FormatInt(i.Value, 10))
	buf.Write(PROTOCOL_SEPARATOR)
	return nil
}

func (i *RESPInteger) Decode(buf *bytes.Buffer, start int) (int, error) {
	if buf.Len() <= start {
		return 0, errIncompleteData
	}

	if buf.Bytes()[start] != byte(INTEGER) {
		return 0, errUnrecoverableProtocol
	}

	end := bytes.Index(buf.Bytes()[start:], PROTOCOL_SEPARATOR)
	if end == -1 {
		return 0, errIncompleteData
	}

	value, err := strconv.ParseInt(string(buf.Bytes()[start+1:start+end]), 10, 64)
	if err != nil {
		return 0, err
	}

	i.Value = value
	return end + len(PROTOCOL_SEPARATOR), nil
}
