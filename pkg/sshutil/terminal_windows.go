//go:build windows
// +build windows

package sshutil

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// TerminalSession manages input/output streams with terminal state control.
type TerminalSession struct {
	stdin    io.Writer
	stdout   io.Reader
	stderr   io.Reader
	fd       int
	origTerm *term.State
}

// NewTerminalSession creates and returns a TerminalSession.
func NewTerminalSession(stdin io.Writer, stdout, stderr io.Reader) (*TerminalSession, error) {
	fd := int(os.Stdin.Fd())
	origTerm, err := term.GetState(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal state: %w", err)
	}
	return &TerminalSession{
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		fd:       fd,
		origTerm: origTerm,
	}, nil
}

// Start activates the terminal session, manages raw mode and I/O streaming.
// Similar to the Unix implementation but without signal handling.
func (ts *TerminalSession) Start() error {
	// Set terminal to raw mode
	if _, err := term.MakeRaw(ts.fd); err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	// Create a channel to signal when to exit and restore terminal
	doneCh := make(chan struct{})

	// Create a mutex to prevent terminal state from being restored multiple times
	var restoreMutex sync.Mutex
	var alreadyRestored bool

	safeRestore := func() {
		restoreMutex.Lock()
		defer restoreMutex.Unlock()
		if !alreadyRestored {
			term.Restore(ts.fd, ts.origTerm)
			alreadyRestored = true
		}
	}

	// Create error channel for propagating errors
	errCh := make(chan error, 3)

	// Handle stdin
	go func() {
		_, err := io.Copy(ts.stdin, os.Stdin)
		if err != nil && err != io.EOF {
			errCh <- err
		}
		// Signal completion
		close(doneCh)
	}()

	// Handle stdout
	go func() {
		_, err := io.Copy(os.Stdout, ts.stdout)
		if err != nil && err != io.EOF {
			errCh <- err
		}
		// When stdout ends (SSH connection terminates), restore terminal
		safeRestore()
		// Signal completion
		close(doneCh)
	}()

	// Handle stderr
	go func() {
		_, err := io.Copy(os.Stderr, ts.stderr)
		if err != nil && err != io.EOF {
			errCh <- err
		}
	}()

	// Wait for either an error or completion signal
	select {
	case err := <-errCh:
		safeRestore()
		return err
	case <-doneCh:
		safeRestore()
		return nil
	}
}

// GetTerminalSize returns the current terminal width and height.
func GetTerminalSize() (width, height int, err error) {
	return term.GetSize(int(os.Stdin.Fd()))
}
