package pool

import (
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

var (
	InitialCap = 5
	MaximumCap = 30
	network    = "tcp"
	address    = "127.0.0.1:7777"
	factory    = func() (net.Conn, error) { return net.Dial(network, address) }
	testPool   = &Pool{}
)

func init() {
	// used for factory function
	go simpleTCPServer()
	time.Sleep(time.Millisecond * 300) // wait until tcp server has been settled

	var err error
	testPool, err = newPool()
	if err != nil {
		log.Fatalln(err)
	}
}

func TestNew(t *testing.T) {
	_, err := newPool()
	if err != nil {
		t.Errorf("New error: %s", err)
	}
}

func TestPool_Get(t *testing.T) {
	_, err := testPool.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	if testPool.CurrentCapacity() != (InitialCap - 1) {
		t.Errorf("Get error. Expecting %d, got %d",
			(InitialCap - 1), testPool.CurrentCapacity())
	}

	var wg sync.WaitGroup
	for i := 0; i < (InitialCap - 1); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testPool.Get()
			if err != nil {
				t.Errorf("Get error: %s", err)
			}
		}()
	}

	wg.Wait()

	if testPool.CurrentCapacity() != 0 {
		t.Errorf("Get error. Expecting %d, got %d",
			(InitialCap - 1), testPool.CurrentCapacity())
	}

	_, err = testPool.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}
}

func TestPool_Put(t *testing.T) {
	conn, err := testPool.Get()
	if err != nil {
		t.Errorf("Put, get error: %s", err)
	}

	testPool.Put(conn)

	if testPool.CurrentCapacity() != 1 {
		t.Errorf("Put error. Expecting %d, got %d",
			1, testPool.CurrentCapacity())
	}
}

func TestPool_MaximumCapacity(t *testing.T) {
	// Create new pool to test it
	p, _ := newPool()
	if p.MaximumCapacity() != MaximumCap {
		t.Errorf("MaximumCapacity error. Expecting %d, got %d",
			MaximumCap, testPool.CurrentCapacity())
	}
}

func TestPool_UsedCapacity(t *testing.T) {
	// Create new pool to test it
	p, _ := newPool()
	if p.CurrentCapacity() != InitialCap {
		t.Errorf("InitialCap error. Expecting %d, got %d",
			InitialCap, p.CurrentCapacity())
	}
}

func TestPool_Close(t *testing.T) {
	testPool.Close()

	conn, _ := testPool.Get()
	if conn != nil {
		t.Errorf("Close error, conn should be nil")
	}

	testPool.Put(conn)

	if testPool.CurrentCapacity() != 0 {
		t.Errorf("Close error used capacity. Expecting 0, got %d", testPool.MaximumCapacity())
	}

	if testPool.MaximumCapacity() != 0 {
		t.Errorf("Close error max capacity. Expecting 0, got %d", testPool.MaximumCapacity())
	}
}

func TestPoolConcurrent(t *testing.T) {
	p, _ := newPool()
	pipe := make(chan net.Conn, 0)

	go func() {
		p.Close()
	}()

	for i := 0; i < MaximumCap; i++ {
		go func() {
			conn, _ := p.Get()

			pipe <- conn
		}()

		go func() {
			conn := <-pipe
			p.Put(conn)
		}()
	}
}

func TestPoolConcurrent2(t *testing.T) {
	p, _ := newPool()

	for i := 0; i < MaximumCap; i++ {
		conn, _ := p.factory()
		p.Put(conn)
	}

	for i := 0; i < MaximumCap; i++ {
		go func() {
			p.Get()
		}()
	}

	p.Close()
}

func newPool() (*Pool, error) { return New(InitialCap, MaximumCap, factory) }

func simpleTCPServer() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		buffer := make([]byte, 256)
		conn.Read(buffer)
	}
}
