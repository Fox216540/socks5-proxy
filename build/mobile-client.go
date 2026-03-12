package main

import (
        "bufio"
        "fmt"
        "io"
        "log"
        "net"
        "os"
        "strings"
        "time"

        "github.com/hashicorp/yamux"
)

const (
        reconnectDelay    = 5 * time.Second
        heartbeatInterval = 20 * time.Second
)

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

        target, err := net.Dial("tcp", addr)
        if err != nil {
                stream.Close()
                return
        }

        go func() {
                io.Copy(target, reader)
                target.Close()
        }()

        io.Copy(stream, target)
        stream.Close()
}
