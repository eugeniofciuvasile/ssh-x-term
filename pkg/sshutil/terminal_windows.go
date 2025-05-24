//go:build windows
// +build windows

package sshutil

import (
	"fmt"
	"io"
	"log"
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
		log.Printf("Failed to get terminal state: %v", err)
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
		log.Printf("Failed to set terminal to raw mode: %v", err)
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	doneCh := make(chan struct{})

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

	errCh := make(chan error, 3)

	go func() {
		_, err := io.Copy(ts.stdin, os.Stdin)
		if err != nil && err != io.EOF {
			log.Printf("stdin copy error: %v", err)
			errCh <- err
		}
		close(doneCh)
	}()

	go func() {
		_, err := io.Copy(os.Stdout, ts.stdout)
		if err != nil && err != io.EOF {
			log.Printf("stdout copy error: %v", err)
			errCh <- err
		}
		safeRestore()
		close(doneCh)
	}()

	go func() {
		_, err := io.Copy(os.Stderr, ts.stderr)
		if err != nil && err != io.EOF {
			log.Printf("stderr copy error: %v", err)
			errCh <- err
		}
	}()

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
