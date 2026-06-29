package connection

import "errors"

var (
	// ErrCommandQueueFull is reported via Events.OnError when Send cannot enqueue
	// a command because the outbound queue is full.
	ErrCommandQueueFull = errors.New("connection: command queue full")
	// ErrStreamStalled is passed to OnDisconnect when the connection is torn
	// down because no data arrived within the health-check window even though no
	// socket error occurred.
	ErrStreamStalled = errors.New("connection: stream stalled (no data)")
)
