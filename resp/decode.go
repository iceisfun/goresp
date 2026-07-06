package resp

import (
	"bytes"
)

// Decode is a streaming RESP decoder. It is stream-agnostic: feed it bytes
// from any source (a socket, a file, a fixture) with Provide and pull complete
// values with Parse. When the buffered data ends in the middle of a value,
// Parse returns (nil, nil) and the decoder retains its state, resuming exactly
// where it left off once more data is provided. This makes it safe to drive
// from a stream that starves at arbitrary packet boundaries.
type Decode struct {
	buffer bytes.Buffer
	// need is the number of buffered bytes required before the next value can
	// possibly be complete. While the buffer holds fewer bytes Parse short
	// circuits without a full descent, so a starved stream does not re-scan the
	// same partial value on every Provide. Zero means "unknown, always try".
	need int
}

// NewDecode returns a ready-to-use streaming decoder with an empty buffer.
func NewDecode() *Decode {
	return &Decode{}
}

// Provide adds data to the decoder's buffer. The bytes are copied, so the
// caller may reuse the slice immediately.
func (p *Decode) Provide(data []byte) {
	p.buffer.Write(data)
}

// Reset clears the decoder's buffer and resume state, usually after a reconnect.
func (p *Decode) Reset() {
	p.buffer.Reset()
	p.need = 0
}

// Parse attempts to parse a complete RESP value from the current buffer. It
// returns (nil, nil) when more data is required (the buffer is left intact so a
// subsequent Provide + Parse resumes), a non-nil value when one is decoded, or
// an error on malformed input.
func (p *Decode) Parse() (RESPValue, error) {
	if p.buffer.Len() < p.need {
		// Known starvation: not enough bytes buffered yet to complete the value
		// we last looked at. Skip the descent entirely.
		return nil, nil
	}

	value, bytesConsumed, err := DecodeValue(&p.buffer, 0)
	if err != nil {
		return nil, err
	}

	if value == nil {
		// Incomplete: remember how many bytes we need so the next Provide that
		// falls short can short circuit instead of re-descending.
		if n, ok := scanValue(p.buffer.Bytes(), 0); !ok {
			p.need = n
		}
		return nil, nil
	}

	p.buffer.Next(bytesConsumed)
	p.need = 0
	return value, nil
}

// HasData reports whether the buffer holds enough bytes to potentially contain
// a complete RESP value (an opcode, a body, and a trailing CRLF).
func (p *Decode) HasData() bool {
	return p.buffer.Len() > 3
}
