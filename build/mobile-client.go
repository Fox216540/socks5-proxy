package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
)

const (
	reconnectDelay    = 5 * time.Second
	heartbeatInterval = 20 * time.Second
)

var (
	totalUp   uint64
	totalDown uint64
)

func main() 
func main() {

	if len(os.Args) < 2 {
		fmt.Println("usage: ./mobile-client ip:port")
		return
	}

	serverAddr := os.Args[1]

	for {
		err := run(serverAddr)
		log.Println("connection ended:", err)
		time.Sleep(reconnectDelay)
	}
}

func run(serverAddr string) error {

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}

	log.Println("connected to", serverAddr)

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}

	go heartbeat(session)

	for {
		stream, err := session.Accept()
		if err != nil {
			return err
		}

		go handle(stream)
	}
}

func heartbeat(session *yamux.Session) {

	for {
		time.Sleep(heartbeatInterval)

		stream, err := session.Open()
		if err != nil {
			return
		}

		stream.Write([]byte("PING\n"))
		stream.Close()
	}
}

func handle(stream net.Conn) {

	reader := bufio.NewReader(stream)

	addr, err := reader.ReadString('\n')
	if err != nil {
		stream.Close()
		return
	}

	addr = strings.TrimSpace(addr)

	if addr == "PING" {
		stream.Close()
		return
	}

	log.Printf("proxy request: %s -> %s", stream.RemoteAddr(), addr)

	target, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("dial target failed: %s: %v", addr, err)
		stream.Close()
		return
	}

	type copyResult struct {
		n   int64
		err error
	}

	upCh := make(chan copyResult, 1)
	go func() {
		n, err := io.Copy(target, reader)
		target.Close()
		upCh <- copyResult{n: n, err: err}
	}()

	downN, downErr := io.Copy(stream, target)
	stream.Close()
	up := <-upCh

	atomic.AddUint64(&totalUp, uint64(up.n))
	atomic.AddUint64(&totalDown, uint64(downN))
	totalUpNow := atomic.LoadUint64(&totalUp)
	totalDownNow := atomic.LoadUint64(&totalDown)
	log.Printf("proxy done: %s -> %s (up=%d bytes err=%v, down=%d bytes err=%v, total_up=%d bytes, total_down=%d bytes)", stream.RemoteAddr(), addr, up.n, up.err, downN, downErr, totalUpNow, totalDownNow)
}
