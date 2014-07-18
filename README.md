# Pool [![GoDoc](https://godoc.org/github.com/fatih/pool?status.svg)](http://godoc.org/github.com/fatih/pool) [![Build Status](https://travis-ci.org/fatih/pool.svg)](https://travis-ci.org/fatih/pool)


Pool is a thread safe connection pool for net.Conn interface. It can be used to
manage and reuse connections.


## Install and Usage

Install the package with:

```bash
go get gopkg.in/fatih/pool.v1
```

Import it with:

```go
import "gopkg.in/fatih/pool.v1"
```

and use `pool` as the package name inside the code.

## Example

```go

// create a factory() to be used with pool
factory    := func() (net.Conn, error) { return net.Dial("tcp", "127.0.0.1:4000") }

// create a new pool with an initial capacity of 5 and maximum capacity of
// 30. The factory will create 5 initial connections and put it into the pool
p, err := pool.NewChannelPool(5, 30, factory)

// now you can get a connection from the pool, if there is no connection
// available it will create a new one via the factory function.
conn, err := p.Get()

// do something with conn and put it back to the pool
p.Put(conn)

// close pool any time you want
p.Close()

// currently available connections in the pool
current := p.CurrentCapacity()

// maximum capacity of your pool
max := p.MaximumCapacity()
```


## Credits

 * [Fatih Arslan](https://github.com/fatih)
 * [sougou](https://github.com/sougou)

## License

The MIT License (MIT) - see LICENSE for more details
