package command_test

import (
	"bytes"
	"testing"

	"github.com/iceisfun/goresp/command"
)

func TestFormatCommandWriter(t *testing.T) {
	buffer := &bytes.Buffer{}
	args := []string{"SET", "key", "value"}

	if err := command.FormatCommandWriter(buffer, args...); err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	expected := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
	if got := buffer.String(); got != expected {
		t.Errorf("Expected %q, but got %q", expected, got)
	}
}

func TestFormatCommand(t *testing.T) {
	args := []string{"GET", "key"}
	expected := "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n"
	got := command.FormatCommand(args...)

	if string(got) != expected {
		t.Errorf("Expected %q, but got %q", expected, string(got))
	}
}
