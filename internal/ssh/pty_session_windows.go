//go:build windows
// +build windows

package ssh

import (
	"fmt"
	"io"
	"log"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"golang.org/x/crypto/ssh"
)

// PTYSession represents a PTY-based SSH session for use within Bubble Tea (Windows)
type PTYSession struct {
	client  *Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	width   int
	height  int
}

// NewPTYSession creates a new PTY SSH session that can be integrated with Bubble Tea
func NewPTYSession(connConfig config.SSHConnection, width, height int) (*PTYSession, error) {
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

	if err := sshSession.RequestPty("xterm-256color", height, width, modes); err != nil {
		sshSession.Close()
		client.Close()
		log.Printf("Failed to request PTY: %v", err)
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

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

	// Start the shell
	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		log.Printf("Failed to start shell: %v", err)
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	s := &PTYSession{
		client:  client,
		session: sshSession,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		width:   width,
		height:  height,
	}

	return s, nil
}

// Read reads output from the SSH session
func (s *PTYSession) Read(p []byte) (n int, err error) {
	return s.stdout.Read(p)
}

// Write writes input to the SSH session
func (s *PTYSession) Write(p []byte) (n int, err error) {
	return s.stdin.Write(p)
}

// Resize changes the terminal size
func (s *PTYSession) Resize(width, height int) error {
	s.width = width
	s.height = height
	return s.session.WindowChange(height, width)
}

// Close closes the SSH session
func (s *PTYSession) Close() error {
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

// Wait waits for the session to finish
func (s *PTYSession) Wait() error {
	if s.session != nil {
		return s.session.Wait()
	}
	return nil
}
