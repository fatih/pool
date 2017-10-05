// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package pool

import (
	"errors"
	"net"
)

var (
	// ErrClosed is the error resulting if the pool is closed via pool.Close().
	ErrClosed = errors.New("pool is closed")
	// This is the error resulting if the active connection limit is reached.
	ErrConnLimit = errors.New("connection limit reached")
)

// Pool interface describes a pool implementation. A pool should have maximum
// capacity. An ideal pool is threadsafe and easy to use.
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error.
	// Setting an active connection limit will cause this function to block if
	// the limit is reached (use TryGet() to avoid blocking).
	Get() (net.Conn, error)

	// Behaves like Get() but will return ErrConnLimit instead of blocking when
	// the active connection limit is reached.
	TryGet() (net.Conn, error)

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable.
	Close()

	// Len returns the current number of connections of the pool.
	Len() int

	// Returns total active connections.
	LenActives() int
}
