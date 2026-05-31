package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-socks5"
	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"
)

type Proxy struct {
	Port       int
	TargetPort int
	Listener   net.Listener
}

var (
	proxies = map[int]*Proxy{}
	mu      sync.Mutex
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	login := os.Getenv("LOGIN")
	password := os.Getenv("PASSWORD")

	if login == "" || password == "" {
		log.Fatal("LOGIN or PASSWORD missing")
	}

	startMainProxy(login, password)

	go watchReverse(login, password)

	select {}
}

func startMainProxy(login, password string) {
	conf := &socks5.Config{
		Credentials: socks5.StaticCredentials{
			login: password,
		},
	}

	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("main socks5 :50000")

		err := server.ListenAndServe("tcp", ":50000")
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func watchReverse(login, password string) {
	for {
		syncProxies(login, password)

		time.Sleep(5 * time.Second)
	}
}

func syncProxies(login, password string) {
	file, err := os.Open("reverse.txt")
	if err != nil {
		log.Println(err)
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	current := map[int]int{}

	forwardPort := 50001

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var target int

		_, err := fmt.Sscanf(line, "%d", &target)
		if err != nil {
			continue
		}

		current[forwardPort] = target

		forwardPort++
	}

	mu.Lock()
	defer mu.Unlock()

	for port, target := range current {
		if _, ok := proxies[port]; ok {
			continue
		}

		go startReverseProxy(port, target, login, password)
	}
}

func startReverseProxy(port, target int, login, password string) {
	auth := socks5.StaticCredentials{
		login: password,
	}

	conf := &socks5.Config{
		Credentials: auth,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialViaSocks(target, network, addr)
		},
		Resolver: socks5.DNSResolver{},
	}

	server, err := socks5.New(conf)
	if err != nil {
		log.Println(err)
		return
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("listen %d failed: %v", port, err)
		return
	}

	proxies[port] = &Proxy{
		Port:       port,
		TargetPort: target,
		Listener:   ln,
	}

	log.Printf("%d -> 127.0.0.1:%d", port, target)

	err = server.Serve(ln)
	if err != nil {
		log.Printf("serve %d failed: %v", port, err)
	}
}

func dialViaSocks(target int, network, addr string) (net.Conn, error) {
	dialer, err := proxy.SOCKS5(
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", target),
		nil,
		proxy.Direct,
	)
	if err != nil {
		return nil, err
	}

	return dialer.Dial(network, addr)
}
