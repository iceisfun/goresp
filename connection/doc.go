// Package connection provides a self-healing Redis pub/sub client.
//
// [New] dials a Redis server and returns a [Reconnecting] that automatically
// reconnects with backoff and restores subscriptions. Decoded messages are
// delivered to an injected [Handler]; connection lifecycle and errors are
// reported to an optional injected [EventHandler], whose OnConnect receives a
// [Subscriber] for establishing subscriptions on every connect. Behavior is
// tuned with the With* [Option] values (events, context, keepalive, timeouts).
//
// The package interprets only pub/sub frames; the payload schema is left to the
// caller. The library performs no logging of its own.
package connection
