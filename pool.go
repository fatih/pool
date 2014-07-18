// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

var (
	// ErrPoolClosed is the error resulting if the pool is closed via
	// pool.Close().
	ErrPoolClosed = errors.New("pool is closed")

	// ErrPoolFull is the error resulting if the pool is full.
	ErrPoolFull = errors.New("pool is full")
)

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// Pool allows you to use a pool of net.Conn connections.
type Pool struct {
	// storage for our net.Conn connections
	mu    sync.Mutex
	conns chan net.Conn

	// net.Conn generator
	factory Factory
}

// New returns a new pool with an initial capacity and maximum capacity.
// Factory is used when initial capacity is greater than zero to fill the
// pool.
func New(initialCap, maxCap int, factory Factory) (*Pool, error) {
	if initialCap <= 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	p := &Pool{
		conns:   make(chan net.Conn, maxCap),
		factory: factory,
	}

	// create initial connections, if something goes wrong,
	// just close the pool error out.
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

func (p *Pool) getConns() chan net.Conn {
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
	if conns == nil {
		return nil, ErrPoolClosed
	}

	select {
	case conn := <-conns:
		if conn == nil {
			return nil, ErrPoolClosed
		}
		return conn, nil
	default:
		return p.factory()
	}
}

// Put puts an existing connection into the pool. If the pool is full or
// closed, conn is simply closed. A nil conn will be rejected. Putting into a
// destroyed or full pool will be counted as an error.
func (p *Pool) Put(conn net.Conn) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conns == nil {
		conn.Close()
		return ErrPoolClosed
	}

	select {
	case p.conns <- conn:
		return nil
	default:
		conn.Close()
		return ErrPoolFull
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

// CurrentCapacity returns the current capacity of the pool.
func (p *Pool) CurrentCapacity() int { return len(p.getConns()) }
