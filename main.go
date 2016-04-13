package main

import (
	"fmt"
	"log"
	"net"
	//	"testing"
	"pool"
	"time"
)

const (
	SERVER_HOST  = "192.168.189.129:8888"
	INIT_CONNECT = 5
	MAX_CONNECT  = 10
)

var (
	RUN_FLAG = true
)

type TaskProcess interface {
	Process()
}

type TcpTask struct {
	Cmd   string
	Index int
}

func (task *TcpTask) Process(process_id int) {
	log.Println(process_id, "TcpTask Process", task.Cmd, task.Index)
}

type TimeTask struct {
	Index int
}

func (task *TimeTask) Process() {
	log.Println("TimeTask Process", task.Index)
}

func CheckReconnect(pool_connect pool.Pool, reconn_chan chan int) {
	need_reconnect := 0
	for RUN_FLAG {
		select {
		case count := <-reconn_chan:
			log.Println("Start Reconnect:", count)
			for i := 0; i < count; i += 1 {
				if err := pool_connect.Reconnect(); err != nil {
					need_reconnect += 1
				}
			}
		case <-time.After(5 * time.Second):
			log.Println("CheckReconnect", need_reconnect)
			if need_reconnect > 0 {
				reconn_chan <- need_reconnect
				need_reconnect = 0
			}

		}
	}
}

func OnProcessTcp(p pool.Pool, gb_data *GlobalData) error {
	log.Println("Poollen:", p.Len())
	pc, err := p.Get()
	if err == nil {
		icount, err1 := pc.Write([]byte("1111111111"))
		if err1 != nil && icount == 0 {
			pc.MarkUnusable()
			gb_data.ReconnectChan <- 1
			log.Println("OnProcessTcp Lost Connect")
		}
		pc.Close()
	}
	return err
}

func ProcessTcp(process_id int, p pool.Pool, gb_data *GlobalData) {
	log.Println(process_id, "Start ProcessTcp", p, gb_data)
	time.Sleep(time.Second)
	for RUN_FLAG {
		select {
		case task := <-gb_data.TcpChan:
			task.Process(process_id)
			if err := OnProcessTcp(p, gb_data); err != nil {
				break
			}

		case <-time.After(time.Second):
			//			log.Println(process_id, "time_out, ProcessTcp")
			if err := OnProcessTcp(p, gb_data); err != nil {
				break
			}
		}
	}
	log.Println(process_id, "Stop ProcessTcp", p, gb_data)
}

func CreateConnect() (net.Conn, error) {
	conn, err := net.Dial("tcp", SERVER_HOST)
	// for err != nil {
	// 	conn, err = net.Dial("tcp", SERVER_HOST)
	// 	log.Println("CreateConnectFactory Error reconnect", err)

	// }
	return conn, err
}

type GlobalData struct {
	TcpChan       chan TcpTask
	TimeChan      chan TimeTask
	ReconnectChan chan int
}

func main() {
	pool_connect, start_err := pool.NewChannelPool(INIT_CONNECT, MAX_CONNECT, CreateConnect)
	for start_err != nil {
		log.Println("NewChannelPool err", start_err)
		time.Sleep(time.Second * 5)
		pool_connect, start_err = pool.NewChannelPool(INIT_CONNECT, MAX_CONNECT, CreateConnect)
	}
	defer pool_connect.Close()
	gb_data := new(GlobalData)
	gb_data.TcpChan = make(chan TcpTask, 20000)
	gb_data.TimeChan = make(chan TimeTask, 20000)
	gb_data.ReconnectChan = make(chan int, MAX_CONNECT)

	go CheckReconnect(pool_connect, gb_data.ReconnectChan)
	for i := 0; i < MAX_CONNECT; i += 1 {
		go ProcessTcp(i, pool_connect, gb_data)
	}

	for i := 0; i < 100; i += 1 {
		var task TcpTask
		task.Cmd = fmt.Sprintf("%d", i)
		task.Index = i
		gb_data.TcpChan <- task
	}
	log.Println("AddTaskDown")
	for RUN_FLAG {
		select {
		case task := <-gb_data.TimeChan:
			task.Process()
		}

	}
}
