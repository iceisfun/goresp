package publish

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/iceisfun/goresp/command"
	"github.com/iceisfun/goresp/resp"
)

// syncConn is a minimal synchronous Redis request/reply connection used by the
// context-aware publish methods. Exactly one command is in flight at a time
// (guarded by mu), so replies need no correlation: write a command, read its one
// reply. It dials lazily and redials on the next call after any error. This is
// the right model for a non-subscribed connection, which Redis treats as plain
// request/reply.
type syncConn struct {
	addr        string
	dialTimeout time.Duration

	mu   sync.Mutex
	conn net.Conn
	dec  *resp.Decode
}

func newSyncConn(addr string, dialTimeout time.Duration) *syncConn {
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}
	return &syncConn{addr: addr, dialTimeout: dialTimeout, dec: resp.NewDecode()}
}

// do sends one command and returns its single reply. ctx bounds the whole
// operation: a deadline maps onto socket deadlines, and a cancellation unblocks
// an in-progress read/write promptly.
func (s *syncConn) do(ctx context.Context, args ...string) (resp.RESPValue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.ensure(ctx); err != nil {
		return nil, err
	}

	// Force a past deadline on cancellation so a blocked Read/Write returns at
	// once; we then translate the resulting error into ctx.Err below. Capture
	// the conn locally so reset() niling s.conn cannot race the callback.
	conn := s.conn
	stop := context.AfterFunc(ctx, func() { conn.SetDeadline(time.Unix(0, 0)) })
	defer stop()

	if _, err := conn.Write(command.FormatCommand(args...)); err != nil {
		s.reset()
		return nil, ctxOr(ctx, err)
	}

	v, err := s.read()
	if err != nil {
		s.reset()
		return nil, ctxOr(ctx, err)
	}
	return v, nil
}

func (s *syncConn) ensure(ctx context.Context) error {
	if s.conn != nil {
		return nil
	}
	d := net.Dialer{Timeout: s.dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return err
	}
	s.conn = conn
	s.dec.Reset()
	return nil
}

func (s *syncConn) read() (resp.RESPValue, error) {
	buf := make([]byte, 4096)
	for {
		v, err := s.dec.Parse()
		if err != nil {
			return nil, err
		}
		if v != nil {
			return v, nil
		}
		n, err := s.conn.Read(buf)
		if n > 0 {
			s.dec.Provide(buf[:n])
		}
		if err != nil {
			return nil, err
		}
	}
}

func (s *syncConn) reset() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.dec.Reset()
}

func (s *syncConn) close() {
	s.mu.Lock()
	s.reset()
	s.mu.Unlock()
}

// ctxOr prefers the context error (cancel/deadline) over a derived socket error,
// so callers see context.Canceled / context.DeadlineExceeded as expected.
func ctxOr(ctx context.Context, err error) error {
	if ce := ctx.Err(); ce != nil {
		return ce
	}
	return err
}
