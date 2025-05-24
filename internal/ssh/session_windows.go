//go:build windows
// +build windows

package ssh

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
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
		log.Printf("Warning: failed to get terminal size, using default 80x24: %v", err)
		width, height = 80, 24
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
		log.Printf("Failed to set terminal to raw mode: %v", err)
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	if err := s.session.Shell(); err != nil {
		term.Restore(s.originalFd, s.originalTty)
		log.Printf("Failed to start shell: %v", err)
		return fmt.Errorf("failed to start shell: %w", err)
	}

	s.wg.Add(3)

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
						log.Printf("stdin read error: %v", err)
					}
					close(s.done)
					return
				}
				if n > 0 {
					for i := 0; i < n; i++ {
						if buf[i] == 3 {
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
								log.Printf("stdin write error: %v", err)
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

	go func() {
		defer s.wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := s.stdout.Read(buf)
			if err != nil {
				s.exitMutex.Lock()
				s.exiting = true
				s.exitMutex.Unlock()

				close(s.done)

				if s.originalTty != nil {
					term.Restore(s.originalFd, s.originalTty)
				}
				return
			}

			if n > 0 {
				if containsSequence(buf[:n], "exit") || containsSequence(buf[:n], "logout") {
					os.Stdout.Write(buf[:n])
					time.Sleep(100 * time.Millisecond)
					continue
				}
				os.Stdout.Write(buf[:n])
			}
		}
	}()

	go func() {
		defer s.wg.Done()
		io.Copy(os.Stderr, s.stderr)
	}()

	err = s.session.Wait()

	s.exitMutex.Lock()
	s.exiting = true
	s.exitMutex.Unlock()

	select {
	case <-s.done:
	default:
		close(s.done)
	}

	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
	case <-time.After(500 * time.Millisecond):
	}

	if s.originalTty != nil {
		term.Restore(s.originalFd, s.originalTty)
		s.originalTty = nil
	}

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
	s.exitMutex.Lock()
	s.exiting = true
	s.exitMutex.Unlock()

	select {
	case <-s.done:
	default:
		close(s.done)
	}

	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
	case <-time.After(500 * time.Millisecond):
	}

	if s.originalTty != nil {
		term.Restore(s.originalFd, s.originalTty)
		s.originalTty = nil
	}

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
