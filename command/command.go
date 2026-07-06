package command

import (
	"bytes"
	"io"

	"github.com/iceisfun/goresp/resp"
)

// FormatCommandWriter encodes args as a RESP array of bulk strings — the wire
// form Redis expects for a command — and writes it to w. The first arg is the
// command name (e.g. "SUBSCRIBE"); the rest are its arguments. It returns any
// error from encoding or from w.
func FormatCommandWriter(w io.Writer, args ...string) error {
	commandArray := &resp.RESPArray{
		Items: make([]resp.RESPValue, len(args)),
	}

	for i, arg := range args {
		commandArray.Items[i] = &resp.RESPBulkString{Value: []byte(arg)}
	}

	buf := &bytes.Buffer{}
	err := commandArray.Encode(buf)
	if err != nil {
		return err
	}

	_, err = w.Write(buf.Bytes())
	return err
}

// FormatCommand encodes args as a RESP command (see [FormatCommandWriter]) and
// returns the bytes. Encoding a command to an in-memory buffer cannot fail, so
// FormatCommand panics rather than returning an error — convenient for callers
// building fixed commands like FormatCommand("PING").
func FormatCommand(args ...string) []byte {
	buf := &bytes.Buffer{}
	err := FormatCommandWriter(buf, args...)
	if err != nil {
		panic("Failed to encode command: " + err.Error())
	}
	return buf.Bytes()
}
