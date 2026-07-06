package resp

import (
	"bytes"
)

// RESPValue is a single decoded RESP value. The five wire types used by Redis
// pub/sub — [RESPSimpleString], [RESPError], [RESPInteger], [RESPBulkString],
// and [RESPArray] — all implement it, so a decoded stream is a sequence of
// RESPValues whose concrete type a caller recovers with a type switch.
type RESPValue interface {
	// Type returns a human-readable name for the value's wire type, e.g.
	// "BulkString". It is for diagnostics, not protocol decisions.
	Type() string
	// String renders the value for logging and debugging. It is not the wire
	// encoding — use Encode for that.
	String() string
	// Equal reports whether other is the same concrete type and holds an equal
	// value. Values of different types are never equal.
	Equal(RESPValue) bool
	// Decode parses the value from buf beginning at index start, returning the
	// number of bytes consumed. It returns errIncompleteData when buf does not
	// yet hold the whole value, leaving buf untouched so the caller can retry
	// after buffering more bytes.
	Decode(*bytes.Buffer, int) (int, error)
	// Encode appends the RESP wire form of the value to buf.
	Encode(buf *bytes.Buffer) error
}

func decodeValue(buf *bytes.Buffer, start int) (RESPValue, int, error) {
	if buf.Len() == 0 {
		return nil, 0, errIncompleteData
	}

	if start >= buf.Len() {
		return nil, 0, errIncompleteData
	}

	switch buf.Bytes()[start] {
	case byte(SIMPLE_STRING):
		s := &RESPSimpleString{}
		n, err := s.Decode(buf, start)
		return s, n, err
	case byte(INTEGER):
		i := &RESPInteger{}
		n, err := i.Decode(buf, start)
		return i, n, err
	case byte(BULK_STRING):
		b := &RESPBulkString{}
		n, err := b.Decode(buf, start)
		return b, n, err
	case byte(ERROR):
		e := &RESPError{}
		n, err := e.Decode(buf, start)
		return e, n, err
	case byte(ARRAY):
		e := &RESPArray{}
		n, err := e.Decode(buf, start)
		return e, n, err
	default:
		return nil, 0, errInvalidOpcode
	}
}

// DecodeValue decodes a single RESP value from buf beginning at index start. It
// returns the value and the number of bytes it consumed, or (nil, 0, nil) when
// the buffer does not yet hold a complete value at the top level (start == 0),
// signaling the caller to buffer more bytes and retry. A malformed value at or
// below the top level returns a non-nil error. It is the primitive [Decode]
// drives; most callers should use [Decode] rather than calling this directly.
func DecodeValue(buf *bytes.Buffer, start int) (RESPValue, int, error) {
	v, consume, err := decodeValue(buf, start)
	if err != nil {
		// we return nil, 0, nil when we are at the top
		if start == 0 && err == errIncompleteData {
			return nil, 0, nil
		}

		return nil, 0, err
	}

	if consume == 0 {
		return nil, 0, nil
	}

	return v, consume, nil
}
