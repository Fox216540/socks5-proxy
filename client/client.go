package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
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

	loggerMu sync.RWMutex
	logger   Logger

	runnerMu sync.Mutex
	cancel   context.CancelFunc
	running  bool
)

// Logger используется для передачи логов из Go в Android UI через gomobile.
type Logger interface {
	OnLog(line string)
}

// SetLogger устанавливает получатель логов.
func SetLogger(l Logger) {
	loggerMu.Lock()
	logger = l
	loggerMu.Unlock()
}

// ClearLogger сбрасывает получатель логов.
func ClearLogger() {
	SetLogger(nil)
}

func emitLogf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	log.Println(line)

	loggerMu.RLock()
	l := logger
	loggerMu.RUnlock()
	if l == nil {
		return
	}

	defer func() {
		_ = recover()
	}()
	l.OnLog(line)
}

// Start запускает reverse SOCKS клиент в фоне.
// Если клиент уже запущен, повторный вызов игнорируется.
func Start(addr string) {
	StartWithTLS(addr, false)
}

// StartWithTLS запускает клиент с опциональным TLS.
// Если клиент уже запущен, повторный вызов игнорируется.
func StartWithTLS(addr string, useTLS bool) {
	runnerMu.Lock()
	if running {
		runnerMu.Unlock()
		emitLogf("start ignored: client already running")
		return
	}

	ctx, c := context.WithCancel(context.Background())
	cancel = c
	running = true
	runnerMu.Unlock()

	go func() {
		defer func() {
			runnerMu.Lock()
			running = false
			cancel = nil
			runnerMu.Unlock()
		}()

		emitLogf("client worker started (addr=%s tls=%t)", addr, useTLS)
		runForever(ctx, addr, useTLS)
		emitLogf("client worker stopped")
	}()
}

// Stop останавливает фонового клиента.
func Stop() {
	runnerMu.Lock()
	c := cancel
	runnerMu.Unlock()
	if c != nil {
		c()
		emitLogf("stop requested")
	} else {
		emitLogf("stop ignored: client not running")
	}
}

// RunForever используется CLI-клиентом (блокирующий режим).
func RunForever(addr string) {
	runForever(context.Background(), addr, false)
}

func runForever(ctx context.Context, addr string, useTLS bool) {
	for {
		if ctx.Err() != nil {
			return
		}

		err := runOnce(ctx, addr, useTLS)
		if err != nil && ctx.Err() == nil {
			emitLogf("connection ended: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func runOnce(ctx context.Context, addr string, useTLS bool) error {
	var (
		conn net.Conn
		err  error
	)

	if useTLS {
		conn, err = tls.Dial("tcp", addr, &tls.Config{MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		emitLogf("connect failed: %v", err)
		return err
	}
	defer conn.Close()

	emitLogf("connected to %s", addr)

	session, err := yamux.Client(conn, nil)
	if err != nil {
		emitLogf("yamux client init failed: %v", err)
		return err
	}
	defer session.Close()

	go heartbeat(ctx, session)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		stream, err := session.Accept()
		if err != nil {
			emitLogf("session accept failed: %v", err)
			return err
		}

		go handle(stream)
	}
}

func heartbeat(ctx context.Context, session *yamux.Session) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		stream, err := session.Open()
		if err != nil {
			emitLogf("heartbeat stream open failed: %v", err)
			return
		}

		_, _ = stream.Write([]byte("PING\n"))
		_ = stream.Close()
	}
}

func handle(stream net.Conn) {
	reader := bufio.NewReader(stream)

	addr, err := reader.ReadString('\n')
	if err != nil {
		_ = stream.Close()
		return
	}

	addr = strings.TrimSpace(addr)
	if addr == "PING" {
		_ = stream.Close()
		return
	}

	emitLogf("proxy request: %s -> %s", stream.RemoteAddr(), addr)

	target, err := net.Dial("tcp", addr)
	if err != nil {
		emitLogf("dial target failed: %s: %v", addr, err)
		_ = stream.Close()
		return
	}

	type copyResult struct {
		n   int64
		err error
	}

	upCh := make(chan copyResult, 1)
	go func() {
		n, err := io.Copy(target, reader)
		_ = target.Close()
		upCh <- copyResult{n: n, err: err}
	}()

	downN, downErr := io.Copy(stream, target)
	_ = stream.Close()
	up := <-upCh

	atomic.AddUint64(&totalUp, uint64(up.n))
	atomic.AddUint64(&totalDown, uint64(downN))
	totalUpNow := atomic.LoadUint64(&totalUp)
	totalDownNow := atomic.LoadUint64(&totalDown)
	emitLogf(
		"proxy done: %s -> %s (up=%d bytes err=%v, down=%d bytes err=%v, total_up=%d bytes, total_down=%d bytes)",
		stream.RemoteAddr(),
		addr,
		up.n,
		up.err,
		downN,
		downErr,
		totalUpNow,
		totalDownNow,
	)
}
