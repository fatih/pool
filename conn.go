package pool

import "net"

// poolConn is a wrapper around net.Conn to modify the the behavior of
// net.Conn's Close() method
type poolConn struct {
	net.Conn
	c *ChannelPool
}

// Close() puths the given connects back to the pool instead of closing it
func (p poolConn) Close() error {
	return p.c.put(p.Conn)
}

// newConn wraps a standard net.Conn to a poolConn net.Conn.
func (c *ChannelPool) newConn(conn net.Conn) net.Conn {
	p := poolConn{c: c}
	p.Conn = conn
	return p
}
