package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// Pool allows you to use a pool of net.Conn connections.
type ChannelPool struct {
	// storage for our net.Conn connections
	mu    sync.Mutex
	conns chan net.Conn

	// net.Conn generator
	factory Factory
}

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// NewChannelPool returns a new pool based on buffered channels with an initial
// capacity and maximum capacity. Factory is used when initial capacity is
// greater than zero to fill the pool.
func NewChannelPool(initialCap, maxCap int, factory Factory) (Pool, error) {
	if initialCap <= 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	c := &ChannelPool{
		conns:   make(chan net.Conn, maxCap),
		factory: factory,
	}

	// create initial connections, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.conns <- conn
	}

	return c, nil
}

func (c *ChannelPool) getConns() chan net.Conn {
	c.mu.Lock()
	conns := c.conns
	c.mu.Unlock()
	return conns
}

// Get returns a new connection from the pool. After using the connection it
// should be put back via the Put() method. If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func (c *ChannelPool) Get() (net.Conn, error) {
	conns := c.getConns()
	if conns == nil {
		return nil, ErrClosed
	}

	select {
	case conn := <-conns:
		if conn == nil {
			return nil, ErrClosed
		}
		return conn, nil
	default:
		return c.factory()
	}
}

// Put puts an existing connection into the pool. If the pool is full or
// closed, conn is simply closed. A nil conn will be rejected. Putting into a
// destroyed or full pool will be counted as an error.
func (c *ChannelPool) Put(conn net.Conn) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conns == nil {
		conn.Close()
		return ErrClosed
	}

	select {
	case c.conns <- conn:
		return nil
	default:
		conn.Close()
		return ErrFull
	}
}

// Close closes the pool and all its connections. After Close() the
// pool is no longer usable.
func (c *ChannelPool) Close() {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for conn := range conns {
		conn.Close()
	}
}

// Cap returns the maximum capacity of the pool
func (c *ChannelPool) Cap() int { return cap(c.getConns()) }

// Len returns the current capacity of the pool.
func (c *ChannelPool) Len() int { return len(c.getConns()) }
