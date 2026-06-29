package command

import (
	"bytes"
	"io"

	"github.com/iceisfun/goresp/resp"
)

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

func FormatCommand(args ...string) []byte {
	buf := &bytes.Buffer{}
	err := FormatCommandWriter(buf, args...)
	if err != nil {
		panic("Failed to encode command: " + err.Error())
	}
	return buf.Bytes()
}
