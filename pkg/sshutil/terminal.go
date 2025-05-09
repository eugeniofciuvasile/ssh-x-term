package sshutil

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

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

	// Save the original terminal state
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
func (ts *TerminalSession) Start() error {
	if _, err := term.MakeRaw(ts.fd); err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer term.Restore(ts.fd, ts.origTerm)

	// Handle terminal resize signals
	resizeCh := make(chan os.Signal, 1)
	signal.Notify(resizeCh, syscall.SIGWINCH)
	defer signal.Stop(resizeCh)

	// Launch goroutines for I/O streaming
	errCh := make(chan error, 1)

	go func() {
		_, err := io.Copy(ts.stdin, os.Stdin)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(os.Stdout, ts.stdout)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(os.Stderr, ts.stderr)
		errCh <- err
	}()

	// Handle signals and I/O errors
	for {
		select {
		case err := <-errCh:
			return err
		case <-resizeCh:
			if err := ts.handleResize(); err != nil {
				fmt.Fprintf(os.Stderr, "Resize error: %v\n", err)
			}
		}
	}
}

// handleResize gets and prints the new terminal size. Extend as needed.
func (ts *TerminalSession) handleResize() error {
	width, height, err := term.GetSize(ts.fd)
	if err != nil {
		return fmt.Errorf("failed to get terminal size: %w", err)
	}

	// Optional: handle resize logic (e.g., notify remote, etc.)
	fmt.Fprintf(os.Stderr, "\nTerminal resized to %dx%d\n", width, height)
	return nil
}

// GetTerminalSize returns the current terminal width and height.
func GetTerminalSize() (width, height int, err error) {
	return term.GetSize(int(os.Stdin.Fd()))
}
