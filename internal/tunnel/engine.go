package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	sshclient "github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
	"golang.org/x/crypto/ssh"
)

// TunnelStatus represents current operational state
type TunnelStatus string

const (
	StatusStopped TunnelStatus = "stopped"
	StatusActive  TunnelStatus = "active"
	StatusError   TunnelStatus = "error"
)

// RuntimeInfo contains real-time status and traffic statistics of a tunnel
type RuntimeInfo struct {
	Config      config.TunnelConfig
	Status      TunnelStatus
	ActiveConns int64
	BytesIn     int64
	BytesOut    int64
	StartedAt   time.Time
	LastError   string
}

// DataDirection indicates the direction of traffic flow
type DataDirection string

const (
	DirLocalToRemote DataDirection = "LOCAL -> REMOTE"
	DirRemoteToLocal DataDirection = "REMOTE -> LOCAL"
)

// TrafficPacket stores a captured live packet sample for inspection
type TrafficPacket struct {
	Timestamp time.Time
	Direction DataDirection
	Length    int
	Data      []byte
}

// activeTunnel holds runtime handles for a running tunnel
type activeTunnel struct {
	config      config.TunnelConfig
	connConfig  config.SSHConnection
	status      TunnelStatus
	sshClient   *sshclient.Client
	listener    net.Listener
	cancel      context.CancelFunc
	activeConns int64
	bytesIn     int64
	bytesOut    int64
	startedAt   time.Time
	lastError   string
	packets     []TrafficPacket
	packetMu    sync.RWMutex
	mu          sync.RWMutex
}

// Engine manages the lifecycle of all SSH tunnels
type Engine struct {
	tunnels map[string]*activeTunnel
	mu      sync.RWMutex
}

var globalEngine *Engine
var once sync.Once

// GetEngine returns the singleton tunnel engine
func GetEngine() *Engine {
	once.Do(func() {
		globalEngine = &Engine{
			tunnels: make(map[string]*activeTunnel),
		}
	})
	return globalEngine
}

