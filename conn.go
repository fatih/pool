package pool

import "net"

// PoolConn is a wrapper around net.Conn to modify the the behavior of
// net.Conn's Close() method.
type PoolConn struct {
	net.Conn
	c          *channelPool
	unreusable bool
}

// Close() puts the given connects back to the pool instead of closing it.
func (p PoolConn) Close() error {
	if p.unreusable {
		if p.Conn != nil {
			p.Conn.Close()
		}
		return nil
	}
	return p.c.put(p.Conn)
}

// MarkUnreusable() marks the connection is not reusable.
func (p *PoolConn) MarkUnreusable() {
	p.unreusable = true
}

// newConn wraps a standard net.Conn to a poolConn net.Conn.
func (c *channelPool) wrapConn(conn net.Conn) net.Conn {
	p := &PoolConn{c: c}
	p.Conn = conn
	return p
}
