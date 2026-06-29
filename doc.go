// Package goresp is the module root for a stream-agnostic RESP (Redis
// Serialization Protocol) parser and a self-healing Redis pub/sub connector.
//
// It has no external dependencies and does no logging. The functionality lives
// in subpackages:
//
//   - [github.com/iceisfun/goresp/resp]: streaming, resumable RESP wire codec.
//     Feed it bytes from any source (a socket, a file, a fixture) and pull
//     complete values, resuming cleanly when the stream starves mid-value.
//   - [github.com/iceisfun/goresp/connection]: self-healing pub/sub client.
//     Delivers decoded messages to an injected Handler and lifecycle/errors to
//     an injected EventHandler, with automatic reconnect and resubscribe.
//   - [github.com/iceisfun/goresp/command]: RESP command framing.
//   - [github.com/iceisfun/goresp/publish]: producer-side helper that owns a
//     dedicated (non-subscribed) connection.
//
// The runnable examples directory walks from a minimal subscriber to a
// high-throughput worker pool.
package goresp
