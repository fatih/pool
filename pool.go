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

// Get returns a new connection from the pool. After using the connection it
// should be put back via the Put() method. If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func (p *Pool) Get() (net.Conn, error) {
	conn, err := p.get()
	if err != nil {
		return nil, err
	}
	if conn != nil {
		return conn, nil
	}
	return p.factory()
}

func (p *Pool) get() (net.Conn, error) (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns == nil  {
		return nil, errors.New("pool is closed")
	}

	select {
	case conn := <-p.conns:
		return conn, nil
	default:
	}
	return nil, nil
}

// Put puts an existing connection into the pool. If the pool is full or closed, conn is
// simply closed.
func (p *Pool) Put(conn net.Conn) {
	if !p.put(conn) {
		conn.Close()
	}
}

func (p *Pool) put(conn net.Conn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns == nil  {
		return false
	}

	select {
	case p.conns <- conn:
		return true
	default:
	}
	return false
}

// Close closes the pool and all its connections. After Close() the
// pool is no longer usable.
func (p *Pool) Close() {
	conns := p.closePool()
	if conns == nil {
		return
	}
	for conn := range conns {
		conn.Close()
	}
}

func (p *Pool) closePool() chan net.Conn {
	p.mu.Lock()
	defer p.mu.Lock()
	conns := p.conns
	p.conns = nil
	p.factory = nil

	if conns == nil {
		return nil
	}
	return conns
}

// MaximumCapacity returns the maximum capacity of the pool
func (p *Pool) MaximumCapacity() int { return cap(p.getConns()) }

// UsedCapacity returns the used capacity of the pool.
func (p *Pool) UsedCapacity() int { return len(p.getConns()) }
