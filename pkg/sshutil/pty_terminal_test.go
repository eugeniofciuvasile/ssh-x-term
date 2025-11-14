package sshutil_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
)

// TestScrollbackBuffer tests the scrollback buffer functionality
func TestScrollbackBuffer(t *testing.T) {
	// Create a scrollback buffer with a maximum of 5 lines
	sb := sshutil.NewScrollbackBuffer(5)

	// Add some lines
	sb.AddLine([]byte("Line 1"))
	sb.AddLine([]byte("Line 2"))
	sb.AddLine([]byte("Line 3"))

	// Get lines
	lines := sb.GetLines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Verify content
	if string(lines[0]) != "Line 1" {
		t.Errorf("Expected 'Line 1', got '%s'", string(lines[0]))
	}

	// Add more lines to exceed max
	sb.AddLine([]byte("Line 4"))
	sb.AddLine([]byte("Line 5"))
	sb.AddLine([]byte("Line 6"))
	sb.AddLine([]byte("Line 7"))

	// Should only have 5 lines (max capacity)
	lines = sb.GetLines()
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines (max capacity), got %d", len(lines))
	}

	// First line should be "Line 3" (oldest kept)
	if string(lines[0]) != "Line 3" {
		t.Errorf("Expected 'Line 3' as first line, got '%s'", string(lines[0]))
	}

	// Clear the buffer
	sb.Clear()
	lines = sb.GetLines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines after clear, got %d", len(lines))
	}
}

// TestScrollbackBufferConcurrency tests concurrent access to the scrollback buffer
func TestScrollbackBufferConcurrency(t *testing.T) {
	sb := sshutil.NewScrollbackBuffer(1000)
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			sb.AddLine([]byte("Test line"))
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = sb.GetLines()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify some lines were added
	lines := sb.GetLines()
	if len(lines) == 0 {
		t.Error("Expected some lines in buffer")
	}
}

// TestPTYTerminalOptions tests the terminal options configuration
func TestPTYTerminalOptions(t *testing.T) {
	// Create mock I/O streams
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Create terminal with custom options
	opts := &sshutil.PTYTerminalOptions{
		Shell: "/bin/bash",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		ScrollbackLines: 5000,
		EnableMouse:     true,
		Debug:           true,
	}

	terminal, err := sshutil.NewPTYTerminal(stdin, stdout, stderr, opts)
	if err != nil {
		t.Skipf("Skipping test: %v (likely not in a terminal environment)", err)
		return
	}
	defer terminal.Close()

	// Verify environment was set
	env := terminal.GetEnvironment()
	if env["TEST_VAR"] != "test_value" {
		t.Errorf("Expected TEST_VAR=test_value, got %s", env["TEST_VAR"])
	}

	// Verify scrollback buffer
	scrollback := terminal.GetScrollback()
	if scrollback == nil {
		t.Error("Expected non-nil scrollback buffer")
	}

	// Verify terminal size
	width, height := terminal.GetSize()
	if width == 0 || height == 0 {
		t.Logf("Terminal size: %dx%d (may be 0 in test environment)", width, height)
	}
}

// TestPTYTerminalEnvironment tests environment variable management
func TestPTYTerminalEnvironment(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	opts := &sshutil.PTYTerminalOptions{
		Environment: map[string]string{
			"VAR1": "value1",
		},
	}

	terminal, err := sshutil.NewPTYTerminal(stdin, stdout, stderr, opts)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}
	defer terminal.Close()

	// Set additional environment variable
	terminal.SetEnvironment("VAR2", "value2")

	// Get environment and verify
	env := terminal.GetEnvironment()
	if env["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", env["VAR1"])
	}
	if env["VAR2"] != "value2" {
		t.Errorf("Expected VAR2=value2, got %s", env["VAR2"])
	}

	// Verify default environment variables are set
	if env["TERM"] == "" {
		t.Error("Expected TERM to be set")
	}
	if env["PATH"] == "" {
		t.Error("Expected PATH to be set")
	}
}

// TestGetTerminalSize tests the terminal size function
func TestGetTerminalSize(t *testing.T) {
	width, height, err := sshutil.GetTerminalSize()
	if err != nil {
		t.Skipf("Skipping test: %v (not in a terminal)", err)
		return
	}

	if width <= 0 || height <= 0 {
		t.Errorf("Invalid terminal size: %dx%d", width, height)
	}

	t.Logf("Terminal size: %dx%d", width, height)
}

// MockReader is a test helper that implements io.Reader
type MockReader struct {
	data []byte
	pos  int
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

// TestPTYTerminalWithMockIO tests terminal with mock I/O
func TestPTYTerminalWithMockIO(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &MockReader{data: []byte("test output\n")}
	stderr := &bytes.Buffer{}

	opts := &sshutil.PTYTerminalOptions{
		ScrollbackLines: 100,
		Debug:           true,
	}

	terminal, err := sshutil.NewPTYTerminal(stdin, stdout, stderr, opts)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}

	// Close immediately since we're not actually starting the session
	err = terminal.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
