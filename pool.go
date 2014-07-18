// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package pool

import (
	"errors"
	"net"
)

var (
	// ErrPoolClosed is the error resulting if the pool is closed via
	// pool.Close().
	ErrPoolClosed = errors.New("pool is closed")

	// ErrPoolFull is the error resulting if the pool is full.
	ErrPoolFull = errors.New("pool is full")
)

// Pool interface describes a pool implementation.
type Pool interface {
	// Get returns a new connection from the pool. After using the connection it
	// should be put back via the Put() method. If there is no new connection
	// available in the pool, a new connection will be created via the Factory()
	// method.
	Get() (net.Conn, error)

	// Put puts an existing connection into the pool. If the pool is full or
	// closed, conn is simply closed. A nil conn will be rejected. Putting into a
	// destroyed or full pool will be counted as an error.
	Put(conn net.Conn) error

	// Close closes the pool and all its connections. After Close() the
	// pool is no longer usable.
	Close()

	// MaximumCapacity returns the maximum capacity of the pool
	MaximumCapacity() int

	// CurrentCapacity returns the current capacity of the pool.
	CurrentCapacity() int
}
