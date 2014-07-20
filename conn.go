package pool

import (
	"errors"
	"net"
)

// PoolConn is a wrapper around net.Conn to modify the the behavior of
// net.Conn's Close() method.
type PoolConn struct {
	net.Conn
	c      *ChannelPool
	closed bool
}

// Close() puts the given connects back to the pool instead of closing it.
func (p PoolConn) Close() error {
	if p.closed {
		return errors.New("underlying connection is closed")
	}

	return p.c.put(p.Conn)
}

// CloseConn() closes the underlying connection. Usually you want to use
// Close() to put it back to the Pool. The connection cannot be put back to the
// pool after closing the underlying connection.
func (p PoolConn) CloseConn() error {
	p.closed = true
	return p.Close()
}

// newConn wraps a standard net.Conn to a PoolConn net.Conn.
func (c *ChannelPool) newConn(conn net.Conn) net.Conn {
	p := PoolConn{c: c}
	p.Conn = conn
	return p
}
