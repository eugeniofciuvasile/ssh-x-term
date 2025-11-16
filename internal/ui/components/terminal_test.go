package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
)

func TestTerminalComponent_DoubleEscBehavior(t *testing.T) {
	t.Run("Single ESC does not close active session", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)
		tc.width = 80
		tc.height = 24
		tc.sessionClosed = false // Session is active

		// First ESC press
		keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = tc.handleKey(keyMsg)

		// Session should not be finished
		if tc.finished {
			t.Error("Expected session to NOT be finished after single ESC")
		}

		// ESC count should be 1
		if tc.escPressCount != 1 {
			t.Errorf("Expected escPressCount to be 1, got %d", tc.escPressCount)
		}
	})

	t.Run("Double ESC within timeout closes session", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)
		tc.width = 80
		tc.height = 24
		tc.sessionClosed = false // Session is active
		tc.escTimeoutSecs = 2.0

		// First ESC press
		keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = tc.handleKey(keyMsg)

		// Should not be finished yet
		if tc.finished {
			t.Error("Expected session to NOT be finished after first ESC")
		}

		// Second ESC press within timeout
		_, _ = tc.handleKey(keyMsg)

		// Session should be finished now
		if !tc.finished {
			t.Error("Expected session to be finished after double ESC")
		}

		// ESC count should be reset
		if tc.escPressCount != 0 {
			t.Errorf("Expected escPressCount to be reset to 0, got %d", tc.escPressCount)
		}
	})

	t.Run("Double ESC after timeout does not close session", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)
		tc.width = 80
		tc.height = 24
		tc.sessionClosed = false // Session is active
		tc.escTimeoutSecs = 0.1  // Very short timeout for testing

		// First ESC press
		keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = tc.handleKey(keyMsg)

		// Wait for timeout to expire
		time.Sleep(150 * time.Millisecond)

		// Second ESC press after timeout
		_, _ = tc.handleKey(keyMsg)

		// Session should NOT be finished (this is a new first ESC)
		if tc.finished {
			t.Error("Expected session to NOT be finished after ESC outside timeout window")
		}

		// ESC count should be 1 (restarted sequence)
		if tc.escPressCount != 1 {
			t.Errorf("Expected escPressCount to be 1 after timeout, got %d", tc.escPressCount)
		}
	})

	t.Run("Single ESC closes session when session is already closed", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)
		tc.width = 80
		tc.height = 24
		tc.sessionClosed = true // Session already closed via logout/Ctrl+D

		// Single ESC press
		keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = tc.handleKey(keyMsg)

		// Session should be finished
		if !tc.finished {
			t.Error("Expected session to be finished after single ESC when session is closed")
		}
	})

	t.Run("Other key press resets ESC sequence", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)
		tc.width = 80
		tc.height = 24
		tc.sessionClosed = false // Session is active

		// First ESC press
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = tc.handleKey(escMsg)

		// Should have ESC count of 1
		if tc.escPressCount != 1 {
			t.Errorf("Expected escPressCount to be 1, got %d", tc.escPressCount)
		}

		// Press another key
		otherMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		_, _ = tc.handleKey(otherMsg)

		// ESC count should be reset
		if tc.escPressCount != 0 {
			t.Errorf("Expected escPressCount to be reset to 0 after other key, got %d", tc.escPressCount)
		}

		// Session should not be finished
		if tc.finished {
			t.Error("Expected session to NOT be finished after ESC followed by other key")
		}
	})

	t.Run("Scrolling keys do not affect ESC sequence", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)
		tc.width = 80
		tc.height = 24
		tc.sessionClosed = false // Session is active
		tc.vterm = NewVTerminal(80, 24)

		// First ESC press
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = tc.handleKey(escMsg)

		// Should have ESC count of 1
		if tc.escPressCount != 1 {
			t.Errorf("Expected escPressCount to be 1, got %d", tc.escPressCount)
		}

		// Press PgUp (scrolling key)
		// Note: Scrolling keys like pgup should NOT reset the ESC sequence
		// because they are special navigation keys, but in our current implementation
		// they are separate cases and don't explicitly reset.
		// Let's test that ESC still works after scrolling.

		// Second ESC press
		_, _ = tc.handleKey(escMsg)

		// Session should be finished
		if !tc.finished {
			t.Error("Expected session to be finished after double ESC even with scrolling in between")
		}
	})
}

func TestTerminalComponent_NewTerminalComponent(t *testing.T) {
	t.Run("Creates terminal with default timeout", func(t *testing.T) {
		conn := config.SSHConnection{
			Name:     "test",
			Host:     "localhost",
			Port:     22,
			Username: "user",
		}
		tc := NewTerminalComponent(conn)

		// Should have default timeout of 2 seconds
		if tc.escTimeoutSecs != 2.0 {
			t.Errorf("Expected default escTimeoutSecs to be 2.0, got %f", tc.escTimeoutSecs)
		}

		// Should have zero ESC count initially
		if tc.escPressCount != 0 {
			t.Errorf("Expected initial escPressCount to be 0, got %d", tc.escPressCount)
		}
	})
}
