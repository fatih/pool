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
	mu sync.Mutex
	conns chan net.Conn

	// net.Conn generator
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

	// create initial connections, if
	// something goes wrong, close them all via closeConns().
	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		p.conns <- conn
	}

	return p, nil
}

func (p *Pool) getConns() chan net.Conns {
	p.mu.Lock()
	conns := p.conns
	p.mu.Unlock()
	return conns
}

// Get returns a new connection from the pool. After using the connection it
// should be put back via the Put() method. If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func (p *Pool) Get() (net.Conn, error) {
	conns := p.getConns()
	if conns == nil  {
		return nil, errors.New("pool is destroyed")
	}

	select {
	case conn := <-conns:
		return conn, nil
	default:
		return p.factory()
	}
}

// Put puts an existing connection into the pool. If the pool is full or closed, conn is
// simply closed.
func (p *Pool) Put(conn net.Conn) {
	conns := p.getConns()
	if conns == nil  {
		conn.Close()
		return
	}

	select {
	case conns <- conn:
		// No-op
	default:
		conn.Close()
	}
}

// Close closes the pool and all its connections. After Close() the
// pool is no longer usable.
func (p *Pool) Close() {
	p.mu.Lock()
	conns := p.conns
	p.conns = nil
	p.factory = nil
	p.mu.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for conn := range conns {
		conn.Close()
	}
}

// MaximumCapacity returns the maximum capacity of the pool
func (p *Pool) MaximumCapacity() int { return cap(p.getConns()) }

// UsedCapacity returns the used capacity of the pool.
func (p *Pool) UsedCapacity() int { return len(p.getConns()) }
