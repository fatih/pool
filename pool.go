// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// Pool allows you to use a pool of net.Conn connections.
type Pool struct {
	// storage for our net.Conn connections
	conns chan net.Conn

	// net.Conn generator
	factory Factory

	// to prevent access to closed channels
	isDestroyed bool

	mu sync.Mutex // protects isDesroyed field
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

	bufferedConn := make([]net.Conn, initialCap)
	closeConns := func() {
		for _, conn := range bufferedConn {
			conn.Close()
		}
	}

	// create initial connections and buffer all successfull connections, if
	// something goes wrong, close them all via closeConns().
	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			closeConns()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}

		bufferedConn[i] = conn
	}

	// if everything goes right, send the connections to our pool
	for _, conn := range bufferedConn {
		p.conns <- conn
	}

	bufferedConn = nil

	return p, nil
}

// Get returns a new connection from the pool. After using the connection it
// should be put back via the Put() method. If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func (p *Pool) Get() (net.Conn, error) {
	p.mu.Lock()
	if p.isDestroyed {
		p.mu.Unlock()
		return nil, errors.New("pool is destroyed")
	}
	p.mu.Unlock()

	select {
	case conn := <-p.conns:
		return conn, nil
	default:
		return p.factory()
	}
}

// Put puts a new connection into the pool. If the pool is full, conn is
// discarded and a warning is output to stderr.
func (p *Pool) Put(conn net.Conn) error {
	p.mu.Lock()
	if p.isDestroyed {
		p.mu.Unlock()
		return errors.New("pool is destroyed")
	}
	p.mu.Unlock()

	select {
	case p.conns <- conn:
		return nil
	default:
		return errors.New("attempt to put into a full pool")
	}
}

// Destroy destroys the pool and close all connections. After Destroy() the
// pool is no longer usable.
func (p *Pool) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isDestroyed {
		return
	}

	// close all connections
	for i := 0; i < p.MaximumCapacity(); i++ {
		select {
		case conn := <-p.conns:
			conn.Close()
		default:
			continue
		}
	}

	// reset variables, make them unusable.
	close(p.conns)
	p.conns = nil
	p.factory = nil
	p.isDestroyed = true
}

// MaximumCapacity returns the maximum capacity of the pool
func (p *Pool) MaximumCapacity() int { return cap(p.conns) }

// UsedCapacity returns the used capacity of the pool.
func (p *Pool) UsedCapacity() int { return len(p.conns) }
