package ssh

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"ssh-x-term/internal/config"
)

// Session represents an SSH terminal session
type Session struct {
	client      *Client
	session     *ssh.Session
	stdin       io.WriteCloser
	stdout      io.Reader
	stderr      io.Reader
	originalFd  int
	originalTty *term.State
	done        chan struct{}
	wg          sync.WaitGroup
	exiting     bool
	exitMutex   sync.Mutex
}

// NewSession creates a new SSH terminal session
func NewSession(connConfig config.SSHConnection) (*Session, error) {
	// Create a new SSH client
	client, err := NewClient(connConfig)
	if err != nil {
		return nil, err
	}

	// Create a new SSH session
	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		ssh.ICANON:        1,     // canonical input mode
		ssh.ISIG:          1,     // signals enabled
	}

	// Request pseudo terminal
	fd := int(os.Stdin.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	if err := sshSession.RequestPty("xterm-256color", height, width, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set up pipes for stdin, stdout, stderr
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("failed to set up stdin pipe: %w", err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("failed to set up stdout pipe: %w", err)
	}

	stderr, err := sshSession.StderrPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("failed to set up stderr pipe: %w", err)
	}

	s := &Session{
		client:     client,
		session:    sshSession,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		originalFd: fd,
		done:       make(chan struct{}),
		exiting:    false,
	}

	return s, nil
}

// safeWrite attempts to write to stdin, handling potential EOF errors during exit
func (s *Session) safeWrite(data []byte) (int, error) {
	s.exitMutex.Lock()
	exiting := s.exiting
	s.exitMutex.Unlock()

	if exiting {
		return len(data), nil // Simulate successful write during exit
	}

	n, err := s.stdin.Write(data)
	if err != nil && err == io.EOF {
		// If we get EOF, mark as exiting to prevent further writes
		s.exitMutex.Lock()
		s.exiting = true
		s.exitMutex.Unlock()

		// Don't report EOF as error during exit process
		return n, nil
	}
	return n, err
}

// Start starts the SSH session
func (s *Session) Start() error {
	// Store original terminal state
	var err error
	s.originalTty, err = term.MakeRaw(s.originalFd)
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	// Handle window resize
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	go func() {
		for {
			select {
			case <-winch:
				s.resizeTerminal()
			case <-s.done:
				return
			}
		}
	}()

	// Trigger initial resize
	s.resizeTerminal()

	// Start shell
	if err := s.session.Shell(); err != nil {
		term.Restore(s.originalFd, s.originalTty)
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Handle input and output with improved buffering
	s.wg.Add(3)

	// Handle stdin with improved buffering
	go func() {
		defer s.wg.Done()

		buf := make([]byte, 1024)
		for {
			select {
			case <-s.done:
				return
			default:
				n, err := os.Stdin.Read(buf)
				if err != nil {
					if err != io.EOF {
						// Only log if not EOF
						fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
					}
					return
				}

				if n > 0 {
					// Check if we're in an exiting state before writing
					s.exitMutex.Lock()
					exiting := s.exiting
					s.exitMutex.Unlock()

					if exiting {
						// If we're exiting, don't try to write any more data
						continue
					}

					// Write in chunks to prevent blocking
					written := 0
					for written < n {
						wn, err := s.safeWrite(buf[written:n])
						if err != nil {
							// Only log if it's not an EOF during exit
							s.exitMutex.Lock()
							exiting := s.exiting
							s.exitMutex.Unlock()

							if !exiting {
								fmt.Fprintf(os.Stderr, "stdin write error: %v\n", err)
							}
							return
						}
						written += wn
						if written < n {
							// Check if we need to stop
							select {
							case <-s.done:
								return
							default:
								// Continue writing remaining data
							}
						}
					}
				}
			}
		}
	}()

	// Create a custom stdout writer that preserves cursor visibility
	stdoutWriter := &cursorPreservingWriter{
		out:     os.Stdout,
		session: s,
	}

	// Handle stdout with cursor preservation
	go func() {
		defer s.wg.Done()
		io.Copy(stdoutWriter, s.stdout)
	}()

	// Handle stderr
	go func() {
		defer s.wg.Done()
		io.Copy(os.Stderr, s.stderr)
	}()

	// Wait for session to complete
	err = s.session.Wait()

	// Mark as exiting to prevent further stdin writes
	s.exitMutex.Lock()
	s.exiting = true
	s.exitMutex.Unlock()

	// Signal all goroutines to stop
	close(s.done)

	// Wait for all goroutines to finish (with timeout)
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	// Wait with timeout
	select {
	case <-waitCh:
		// Normal completion
	case <-time.After(500 * time.Millisecond):
		// Timeout - force continue
	}

	// We don't show cursor here because it might cause an EOF error
	// The cursor visibility will be handled by BubbleTea

	return err
}

// cursorPreservingWriter is a custom io.Writer that ensures cursor visibility
type cursorPreservingWriter struct {
	out     io.Writer
	session *Session
}

func (w *cursorPreservingWriter) Write(p []byte) (n int, err error) {
	// Write the actual data
	return w.out.Write(p)
}

// containsSequence checks if a byte slice contains an escape sequence
func containsSequence(data []byte, sequence string) bool {
	seqBytes := []byte(sequence)
	if len(seqBytes) > len(data) {
		return false
	}

	// Simple search - could be optimized for production
	for i := 0; i <= len(data)-len(seqBytes); i++ {
		match := true
		for j := 0; j < len(seqBytes); j++ {
			if data[i+j] != seqBytes[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Close closes the SSH session and restores the terminal
func (s *Session) Close() error {
	// Mark as exiting first to prevent further stdin writes
	s.exitMutex.Lock()
	s.exiting = true
	s.exitMutex.Unlock()

	// Signal all goroutines to stop
	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}

	// Wait for all goroutines to finish (with timeout)
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	// Wait with timeout
	select {
	case <-waitCh:
		// Normal completion
	case <-time.After(500 * time.Millisecond):
		// Timeout - force continue
	}

	// We intentionally don't try to show cursor here because it might cause EOF errors
	// Let BubbleTea handle cursor visibility after we return

	// Restore terminal state
	if s.originalTty != nil {
		term.Restore(s.originalFd, s.originalTty)
	}

	// Close session and client
	var sessionErr, clientErr error
	if s.session != nil {
		sessionErr = s.session.Close()
	}
	if s.client != nil {
		clientErr = s.client.Close()
	}

	if sessionErr != nil {
		return sessionErr
	}
	return clientErr
}

// resizeTerminal updates the terminal size in the SSH session
func (s *Session) resizeTerminal() {
	// Don't attempt to resize if we're exiting
	s.exitMutex.Lock()
	exiting := s.exiting
	s.exitMutex.Unlock()

	if exiting {
		return
	}

	width, height, err := term.GetSize(s.originalFd)
	if err != nil {
		return
	}
	s.session.WindowChange(height, width)
}

// IsTerminated checks if the session has been terminated
func (s *Session) IsTerminated() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}