// StartTunnel starts forwarding for the given tunnel config
func (e *Engine) StartTunnel(connConfig config.SSHConnection, tun config.TunnelConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// If already running, return
	if active, exists := e.tunnels[tun.ID]; exists {
		active.mu.RLock()
		status := active.status
		active.mu.RUnlock()
		if status == StatusActive {
			return nil
		}
	}

	// 1. Port conflict check for Local and Dynamic tunnels
	if tun.Type == config.TunnelTypeLocal || tun.Type == config.TunnelTypeDynamic {
		bindHost := tun.BindHost
		if bindHost == "" {
			bindHost = "127.0.0.1"
		}
		check := CheckLocalPort(bindHost, tun.BindPort)
		if !check.Available {
			err := fmt.Errorf("local port %d is already in use", tun.BindPort)
			e.tunnels[tun.ID] = &activeTunnel{
				config:     tun,
				connConfig: connConfig,
				status:     StatusError,
				lastError:  err.Error(),
			}
			return err
		}
	}

	// 2. Connect SSH client
	client, err := sshclient.NewClient(connConfig)
	if err != nil {
		e.tunnels[tun.ID] = &activeTunnel{
			config:     tun,
			connConfig: connConfig,
			status:     StatusError,
			lastError:  fmt.Sprintf("SSH connection failed: %v", err),
		}
		return fmt.Errorf("failed to connect SSH for tunnel %s: %w", tun.Name, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	at := &activeTunnel{
		config:     tun,
		connConfig: connConfig,
		status:     StatusActive,
		sshClient:  client,
		cancel:     cancel,
		startedAt:  time.Now(),
	}

	e.tunnels[tun.ID] = at

	// 3. Start listener depending on tunnel type
	switch tun.Type {
	case config.TunnelTypeLocal:
		go e.runLocalForward(ctx, at)
	case config.TunnelTypeRemote:
		go e.runRemoteForward(ctx, at)
	case config.TunnelTypeDynamic:
		go e.runDynamicForward(ctx, at)
	default:
		cancel()
		_ = client.Close()
		at.status = StatusError
		at.lastError = "unknown tunnel type"
		return errors.New("unknown tunnel type")
	}

	return nil
}

// StopTunnel terminates an active tunnel
func (e *Engine) StopTunnel(tunnelID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	at, exists := e.tunnels[tunnelID]
	if !exists {
		return nil
	}

	at.mu.Lock()
	defer at.mu.Unlock()

	if at.cancel != nil {
		at.cancel()
	}
	if at.listener != nil {
		_ = at.listener.Close()
	}
	if at.sshClient != nil {
		_ = at.sshClient.Close()
	}

	at.status = StatusStopped
	at.activeConns = 0
	delete(e.tunnels, tunnelID)
	return nil
}

// ToggleTunnel toggles a tunnel on/off
func (e *Engine) ToggleTunnel(connConfig config.SSHConnection, tun config.TunnelConfig) (bool, error) {
	info := e.GetRuntimeInfo(tun.ID)
	if info.Status == StatusActive {
		err := e.StopTunnel(tun.ID)
		return false, err
	}
	err := e.StartTunnel(connConfig, tun)
	return err == nil, err
}

// GetRuntimeInfo retrieves status and metrics for a given tunnel
func (e *Engine) GetRuntimeInfo(tunnelID string) RuntimeInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	at, exists := e.tunnels[tunnelID]
	if !exists {
		return RuntimeInfo{
			Status: StatusStopped,
		}
	}

	at.mu.RLock()
	defer at.mu.RUnlock()

	return RuntimeInfo{
		Config:      at.config,
		Status:      at.status,
		ActiveConns: atomic.LoadInt64(&at.activeConns),
		BytesIn:     atomic.LoadInt64(&at.bytesIn),
		BytesOut:    atomic.LoadInt64(&at.bytesOut),
		StartedAt:   at.startedAt,
		LastError:   at.lastError,
	}
}

// RecordPacket stores a sample of a live packet into the tunnel's circular buffer
func (at *activeTunnel) RecordPacket(dir DataDirection, data []byte) {
	at.packetMu.Lock()
	defer at.packetMu.Unlock()

	sampleLen := len(data)
	if sampleLen > 512 {
		sampleLen = 512
	}
	sample := make([]byte, sampleLen)
	copy(sample, data[:sampleLen])

	pkt := TrafficPacket{
		Timestamp: time.Now(),
		Direction: dir,
		Length:    len(data),
		Data:      sample,
	}

	if len(at.packets) >= 150 {
		at.packets = append(at.packets[1:], pkt)
	} else {
		at.packets = append(at.packets, pkt)
	}
}

// GetRecentPackets returns a copy of recently captured packets for inspection
func (e *Engine) GetRecentPackets(tunnelID string) []TrafficPacket {
	e.mu.RLock()
	at, exists := e.tunnels[tunnelID]
	e.mu.RUnlock()

	if !exists {
		return nil
	}

	at.packetMu.RLock()
	defer at.packetMu.RUnlock()

	res := make([]TrafficPacket, len(at.packets))
	copy(res, at.packets)
	return res
}

// ClearPackets empties the packet history buffer for a tunnel
func (e *Engine) ClearPackets(tunnelID string) {
	e.mu.RLock()
	at, exists := e.tunnels[tunnelID]
	e.mu.RUnlock()

	if !exists {
		return
	}

	at.packetMu.Lock()
	at.packets = nil
	at.packetMu.Unlock()
}

// ListAllRuntimes returns runtime info for all tracked tunnels
func (e *Engine) ListAllRuntimes() map[string]RuntimeInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	res := make(map[string]RuntimeInfo)
	for id, at := range e.tunnels {
		at.mu.RLock()
		res[id] = RuntimeInfo{
			Config:      at.config,
			Status:      at.status,
			ActiveConns: atomic.LoadInt64(&at.activeConns),
			BytesIn:     atomic.LoadInt64(&at.bytesIn),
			BytesOut:    atomic.LoadInt64(&at.bytesOut),
			StartedAt:   at.startedAt,
			LastError:   at.lastError,
		}
		at.mu.RUnlock()
	}
	return res
}

// StopAll terminates all active tunnels
func (e *Engine) StopAll() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, at := range e.tunnels {
		at.mu.Lock()
		if at.cancel != nil {
			at.cancel()
		}
		if at.listener != nil {
			_ = at.listener.Close()
		}
		if at.sshClient != nil {
			_ = at.sshClient.Close()
		}
		at.status = StatusStopped
		at.mu.Unlock()
	}
	e.tunnels = make(map[string]*activeTunnel)
}

