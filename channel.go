package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// channelPool implements the Pool interface based on buffered channels.
type channelPool struct {
	// storage for our net.Conn connections
	mu    sync.Mutex
	conns chan net.Conn

	// active connections limiter
	isLimited bool
	actives   chan struct{}

	// net.Conn generator
	factory Factory
}

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// NewChannelPool returns a new pool based on buffered channels with an initial
// capacity and maximum capacity. Factory is used when initial capacity is
// greater than zero to fill the pool. A zero initialCap doesn't fill the Pool
// until a new Get() is called. During a Get(), If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method (unless maxActive > 0, i.e. there is a limit for active connections).
func NewChannelPoolMaxActive(initialCap, maxCap int, maxActive int, factory Factory) (Pool, error) {
	if initialCap < 0 || maxCap <= 0 || maxActive < 0 || initialCap > maxCap ||
		(maxActive > 0 && maxActive < maxCap) {
		return nil, errors.New("invalid capacity settings")
	}

	c := &channelPool{
		conns:   make(chan net.Conn, maxCap),
		factory: factory,
	}
	if maxActive > 0 {
		c.isLimited = true
		c.actives = make(chan struct{}, maxActive)
	}

	// create initial connections, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		conn, err := c.tryOpen()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.conns <- conn
	}

	return c, nil
}

func NewChannelPool(initialCap, maxCap int, factory Factory) (Pool, error) {
	return NewChannelPoolMaxActive(initialCap, maxCap, 0, factory)
}

func (c *channelPool) tryOpen() (net.Conn, error) {
	// this will block if active connections are limited.
	if c.isLimited {
		c.actives <- struct{}{}
	}
	conn, err := c.factory()
	if err != nil {
		c.tryClose(conn)
	}
	return conn, err
}

func (c *channelPool) tryClose(conn net.Conn) error {
	// update active connection limit.
	if c.isLimited {
		<-c.actives
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *channelPool) getConns() chan net.Conn {
	c.mu.Lock()
	conns := c.conns
	c.mu.Unlock()
	return conns
}

func (c *channelPool) getActives() chan struct{} {
	c.mu.Lock()
	actives := c.actives
	c.mu.Unlock()
	return actives
}

// Get implements the Pool interfaces Get() method. If there is no new
// connection available in the pool, a new connection will be created via the
// Factory() method.
func (c *channelPool) Get() (net.Conn, error) {
	conns := c.getConns()
	if conns == nil {
		return nil, ErrClosed
	}

	// wrap our connections with out custom net.Conn implementation (wrapConn
	// method) that puts the connection back to the pool if it's closed.
	select {
	case conn := <-conns:
		if conn == nil {
			return nil, ErrClosed
		}

		return c.wrapConn(conn), nil
	default:
		conn, err := c.tryOpen()
		if err != nil {
			return nil, err
		}

		return c.wrapConn(conn), nil
	}
}

// put puts the connection back to the pool. If the pool is full or closed,
// conn is simply closed. A nil conn will be rejected.
func (c *channelPool) put(conn net.Conn) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conns == nil {
		// pool is closed, close passed connection
		return conn.Close()
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case c.conns <- conn:
		return nil
	default:
		// pool is full, close passed connection
		return c.tryClose(conn)
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	conns := c.conns
	actives := c.actives
	c.conns = nil
	c.actives = nil
	c.factory = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for conn := range conns {
		conn.Close()
	}
	if c.isLimited {
		close(actives)
		for _ = range actives {
		}
	}
}

func (c *channelPool) Len() int        { return len(c.getConns()) }
func (c *channelPool) LenActives() int { return len(c.getActives()) }
