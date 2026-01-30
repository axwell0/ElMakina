// Package websocket provides a WebSocket-backed InputProvider for the engine.
//
// The package implements a request/response protocol with request_id correlation.
// It is intentionally transport-focused: game rules and validation remain in the
// engine and actions layers.
package websocket