// runLocalForward handles Local Port Forwarding (-L)
func (e *Engine) runLocalForward(ctx context.Context, at *activeTunnel) {
	bindHost := at.config.BindHost
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	localAddr := net.JoinHostPort(bindHost, strconv.Itoa(at.config.BindPort))
	targetHost := at.config.TargetHost
	if targetHost == "" {
		targetHost = "127.0.0.1"
	}
	targetAddr := net.JoinHostPort(targetHost, strconv.Itoa(at.config.TargetPort))

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		at.mu.Lock()
		at.status = StatusError
		at.lastError = fmt.Sprintf("failed to bind %s: %v", localAddr, err)
		at.mu.Unlock()
		return
	}

	at.mu.Lock()
	at.listener = listener
	at.mu.Unlock()

	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[Tunnel %s] Accept error: %v", at.config.Name, err)
				return
			}
		}

		go func(c net.Conn) {
			atomic.AddInt64(&at.activeConns, 1)
			defer atomic.AddInt64(&at.activeConns, -1)
			defer c.Close()

			remoteConn, err := at.sshClient.SSHClient().Dial("tcp", targetAddr)
			if err != nil {
				log.Printf("[Tunnel %s] Dial to %s failed: %v", at.config.Name, targetAddr, err)
				at.mu.Lock()
				at.lastError = fmt.Sprintf("Dial to %s failed: %v", targetAddr, err)
				at.mu.Unlock()
				return
			}
			defer remoteConn.Close()

			e.pipeBidirectional(c, remoteConn, at)
		}(clientConn)
	}
}

// runRemoteForward handles Remote Port Forwarding (-R)
func (e *Engine) runRemoteForward(ctx context.Context, at *activeTunnel) {
	bindHost := at.config.BindHost
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}
	remoteAddr := net.JoinHostPort(bindHost, strconv.Itoa(at.config.BindPort))
	targetHost := at.config.TargetHost
	if targetHost == "" {
		targetHost = "127.0.0.1"
	}
	targetAddr := net.JoinHostPort(targetHost, strconv.Itoa(at.config.TargetPort))

	listener, err := at.sshClient.SSHClient().Listen("tcp", remoteAddr)
	if err != nil {
		at.mu.Lock()
		at.status = StatusError
		at.lastError = fmt.Sprintf("failed remote listen on %s: %v", remoteAddr, err)
		at.mu.Unlock()
		return
	}

	at.mu.Lock()
	at.listener = listener
	at.mu.Unlock()

	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[Remote Tunnel %s] Accept error: %v", at.config.Name, err)
				return
			}
		}

		go func(rc net.Conn) {
			atomic.AddInt64(&at.activeConns, 1)
			defer atomic.AddInt64(&at.activeConns, -1)
			defer rc.Close()

			localConn, err := net.Dial("tcp", targetAddr)
			if err != nil {
				log.Printf("[Remote Tunnel %s] Local dial to %s failed: %v", at.config.Name, targetAddr, err)
				return
			}
			defer localConn.Close()

			e.pipeBidirectional(rc, localConn, at)
		}(remoteConn)
	}
}

// runDynamicForward handles Dynamic SOCKS5 Forwarding (-D)
func (e *Engine) runDynamicForward(ctx context.Context, at *activeTunnel) {
	bindHost := at.config.BindHost
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	localAddr := net.JoinHostPort(bindHost, strconv.Itoa(at.config.BindPort))

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		at.mu.Lock()
		at.status = StatusError
		at.lastError = fmt.Sprintf("failed to bind SOCKS5 on %s: %v", localAddr, err)
		at.mu.Unlock()
		return
	}

	at.mu.Lock()
	at.listener = listener
	at.mu.Unlock()

	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("[Dynamic Tunnel %s] Accept error: %v", at.config.Name, err)
				return
			}
		}

		go func(c net.Conn) {
			atomic.AddInt64(&at.activeConns, 1)
			defer atomic.AddInt64(&at.activeConns, -1)
			defer c.Close()

			targetConn, err := HandleSOCKS5Conn(c, at.sshClient.SSHClient())
			if err != nil {
				log.Printf("[Dynamic Tunnel %s] SOCKS5 handshake error: %v", at.config.Name, err)
				return
			}
			defer targetConn.Close()

			e.pipeBidirectional(c, targetConn, at)
		}(clientConn)
	}
}

// copyStream transfers data from src to dst, increments counter, and records samples
func (e *Engine) copyStream(src io.Reader, dst io.Writer, counter *int64, dir DataDirection, at *activeTunnel) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			atomic.AddInt64(counter, int64(n))
			if at != nil {
				at.RecordPacket(dir, buf[:n])
			}
			if _, werr := dst.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
}

// pipeBidirectional proxies data between two connections and tracks byte metrics
func (e *Engine) pipeBidirectional(connA, connB net.Conn, at *activeTunnel) {
	var wg sync.WaitGroup
	wg.Add(2)

	// connA -> connB (Local -> Remote)
	go func() {
		defer wg.Done()
		e.copyStream(connA, connB, &at.bytesIn, DirLocalToRemote, at)
		if tc, ok := connB.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else if sc, ok := connB.(ssh.Channel); ok {
			_ = sc.CloseWrite()
		}
	}()

	// connB -> connA (Remote -> Local)
	go func() {
		defer wg.Done()
		e.copyStream(connB, connA, &at.bytesOut, DirRemoteToLocal, at)
		if tc, ok := connA.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	wg.Wait()
}
