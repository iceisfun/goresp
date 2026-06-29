package resp

import (
	"errors"
)

var errIncompleteData = errors.New("incomplete data")
var errUnrecoverableProtocol = errors.New("unrecoverable protocol error")
var errInvalidOpcode = errors.New("invalid opcode")
