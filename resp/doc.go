// Package resp implements the Redis Serialization Protocol (RESP) wire format
// as a streaming, resumable codec.
//
// The five RESP types used by pub/sub — simple strings, errors, integers, bulk
// strings, and arrays — each implement [RESPValue] (encode and decode).
//
// [Decode] is the stream-agnostic entry point: feed it bytes with Provide and
// pull complete values with Parse. When the buffered data ends mid-value, Parse
// returns (nil, nil) and retains its state, resuming on the next Provide. This
// makes it safe to drive from a live socket or to replay a captured stream from
// disk.
package resp
