//go:build windows
// +build windows

package ssh

import (
	"fmt"
	"io"
	"os"
	"sync"
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
	client, err := NewClient(connConfig)
	if err != nil {
		return nil, err
	}

	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.ICANON:        1,
		ssh.ISIG:          1,
	}

	fd := int(os.Stdin.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil || width <= 0 || height <= 0 {
		fmt.Fprintf(os.Stderr, "Warning: failed to get terminal size, using default 80x24: %v\n", err)
		width, height = 80, 24
	}

	if err := sshSession.RequestPty("xterm-256color", height, width, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

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

func (s *Session) safeWrite(data []byte) (int, error) {
	s.exitMutex.Lock()
	exiting := s.exiting
	s.exitMutex.Unlock()

	if exiting {
		return len(data), nil
	}

	n, err := s.stdin.Write(data)
	if err != nil && err == io.EOF {
		s.exitMutex.Lock()
		s.exiting = true
		s.exitMutex.Unlock()
		return n, nil
	}
	return n, err
}

func (s *Session) Start() error {
	var err error
	s.originalTty, err = term.MakeRaw(s.originalFd)
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	// Start shell
	if err := s.session.Shell(); err != nil {
		term.Restore(s.originalFd, s.originalTty)
		return fmt.Errorf("failed to start shell: %w", err)
	}

	s.wg.Add(3)

	// Handle stdin
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
						fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
					}
					close(s.done)
					return
				}
				if n > 0 {
					// Check for exit sequences (Ctrl+C, etc.)
					for i := 0; i < n; i++ {
						if buf[i] == 3 { // Ctrl+C
							s.exitMutex.Lock()
							s.exiting = true
							s.exitMutex.Unlock()
							close(s.done)
							return
						}
					}

					s.exitMutex.Lock()
					exiting := s.exiting
					s.exitMutex.Unlock()

					if exiting {
						continue
					}

					written := 0
					for written < n {
						wn, err := s.safeWrite(buf[written:n])
						if err != nil {
							s.exitMutex.Lock()
							exiting := s.exiting
							s.exitMutex.Unlock()
							if !exiting {
								fmt.Fprintf(os.Stderr, "stdin write error: %v\n", err)
							}
							close(s.done)
							return
						}
						written += wn
						if written < n {
							select {
							case <-s.done:
								return
							default:
							}
						}
					}
				}
			}
		}
	}()

	// Handle stdout - this is where we detect SSH session termination
	go func() {
		defer s.wg.Done()
		buf := make([]byte, 32*1024)

		for {
			n, err := s.stdout.Read(buf)
			if err != nil {
				// This is most likely EOF, meaning the SSH session has ended
				s.exitMutex.Lock()
				s.exiting = true
				s.exitMutex.Unlock()

				// Signal session end
				close(s.done)

				// Since we're exiting, immediately restore terminal state
				if s.originalTty != nil {
					term.Restore(s.originalFd, s.originalTty)
				}

				return
			}

			if n > 0 {
				// Check if output contains "exit" or termination sequences
				if containsSequence(buf[:n], "exit") || containsSequence(buf[:n], "logout") {
					// This might be an exit command, but still write the data
					os.Stdout.Write(buf[:n])

					// Wait a brief moment for the SSH session to process exit
					time.Sleep(100 * time.Millisecond)
					continue
				}

				// Normal output
				os.Stdout.Write(buf[:n])
			}
		}
	}()

	// Handle stderr
	go func() {
		defer s.wg.Done()
		io.Copy(os.Stderr, s.stderr)
	}()

	// Wait for shell session to complete
	err = s.session.Wait()

	// Mark session as exiting
	s.exitMutex.Lock()
	s.exiting = true
	s.exitMutex.Unlock()

	// Signal all goroutines to exit
	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}

	// Wait for goroutines with timeout
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// All goroutines finished
	case <-time.After(500 * time.Millisecond):
		// Timeout, continue anyway
	}

	// Explicitly restore terminal state if needed
	if s.originalTty != nil {
		term.Restore(s.originalFd, s.originalTty)
	}

	return err
}

// containsSequence checks if the data contains a specific byte sequence
func containsSequence(data []byte, sequence string) bool {
	seqBytes := []byte(sequence)
	if len(seqBytes) > len(data) {
		return false
	}

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

func (s *Session) Close() error {
	// Mark as exiting
	s.exitMutex.Lock()
	s.exiting = true
	s.exitMutex.Unlock()

	// Signal all goroutines to exit
	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}

	// Wait for goroutines with timeout
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// All goroutines finished
	case <-time.After(500 * time.Millisecond):
		// Timeout, continue anyway
	}

	// Restore terminal state
	if s.originalTty != nil {
		term.Restore(s.originalFd, s.originalTty)
		s.originalTty = nil // Mark as restored to prevent double restore
	}

	// Close session and client
	var sessionErr, clientErr error
	if s.session != nil {
		sessionErr = s.session.Close()
		s.session = nil
	}
	if s.client != nil {
		clientErr = s.client.Close()
		s.client = nil
	}

	if sessionErr != nil {
		return sessionErr
	}
	return clientErr
}

func (s *Session) IsTerminated() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}
