package publish_test

import (
	"testing"

	"github.com/iceisfun/goresp/publish"
)

type fakeSender struct {
	sent [][]byte
	full bool
}

func (f *fakeSender) Send(cmd []byte) bool {
	if f.full {
		return false
	}
	f.sent = append(f.sent, append([]byte(nil), cmd...))
	return true
}

func TestPublish(t *testing.T) {
	fs := &fakeSender{}
	p := publish.Wrap(fs)

	if err := p.Publish("chan", []byte("hello")); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	want := "*3\r\n$7\r\nPUBLISH\r\n$4\r\nchan\r\n$5\r\nhello\r\n"
	if len(fs.sent) != 1 || string(fs.sent[0]) != want {
		t.Fatalf("sent %q, want %q", fs.sent, want)
	}
}

func TestPublishJSON(t *testing.T) {
	fs := &fakeSender{}
	p := publish.Wrap(fs)

	if err := p.PublishJSON("events", map[string]int{"n": 1}); err != nil {
		t.Fatalf("PublishJSON: %v", err)
	}
	want := "*3\r\n$7\r\nPUBLISH\r\n$6\r\nevents\r\n$7\r\n{\"n\":1}\r\n"
	if len(fs.sent) != 1 || string(fs.sent[0]) != want {
		t.Fatalf("sent %q, want %q", fs.sent[0], want)
	}
}

func TestPublishQueueFull(t *testing.T) {
	fs := &fakeSender{full: true}
	p := publish.Wrap(fs)

	if err := p.Publish("chan", []byte("x")); err != publish.ErrQueueFull {
		t.Fatalf("got %v, want ErrQueueFull", err)
	}
}
