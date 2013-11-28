// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package pool

import (
	"errors"
	"fmt"
	"log"
	"net"
)

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// Pool allows you to use a pool of resources.
type Pool struct {
	conns   chan net.Conn
	factory Factory
}

// New returns a new pool with an initial capacity and maximum capacity.
// Factory is used when initial capacity is greater then zero to fill the
// pool.
func New(initialCap, maxCap int, factory Factory) (*Pool, error) {
	if initialCap <= 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	p := &Pool{
		conns:   make(chan net.Conn, maxCap),
		factory: factory,
	}

	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			return nil, fmt.Errorf("WARNING: factory is not able to fill the pool: %s", err)
		}

		p.conns <- conn
	}

	return p, nil
}

// Get returns a new connection from the pool. After using the connection it
// should be put back via the Put() method. If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func (p *Pool) Get() (net.Conn, error) {
	select {
	case conn, ok := <-p.conns:
		if !ok {
			return nil, errors.New("channel is closed")
		}
		return conn, nil
	default:
		return p.factory()
	}
}

// Put puts a new connection into the pool. If the pool is full, conn is
// discarded and a warning is output to stderr.
func (p *Pool) Put(conn net.Conn) {
	select {
	case p.conns <- conn:
	default:
		log.Println("WARNING: attempt to put into a full pool")
	}

}

// Close closes all the pools connections
func (p *Pool) Close() { close(p.conns) }

// MaximumCapacity returns the maximum capacity of the pool
func (p *Pool) MaximumCapacity() int { return cap(p.conns) }

// UsedCapacity returns the used capacity of the pool.
func (p *Pool) UsedCapacity() int { return len(p.conns) }
