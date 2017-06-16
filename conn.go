package pool

import (
	"net"
	"sync"
)

// PoolConn is a wrapper around net.Conn to modify the the behavior of
// net.Conn's Close() method.
type PoolConn struct {
	net.Conn
	mu       sync.RWMutex
	c        *channelPool
	unusable bool
}

func (p *PoolConn) Read(b []byte) (int, error) {
	n, err := p.Conn.Read(b)
	if terr, ok := err.(net.Error); ok && !terr.Timeout() {
		p.MarkUnusable()
	}
	return n, err
}

func (p *PoolConn) Write(b []byte) (int, error) {
	n, err := p.Conn.Write(b)
	if terr, ok := err.(net.Error); ok && !terr.Timeout() {
		p.MarkUnusable()
	}
	return n, err
}

// Close() puts the given connects back to the pool instead of closing it.
func (p *PoolConn) Close() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.unusable {
		return p.c.tryClose(p.Conn)
	}
	return p.c.put(p.Conn)
}

// MarkUnusable() marks the connection not usable any more, to let the pool close it instead of returning it to pool.
func (p *PoolConn) MarkUnusable() {
	p.mu.Lock()
	p.unusable = true
	p.mu.Unlock()
}

// newConn wraps a standard net.Conn to a poolConn net.Conn.
func (c *channelPool) wrapConn(conn net.Conn) net.Conn {
	p := &PoolConn{c: c}
	p.Conn = conn
	return p
}
