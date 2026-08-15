package tunnel

import (
	"net"
	"testing"
)

func TestCheckLocalPort(t *testing.T) {
	// 1. Check an ephemeral free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind ephemeral port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close() // Now it should be free

	res := CheckLocalPort("127.0.0.1", port)
	if !res.Available {
		t.Errorf("Expected port %d to be available, got: %v", port, res.Error)
	}

	// 2. Bind port and verify CheckLocalPort reports it as unavailable
	l2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind port: %v", err)
	}
	defer l2.Close()
	busyPort := l2.Addr().(*net.TCPAddr).Port

	resBusy := CheckLocalPort("127.0.0.1", busyPort)
	if resBusy.Available {
		t.Errorf("Expected port %d to be in use, but reported available", busyPort)
	}
}

func TestFindNextAvailablePort(t *testing.T) {
	port := FindNextAvailablePort("127.0.0.1", 20000)
	if port < 20000 || port > 65535 {
		t.Errorf("Unexpected available port found: %d", port)
	}
}
