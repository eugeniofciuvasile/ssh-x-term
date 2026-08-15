package tunnel

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// PortCheckResult represents the outcome of checking port availability
type PortCheckResult struct {
	Host      string
	Port      int
	Available bool
	Error     error
	Message   string
}

// CheckLocalPort tests if a local TCP port is free to bind on the given host/interface
func CheckLocalPort(host string, port int) PortCheckResult {
	if port <= 0 || port > 65535 {
		return PortCheckResult{
			Host:      host,
			Port:      port,
			Available: false,
			Error:     fmt.Errorf("invalid port number: %d (must be 1-65535)", port),
			Message:   "Invalid port number",
		}
	}

	if host == "" {
		host = "127.0.0.1"
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return PortCheckResult{
			Host:      host,
			Port:      port,
			Available: false,
			Error:     err,
			Message:   fmt.Sprintf("Port %d is already in use", port),
		}
	}

	// Successfully bound, close it immediately
	_ = listener.Close()

	return PortCheckResult{
		Host:      host,
		Port:      port,
		Available: true,
		Message:   fmt.Sprintf("Port %d is available", port),
	}
}

// FindNextAvailablePort searches for the nearest free port starting at startPort
func FindNextAvailablePort(host string, startPort int) int {
	if startPort <= 0 {
		startPort = 1024
	}
	if startPort > 65500 {
		startPort = 1024
	}

	for p := startPort; p <= 65535; p++ {
		res := CheckLocalPort(host, p)
		if res.Available {
			return p
		}
	}
	return 0
}

// CheckRemotePort tests if a remote target port is reachable
func CheckRemotePort(host string, port int, timeout time.Duration) bool {
	if port <= 0 || port > 65535 || host == "" {
		return false
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
