package publish_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/iceisfun/goresp/publish"
	"github.com/iceisfun/goresp/resp"
)

// fakeRedis is a minimal server that reads RESP commands and replies with a
// scripted reply per command. reply is raw RESP bytes (e.g. ":3\r\n").
func fakeRedis(t *testing.T, reply []byte, replyDelay time.Duration) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				dec := resp.NewDecode()
				buf := make([]byte, 4096)
				for {
					n, err := conn.Read(buf)
					if n > 0 {
						dec.Provide(buf[:n])
						for {
							v, perr := dec.Parse()
							if perr != nil || v == nil {
								break
							}
							select {
							case <-time.After(replyDelay):
							case <-done:
								return
							}
							conn.Write(reply)
						}
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()
	return ln.Addr().String(), func() { close(done); ln.Close() }
}

func TestPublishCtxReturnsSubscriberCount(t *testing.T) {
	addr, stop := fakeRedis(t, []byte(":3\r\n"), 0)
	defer stop()

	p := publish.New(addr)
	defer p.Close()

	n, err := p.PublishCtx(context.Background(), "events", []byte("hello"))
	if err != nil {
		t.Fatalf("PublishCtx: %v", err)
	}
	if n != 3 {
		t.Fatalf("subscriber count = %d, want 3", n)
	}
}

func TestPublishCtxRedisError(t *testing.T) {
	addr, stop := fakeRedis(t, []byte("-ERR wrong number of arguments\r\n"), 0)
	defer stop()

	p := publish.New(addr)
	defer p.Close()

	_, err := p.PublishCtx(context.Background(), "events", []byte("x"))
	var re *publish.RedisError
	if !errors.As(err, &re) {
		t.Fatalf("got %v, want *publish.RedisError", err)
	}
}

func TestPublishJSONCtx(t *testing.T) {
	addr, stop := fakeRedis(t, []byte(":1\r\n"), 0)
	defer stop()

	p := publish.New(addr)
	defer p.Close()

	n, err := p.PublishJSONCtx(context.Background(), "events", map[string]int{"id": 7})
	if err != nil || n != 1 {
		t.Fatalf("PublishJSONCtx = (%d, %v), want (1, nil)", n, err)
	}
}

func TestPublishCtxTimeout(t *testing.T) {
	// Server delays its reply well past the context deadline.
	addr, stop := fakeRedis(t, []byte(":3\r\n"), 500*time.Millisecond)
	defer stop()

	p := publish.New(addr)
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.PublishCtx(ctx, "events", []byte("slow"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
}

func TestPublishCtxSequential(t *testing.T) {
	addr, stop := fakeRedis(t, []byte(":2\r\n"), 0)
	defer stop()

	p := publish.New(addr)
	defer p.Close()

	for i := range 5 {
		n, err := p.PublishCtx(context.Background(), "events", []byte("msg"))
		if err != nil || n != 2 {
			t.Fatalf("iteration %d: (%d, %v)", i, n, err)
		}
	}
}

func TestConfirmUnavailableOnWrap(t *testing.T) {
	p := publish.Wrap(&fakeSender{})
	_, err := p.PublishCtx(context.Background(), "events", []byte("x"))
	if !errors.Is(err, publish.ErrConfirmUnavailable) {
		t.Fatalf("got %v, want ErrConfirmUnavailable", err)
	}
}
