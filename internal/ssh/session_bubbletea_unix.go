//go:build !windows
// +build !windows

package ssh

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"golang.org/x/crypto/ssh"
)

// BubbleTeaSession represents an SSH session that works within Bubble Tea
type BubbleTeaSession struct {
	client  *Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	done    chan struct{}
	wg      sync.WaitGroup
	exiting bool
	mutex   sync.Mutex
	width   int
	height  int
}

// NewBubbleTeaSession creates a new SSH session for use within Bubble Tea
func NewBubbleTeaSession(connConfig config.SSHConnection, width, height int) (*BubbleTeaSession, error) {
	client, err := NewClient(connConfig)
	if err != nil {
		return nil, err
	}

	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.ICANON:        1,
		ssh.ISIG:          1,
	}

	// Request PTY
	if err := sshSession.RequestPty("xterm-256color", height, width, modes); err != nil {
		sshSession.Close()
		client.Close()
		log.Printf("Failed to request PTY: %v", err)
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set up pipes
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		log.Printf("Failed to set up stdin pipe: %v", err)
		return nil, fmt.Errorf("failed to set up stdin pipe: %w", err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		log.Printf("Failed to set up stdout pipe: %v", err)
		return nil, fmt.Errorf("failed to set up stdout pipe: %w", err)
	}

	stderr, err := sshSession.StderrPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		log.Printf("Failed to set up stderr pipe: %v", err)
		return nil, fmt.Errorf("failed to set up stderr pipe: %w", err)
	}

	s := &BubbleTeaSession{
		client:  client,
		session: sshSession,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		done:    make(chan struct{}),
		width:   width,
		height:  height,
	}

	return s, nil
}

// Start starts the SSH shell
func (s *BubbleTeaSession) Start() error {
	if err := s.session.Shell(); err != nil {
		log.Printf("Failed to start shell: %v", err)
		return fmt.Errorf("failed to start shell: %w", err)
	}
	return nil
}

// Write sends data to the SSH session stdin
func (s *BubbleTeaSession) Write(data []byte) (int, error) {
	s.mutex.Lock()
	exiting := s.exiting
	s.mutex.Unlock()

	if exiting {
		return len(data), nil
	}

	return s.stdin.Write(data)
}

// Read reads data from the SSH session stdout
func (s *BubbleTeaSession) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

// ReadStderr reads data from the SSH session stderr
func (s *BubbleTeaSession) ReadStderr(p []byte) (int, error) {
	return s.stderr.Read(p)
}

// Resize sends a window change signal to the SSH session
func (s *BubbleTeaSession) Resize(width, height int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.exiting {
		return nil
	}

	s.width = width
	s.height = height

	return s.session.WindowChange(height, width)
}

// Wait waits for the session to complete
func (s *BubbleTeaSession) Wait() error {
	return s.session.Wait()
}

// Close closes the SSH session
func (s *BubbleTeaSession) Close() error {
	s.mutex.Lock()
	s.exiting = true
	s.mutex.Unlock()

	select {
	case <-s.done:
	default:
		close(s.done)
	}

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

// IsTerminated returns whether the session has been terminated
func (s *BubbleTeaSession) IsTerminated() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}
