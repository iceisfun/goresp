package resp

// scanValue performs a non-allocating structural scan of one RESP value
// starting at b[pos]. It is used purely to decide whether the buffer holds
// enough bytes to attempt a (real, allocating) decode.
//
// It returns:
//
//	next - the index immediately past the value when ok is true, otherwise a
//	       lower bound on the total number of bytes required before a complete
//	       value can exist (always > len(b) when ok is false).
//	ok   - true when a complete value is present in b[pos:].
//
// On any structural ambiguity (malformed length prefix, unknown opcode) it
// returns ok=true with a minimal next so the real decoder runs and surfaces
// the precise error. It never reports a need larger than reality, so callers
// can safely skip a decode attempt while len(b) < next.
func scanValue(b []byte, pos int) (next int, ok bool) {
	if pos >= len(b) {
		return len(b) + 1, false
	}

	switch b[pos] {
	case byte(SIMPLE_STRING), byte(ERROR), byte(INTEGER):
		i := indexCRLF(b, pos+1)
		if i < 0 {
			return len(b) + 1, false
		}
		return i + len(PROTOCOL_SEPARATOR), true

	case byte(BULK_STRING):
		i := indexCRLF(b, pos+1)
		if i < 0 {
			return len(b) + 1, false
		}
		n, valid := parseLen(b[pos+1 : i])
		header := i + len(PROTOCOL_SEPARATOR)
		if !valid || n < 0 {
			// null bulk string ($-1) or a malformed length: the header alone
			// is a complete (or decodable) unit.
			return header, true
		}
		end := header + n + len(PROTOCOL_SEPARATOR)
		if end > len(b) {
			return end, false
		}
		return end, true

	case byte(ARRAY):
		i := indexCRLF(b, pos+1)
		if i < 0 {
			return len(b) + 1, false
		}
		n, valid := parseLen(b[pos+1 : i])
		cur := i + len(PROTOCOL_SEPARATOR)
		if !valid || n < 0 {
			return cur, true // null/malformed array header
		}
		for range n {
			next, ok := scanValue(b, cur)
			if !ok {
				return next, false
			}
			cur = next
		}
		return cur, true

	default:
		// Unknown opcode: let the real decoder reject it.
		return pos + 1, true
	}
}

// indexCRLF returns the index of the first "\r\n" at or after start, or -1.
func indexCRLF(b []byte, start int) int {
	for i := start; i+1 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' {
			return i
		}
	}
	return -1
}

// parseLen parses a RESP length prefix (an optionally negative integer). It
// mirrors what the real decoder accepts; on anything unexpected it reports
// valid=false so the caller defers to the real decoder.
func parseLen(b []byte) (n int, valid bool) {
	if len(b) == 0 {
		return 0, false
	}
	i := 0
	neg := false
	if b[0] == '-' {
		neg = true
		i = 1
		if len(b) == 1 {
			return 0, false
		}
	}
	for ; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}
