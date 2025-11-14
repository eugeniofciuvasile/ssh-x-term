package components

import (
	"strings"
	"testing"
)

func TestVTerminalBasics(t *testing.T) {
	vt := NewVTerminal(80, 24)

	// Test writing simple text
	vt.Write([]byte("Hello World"))
	output := vt.Render()
	if !strings.Contains(output, "Hello World") {
		t.Errorf("Expected output to contain 'Hello World', got: %s", output)
	}

	// Test cursor position
	x, y := vt.GetCursorPosition()
	if x != 11 || y != 0 {
		t.Errorf("Expected cursor at (11, 0), got (%d, %d)", x, y)
	}
}

func TestVTerminalNewline(t *testing.T) {
	vt := NewVTerminal(80, 24)

	vt.Write([]byte("Line 1\n"))
	vt.Write([]byte("Line 2"))

	output := vt.Render()
	if !strings.Contains(output, "Line 1") || !strings.Contains(output, "Line 2") {
		t.Errorf("Expected output to contain both lines, got: %s", output)
	}

	_, y := vt.GetCursorPosition()
	if y != 1 {
		t.Errorf("Expected cursor Y at 1, got %d", y)
	}
}

func TestVTerminalEscapeSequences(t *testing.T) {
	vt := NewVTerminal(80, 24)

	// Test cursor positioning with ESC[H
	vt.Write([]byte("Hello\x1B[1;1HWorld"))
	output := vt.Render()

	// The cursor should have moved to 1,1 (top-left), overwriting "Hello" with "World"
	if !strings.Contains(output, "World") {
		t.Errorf("Expected output to contain 'World', got: %s", output)
	}
}

func TestVTerminalResize(t *testing.T) {
	vt := NewVTerminal(80, 24)

	vt.Write([]byte("Test"))
	vt.Resize(40, 12)

	// After resize, the buffer should be cleared
	x, y := vt.GetCursorPosition()
	if x != 0 || y != 0 {
		t.Errorf("Expected cursor at (0, 0) after resize, got (%d, %d)", x, y)
	}
}

func TestVTerminalScrolling(t *testing.T) {
	vt := NewVTerminal(80, 10)

	// Write enough lines to cause scrolling
	for i := 0; i < 15; i++ {
		vt.Write([]byte("Line "))
		vt.Write([]byte{byte('0' + i)})
		vt.Write([]byte("\n"))
	}

	// Check that scrollback buffer has content
	vt.ScrollUp(5)
	if !vt.IsScrolledBack() {
		t.Error("Expected terminal to be scrolled back")
	}

	vt.ScrollToBottom()
	if vt.IsScrolledBack() {
		t.Error("Expected terminal to not be scrolled back after ScrollToBottom")
	}
}

func TestVTerminalClearScreen(t *testing.T) {
	vt := NewVTerminal(80, 24)

	vt.Write([]byte("Some text"))

	// Clear screen with CSI 2J
	vt.Write([]byte("\x1B[2J"))

	output := vt.Render()
	// After clearing, the output should be mostly empty (just whitespace and newlines)
	nonWhitespace := strings.TrimSpace(output)
	if len(nonWhitespace) > 0 {
		t.Errorf("Expected empty output after clear, got: %q", nonWhitespace)
	}
}

func TestVTerminalRenderOutput(t *testing.T) {
	vt := NewVTerminal(80, 24)

	// Write a simple prompt
	vt.Write([]byte("user@host:~$ "))

	output := vt.Render()
	if !strings.Contains(output, "user@host:~$") {
		t.Errorf("Expected output to contain prompt, got: %s", output)
	}

	// Write a command and newline
	vt.Write([]byte("ls -la\r\n"))

	output = vt.Render()
	if !strings.Contains(output, "ls -la") {
		t.Errorf("Expected output to contain command, got: %s", output)
	}
}

func TestVTerminalMinimumDimensions(t *testing.T) {
	// Test that zero dimensions are handled
	vt := NewVTerminal(0, 0)

	if vt.width < 1 || vt.height < 1 {
		t.Errorf("Expected minimum dimensions, got width=%d, height=%d", vt.width, vt.height)
	}

	// Should be able to write without panic
	vt.Write([]byte("Test"))

	output := vt.Render()
	if !strings.Contains(output, "Test") {
		t.Errorf("Expected output to contain 'Test', got: %s", output)
	}
}
