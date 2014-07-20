package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// ChannelPool implements the Pool interface based on buffered channels.
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
// greater than zero to fill the pool. A zero initialCap doesn't fill the Pool
// until a new Get() is called.
func NewChannelPool(initialCap, maxCap int, factory Factory) (Pool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
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

// Get implements the Pool interfaces Get() method. If there is no new
// connection available in the pool, a new connection will be created via the
// Factory() method.
func (c *ChannelPool) Get() (conn net.Conn, err error) {
	conns := c.getConns()
	if conns == nil {
		return nil, ErrClosed
	}

	defer func() {
		if err == nil {
			conn = c.newConn(conn)
		}
	}()

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

// put puts the connection back to the pool. If the pool is full or closed,
// conn is simply closed. A nil conn will be rejected.
func (c *ChannelPool) put(conn net.Conn) error {
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

func (c *ChannelPool) Len() int { return len(c.getConns()) }
