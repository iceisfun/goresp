package resp

type OPCODE byte

const (
	SIMPLE_STRING OPCODE = '+'
	ERROR         OPCODE = '-'
	INTEGER       OPCODE = ':'
	BULK_STRING   OPCODE = '$'
	ARRAY         OPCODE = '*'
)

var PROTOCOL_SEPARATOR = []byte{'\r', '\n'}
