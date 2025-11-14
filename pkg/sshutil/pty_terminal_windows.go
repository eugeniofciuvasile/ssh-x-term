//go:build windows
// +build windows

package sshutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// PTYTerminal represents a fully functional terminal for Windows with
// scrollback buffer, signal handling, and basic terminal features.
// Note: Windows has limited PTY support compared to Unix systems.
type PTYTerminal struct {
	// I/O streams
	stdin  io.Writer
	stdout io.Reader
	stderr io.Reader

	// Terminal state
	fd       int
	origTerm *term.State
	width    int
	height   int

	// Scrollback buffer
	scrollback *ScrollbackBuffer

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Mouse support (limited on Windows)
	mouseEnabled bool

	// Environment variables
	env map[string]string

	// Logging
	debug bool
}

// ScrollbackBuffer maintains a history of terminal output for scrolling.
type ScrollbackBuffer struct {
	buffer   [][]byte
	maxLines int
	mutex    sync.RWMutex
}

// NewScrollbackBuffer creates a new scrollback buffer with the specified maximum lines.
func NewScrollbackBuffer(maxLines int) *ScrollbackBuffer {
	return &ScrollbackBuffer{
		buffer:   make([][]byte, 0, maxLines),
		maxLines: maxLines,
	}
}

// AddLine adds a line to the scrollback buffer.
func (sb *ScrollbackBuffer) AddLine(line []byte) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	// Make a copy of the line
	lineCopy := make([]byte, len(line))
	copy(lineCopy, line)

	sb.buffer = append(sb.buffer, lineCopy)

	// Trim buffer if it exceeds max lines
	if len(sb.buffer) > sb.maxLines {
		sb.buffer = sb.buffer[1:]
	}
}

// GetLines returns all lines in the scrollback buffer.
func (sb *ScrollbackBuffer) GetLines() [][]byte {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	lines := make([][]byte, len(sb.buffer))
	for i, line := range sb.buffer {
		lines[i] = make([]byte, len(line))
		copy(lines[i], line)
	}
	return lines
}

// Clear clears the scrollback buffer.
func (sb *ScrollbackBuffer) Clear() {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()
	sb.buffer = sb.buffer[:0]
}

// PTYTerminalOptions contains configuration options for PTYTerminal.
type PTYTerminalOptions struct {
	// Shell to launch (defaults to cmd.exe or PowerShell)
	Shell string

	// Environment variables to set
	Environment map[string]string

	// Scrollback buffer size (number of lines, defaults to 10000)
	ScrollbackLines int

	// Enable mouse support (limited on Windows)
	EnableMouse bool

	// Enable debug logging
	Debug bool
}

// NewPTYTerminal creates a new PTY terminal for Windows.
// If opts is nil, default options are used.
func NewPTYTerminal(stdin io.Writer, stdout, stderr io.Reader, opts *PTYTerminalOptions) (*PTYTerminal, error) {
	if opts == nil {
		opts = &PTYTerminalOptions{
			ScrollbackLines: 10000,
			EnableMouse:     false, // Mouse support is limited on Windows
			Debug:           false,
		}
	}

	fd := int(os.Stdin.Fd())

	// Save the original terminal state
	origTerm, err := term.GetState(fd)
	if err != nil {
		log.Printf("Failed to get terminal state: %v", err)
		return nil, fmt.Errorf("failed to get terminal state: %w", err)
	}

	// Get initial terminal size
	width, height, err := term.GetSize(fd)
	if err != nil {
		log.Printf("Failed to get terminal size: %v", err)
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Set up default environment for Windows
	env := make(map[string]string)
	env["TERM"] = "xterm-256color"
	
	// Add PATH if not provided
	if path := os.Getenv("PATH"); path != "" {
		env["PATH"] = path
	}
	
	// Add USERPROFILE (Windows equivalent of HOME)
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		env["USERPROFILE"] = userProfile
		env["HOME"] = userProfile
	}

	// Merge with provided environment
	if opts.Environment != nil {
		for k, v := range opts.Environment {
			env[k] = v
		}
	}

	pt := &PTYTerminal{
		stdin:        stdin,
		stdout:       stdout,
		stderr:       stderr,
		fd:           fd,
		origTerm:     origTerm,
		width:        width,
		height:       height,
		scrollback:   NewScrollbackBuffer(opts.ScrollbackLines),
		ctx:          ctx,
		cancel:       cancel,
		mouseEnabled: opts.EnableMouse,
		env:          env,
		debug:        opts.Debug,
	}

	if pt.debug {
		log.Printf("PTYTerminal (Windows): Initialized with size=%dx%d", width, height)
	}

	return pt, nil
}

