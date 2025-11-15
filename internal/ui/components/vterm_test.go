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

func TestVTerminalOSCSequences(t *testing.T) {
	vt := NewVTerminal(80, 24)

	// Write a prompt
	vt.Write([]byte("user@host:~$ "))

	// Write an OSC sequence (e.g., setting window title) followed by text
	// This simulates what bash does with PS1 that includes title setting
	vt.Write([]byte("\x1B]0;user@host:~\x07"))
	vt.Write([]byte("ls\r\n"))

	output := vt.Render()

	// The OSC sequence should be ignored, we should only see the prompt and command
	if !strings.Contains(output, "user@host:~$") {
		t.Errorf("Expected prompt in output")
	}
	if !strings.Contains(output, "ls") {
		t.Errorf("Expected 'ls' command in output")
	}
	// The OSC content should NOT appear
	if strings.Contains(output, "0;user@host:~") {
		t.Errorf("OSC sequence content should not appear in output, got: %s", output)
	}
}

// Test control character handling
func TestVTerminalControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Carriage return moves cursor to start",
			input:    []byte("Hello\rWorld"),
			expected: "World", // World overwrites Hello from the start
		},
		{
			name:     "Tab moves cursor forward",
			input:    []byte("A\tB"),
			expected: "A       B", // Tab expands to spaces
		},
		{
			name:     "Backspace moves cursor back",
			input:    []byte("Hello\b\b\b\bGoodbye"),
			expected: "Goodbye", // Overwrites "Hello" with "Goodbye"
		},
		{
			name:     "Form feed advances line",
			input:    []byte("Line1\x0CLine2"),
			expected: "Line1", // FF moves to new line
		},
		{
			name:     "Vertical tab advances line",
			input:    []byte("Line1\x0BLine2"),
			expected: "Line1", // VT moves to new line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewVTerminal(80, 24)
			vt.Write(tt.input)
			output := vt.Render()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got: %s", tt.expected, output)
			}
		})
	}
}

// Test additional CSI sequences
func TestVTerminalCSISequences(t *testing.T) {
	t.Run("Cursor save and restore", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		vt.Write([]byte("Hello"))
		// Save cursor (should be at position 5,0)
		vt.Write([]byte("\x1B[s"))
		vt.Write([]byte("World"))
		// Restore cursor (back to 5,0)
		vt.Write([]byte("\x1B[u"))
		vt.Write([]byte("!"))
		
		output := vt.Render()
		if !strings.Contains(output, "Hello") {
			t.Errorf("Expected output to contain 'Hello', got: %s", output)
		}
	})

	t.Run("Vertical position absolute", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		vt.Write([]byte("Line1\n"))
		// Move to line 3
		vt.Write([]byte("\x1B[3d"))
		vt.Write([]byte("Line3"))
		
		_, y := vt.GetCursorPosition()
		if y != 2 { // 0-indexed, so line 3 is y=2
			t.Errorf("Expected cursor Y at 2, got %d", y)
		}
	})

	t.Run("Horizontal position absolute", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Move to column 10
		vt.Write([]byte("\x1B[10G"))
		vt.Write([]byte("X"))
		
		x, _ := vt.GetCursorPosition()
		if x != 10 {
			t.Errorf("Expected cursor X at 10, got %d", x)
		}
	})

	t.Run("Erase characters", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		vt.Write([]byte("HelloWorld"))
		// Move back to position 5
		vt.Write([]byte("\x1B[5D"))
		// Erase 5 characters
		vt.Write([]byte("\x1B[5X"))
		
		output := vt.Render()
		// Should have "Hello     " (5 spaces where "World" was)
		if !strings.Contains(output, "Hello") {
			t.Errorf("Expected output to contain 'Hello', got: %s", output)
		}
	})

	t.Run("Delete characters", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		vt.Write([]byte("HelloWorld"))
		// Move to position 5
		vt.Write([]byte("\x1B[1;6H"))
		// Delete 3 characters (removes "Wor", shifts "ld" left)
		vt.Write([]byte("\x1B[3P"))
		
		output := vt.Render()
		if !strings.Contains(output, "Hellold") {
			t.Errorf("Expected output to contain 'Hellold', got: %s", output)
		}
	})

	t.Run("Insert lines", func(t *testing.T) {
		vt := NewVTerminal(80, 5)
		vt.Write([]byte("Line1\nLine2\nLine3"))
		// Move to line 2
		vt.Write([]byte("\x1B[2;1H"))
		// Insert 1 line (pushes Line2 and Line3 down)
		vt.Write([]byte("\x1B[1L"))
		
		output := vt.Render()
		if !strings.Contains(output, "Line1") {
			t.Errorf("Expected output to contain 'Line1', got: %s", output)
		}
	})

	t.Run("Delete lines", func(t *testing.T) {
		vt := NewVTerminal(80, 5)
		vt.Write([]byte("Line1\nLine2\nLine3"))
		// Move to line 2
		vt.Write([]byte("\x1B[2;1H"))
		// Delete 1 line (removes Line2, pulls Line3 up)
		vt.Write([]byte("\x1B[1M"))
		
		output := vt.Render()
		if !strings.Contains(output, "Line1") {
			t.Errorf("Expected output to contain 'Line1', got: %s", output)
		}
		if !strings.Contains(output, "Line3") {
			t.Errorf("Expected output to contain 'Line3', got: %s", output)
		}
	})
}

// Test that all control characters are properly ignored or handled
func TestVTerminalAllControlCharacters(t *testing.T) {
	vt := NewVTerminal(80, 24)
	
	// Test NUL character (should be ignored)
	vt.Write([]byte("Hello\x00World"))
	output := vt.Render()
	if !strings.Contains(output, "HelloWorld") {
		t.Errorf("Expected NUL to be ignored, got: %s", output)
	}
	
	// Test Bell (should be ignored)
	vt.Clear()
	vt.Write([]byte("Test\x07Text"))
	output = vt.Render()
	if !strings.Contains(output, "TestText") {
		t.Errorf("Expected Bell to be ignored, got: %s", output)
	}
	
	// Test other control characters are safely ignored
	vt.Clear()
	vt.Write([]byte("A\x01\x02\x03B")) // Various control chars
	output = vt.Render()
	if !strings.Contains(output, "AB") {
		t.Errorf("Expected control chars to be ignored, got: %s", output)
	}
}

// Test extended ASCII characters
func TestVTerminalExtendedASCII(t *testing.T) {
	vt := NewVTerminal(80, 24)
	
	// Write some extended ASCII characters
	vt.Write([]byte{0xC1, 0xC2, 0xC3}) // Extended ASCII
	output := vt.Render()
	
	// Should not panic and should render something
	if len(strings.TrimSpace(output)) == 0 {
		t.Error("Expected extended ASCII to be rendered")
	}
}
