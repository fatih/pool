package pool

import (
	"errors"
	"net"
)

// poolConn is a wrapper around net.Conn to modify the the behavior of
// net.Conn's Close() method.
type poolConn struct {
	net.Conn
	c      *channelPool
	closed bool
}

// Close() puts the given connects back to the pool instead of closing it.
func (p poolConn) Close() error {
	if p.closed {
		return errors.New("underlying connection is closed")
	}

	return p.c.put(p.Conn)
}

// closeConn() closes the underlying connection. Usually you want to use
// Close() to put it back to the Pool. The connection cannot be put back to the
// pool after closing the underlying connection.
func (p poolConn) closeConn() error {
	p.closed = true
	return p.Close()
}

// newConn wraps a standard net.Conn to a poolConn net.Conn.
func (c *channelPool) wrapConn(conn net.Conn) net.Conn {
	p := poolConn{c: c}
	p.Conn = conn
	return p
}
