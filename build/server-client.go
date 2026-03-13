package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/armon/go-socks5"
	"github.com/hashicorp/yamux"
)

var (
	currentSession *yamux.Session
	currentPeer    string
	mu             sync.RWMutex
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("usage: ./proxy-server reverse_port socks_port")
		return
	}

	reversePort := os.Args[1]
	socksPort := os.Args[2]

	go reverseListener(reversePort)
	go startSocks(socksPort)

	select {}
}

func reverseListener(port string) {

	addr := ":" + port

	log.Println("reverse listener on", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	for {

		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		session, err := yamux.Server(conn, nil)
		if err != nil {
			conn.Close()
			continue
		}

		if !trySetSession(session, conn.RemoteAddr().String()) {
			peer := conn.RemoteAddr().String()
			active := getCurrentPeer()
			log.Printf("rejecting mobile client %s: session already active (current=%s)", peer, active)
			session.Close()
			conn.Close()
			continue
		}

		log.Println("mobile client connected")

		go watchSession(session)
	}
}

func startSocks(port string) {

	addr := "127.0.0.1:" + port

	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {

			mu.RLock()
			session := currentSession
			mu.RUnlock()

			if session == nil || session.IsClosed() {
				return nil, fmt.Errorf("no mobile connection")
			}

			stream, err := session.Open()
			if err != nil {
				clearSessionIfCurrent(session)
				return nil, err
			}

			_, err = stream.Write([]byte(addr + "\n"))
			if err != nil {
				stream.Close()
				clearSessionIfCurrent(session)
				return nil, err
			}

			return stream, nil
		},
	}

	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("SOCKS5 listening on", addr)

	server.ListenAndServe("tcp", addr)
}

func trySetSession(session *yamux.Session, peer string) bool {
	mu.Lock()
	defer mu.Unlock()

	if currentSession != nil && !currentSession.IsClosed() {
		return false
	}

	currentSession = session
	currentPeer = peer
	return true
}

func clearSessionIfCurrent(session *yamux.Session) {
	mu.Lock()
	if currentSession == session {
		currentSession = nil
		currentPeer = ""
	}
	mu.Unlock()
}

func getCurrentPeer() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentPeer
}

func watchSession(session *yamux.Session) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if session.IsClosed() {
			clearSessionIfCurrent(session)
			log.Println("mobile client disconnected")
			return
		}
	}
}