// Start activates the PTY terminal session.
// On Windows, this provides basic terminal functionality with I/O streaming
// and scrollback buffer support.
func (pt *PTYTerminal) Start() error {
	// Set terminal to raw mode
	if _, err := term.MakeRaw(pt.fd); err != nil {
		log.Printf("Failed to set terminal to raw mode: %v", err)
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer pt.restore()

	// Stream I/O with scrollback capture
	pt.wg.Add(3)

	// stdin -> session (with EOF detection)
	go func() {
		defer pt.wg.Done()
		pt.handleInput()
	}()

	// stdout -> os.Stdout (with scrollback)
	go func() {
		defer pt.wg.Done()
		pt.handleOutput()
	}()

	// stderr -> os.Stderr
	go func() {
		defer pt.wg.Done()
		_, err := io.Copy(os.Stderr, pt.stderr)
		if err != nil && err != io.EOF {
			if pt.debug {
				log.Printf("PTYTerminal (Windows): stderr copy error: %v", err)
			}
		}
	}()

	// Wait for all goroutines to finish
	pt.wg.Wait()

	if pt.debug {
		log.Println("PTYTerminal (Windows): Session ended")
	}

	return nil
}

// handleInput processes input from stdin with EOF detection.
func (pt *PTYTerminal) handleInput() {
	buf := make([]byte, 1024)
	eofCount := 0

	for {
		select {
		case <-pt.ctx.Done():
			return
		default:
			n, err := os.Stdin.Read(buf)
			if err != nil {
				if err == io.EOF {
					if pt.debug {
						log.Println("PTYTerminal (Windows): stdin EOF detected")
					}
					pt.cancel()
					return
				}
				if pt.debug {
					log.Printf("PTYTerminal (Windows): stdin read error: %v", err)
				}
				return
			}

			if n > 0 {
				// Check for EOF (Ctrl+D, ASCII 4) or Ctrl+Z (Windows EOF, ASCII 26)
				for i := 0; i < n; i++ {
					if buf[i] == 4 || buf[i] == 26 { // Ctrl+D or Ctrl+Z
						eofCount++
						if eofCount >= 2 {
							if pt.debug {
								log.Println("PTYTerminal (Windows): Multiple EOFs detected, exiting")
							}
							pt.cancel()
							return
						}
					} else {
						eofCount = 0
					}
				}

				// Write to stdin of the session
				written := 0
				for written < n {
					wn, err := pt.stdin.Write(buf[written:n])
					if err != nil {
						if err == io.EOF {
							if pt.debug {
								log.Println("PTYTerminal (Windows): stdin write EOF")
							}
							pt.cancel()
							return
						}
						if pt.debug {
							log.Printf("PTYTerminal (Windows): stdin write error: %v", err)
						}
						return
					}
					written += wn
				}
			}
		}
	}
}

// handleOutput processes output from stdout with scrollback buffer capture.
func (pt *PTYTerminal) handleOutput() {
	buf := make([]byte, 4096)
	var lineBuffer bytes.Buffer

	for {
		select {
		case <-pt.ctx.Done():
			return
		default:
			n, err := pt.stdout.Read(buf)
			if err != nil {
				if err == io.EOF {
					if pt.debug {
						log.Println("PTYTerminal (Windows): stdout EOF detected")
					}
					pt.cancel()
					return
				}
				if pt.debug {
					log.Printf("PTYTerminal (Windows): stdout read error: %v", err)
				}
				return
			}

			if n > 0 {
				// Write to actual stdout
				_, writeErr := os.Stdout.Write(buf[:n])
				if writeErr != nil {
					if pt.debug {
						log.Printf("PTYTerminal (Windows): stdout write error: %v", writeErr)
					}
					return
				}

				// Process output for scrollback buffer
				for i := 0; i < n; i++ {
					lineBuffer.WriteByte(buf[i])
					// Add to scrollback on newline (handle both \n and \r\n)
					if buf[i] == '\n' {
						pt.scrollback.AddLine(lineBuffer.Bytes())
						lineBuffer.Reset()
					}
				}
			}
		}
	}
}

// restore restores the original terminal state.
func (pt *PTYTerminal) restore() {
	if pt.origTerm != nil {
		term.Restore(pt.fd, pt.origTerm)
		if pt.debug {
			log.Println("PTYTerminal (Windows): Terminal state restored")
		}
	}
}

// Close cleanly shuts down the PTY terminal session.
func (pt *PTYTerminal) Close() error {
	if pt.debug {
		log.Println("PTYTerminal (Windows): Closing terminal session")
	}

	pt.cancel()

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		pt.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(500 * time.Millisecond):
		// Timeout (not ideal but prevents hang)
		if pt.debug {
			log.Println("PTYTerminal (Windows): Close timeout, forcing shutdown")
		}
	}

	pt.restore()

	if pt.debug {
		log.Println("PTYTerminal (Windows): Terminal session closed")
	}

	return nil
}

// GetScrollback returns the current scrollback buffer.
func (pt *PTYTerminal) GetScrollback() *ScrollbackBuffer {
	return pt.scrollback
}

// GetSize returns the current terminal size.
func (pt *PTYTerminal) GetSize() (width, height int) {
	return pt.width, pt.height
}

// SetEnvironment sets or updates environment variables for the terminal.
// Note: This must be called before Start() to take effect.
func (pt *PTYTerminal) SetEnvironment(key, value string) {
	pt.env[key] = value
	if pt.debug {
		log.Printf("PTYTerminal (Windows): Set environment variable %s=%s", key, value)
	}
}

// GetEnvironment returns all environment variables.
func (pt *PTYTerminal) GetEnvironment() map[string]string {
	envCopy := make(map[string]string, len(pt.env))
	for k, v := range pt.env {
		envCopy[k] = v
	}
	return envCopy
}
