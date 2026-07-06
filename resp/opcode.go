package resp

// OPCODE is the leading byte that identifies a RESP value's wire type.
type OPCODE byte

// The RESP type-prefix bytes. Each decoded value begins with one of these.
const (
	SIMPLE_STRING OPCODE = '+' // +OK\r\n
	ERROR         OPCODE = '-' // -ERR message\r\n
	INTEGER       OPCODE = ':' // :1000\r\n
	BULK_STRING   OPCODE = '$' // $6\r\nfoobar\r\n ($-1 is null)
	ARRAY         OPCODE = '*' // *2\r\n... (*-1 is null)
)

// PROTOCOL_SEPARATOR is the CRLF that terminates every RESP line. Treat it as
// read-only.
var PROTOCOL_SEPARATOR = []byte{'\r', '\n'}
