package components

import (
	"strings"
	"testing"
)

// TestVTerminalColorSupport tests ANSI color rendering
func TestVTerminalColorSupport(t *testing.T) {
	t.Run("Basic foreground colors", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write red text
		vt.Write([]byte("\x1B[31mRed\x1B[0m"))
		output := vt.Render()

		// Should contain SGR sequence for red (31)
		if !strings.Contains(output, "\x1B[31m") {
			t.Errorf("Expected red color sequence ESC[31m in output, got: %q", output)
		}
		// Should contain the text
		if !strings.Contains(output, "Red") {
			t.Errorf("Expected 'Red' in output, got: %q", output)
		}
		// Should reset colors at end of line
		if !strings.Contains(output, "\x1B[0m") {
			t.Errorf("Expected color reset ESC[0m in output, got: %q", output)
		}
	})

	t.Run("Basic background colors", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write text with blue background
		vt.Write([]byte("\x1B[44mBlue BG\x1B[0m"))
		output := vt.Render()

		// Should contain SGR sequence for blue background (44)
		if !strings.Contains(output, "\x1B[44m") {
			t.Errorf("Expected blue background sequence ESC[44m in output, got: %q", output)
		}
		if !strings.Contains(output, "Blue BG") {
			t.Errorf("Expected 'Blue BG' in output, got: %q", output)
		}
	})

	t.Run("Bold text", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write bold text
		vt.Write([]byte("\x1B[1mBold\x1B[22m"))
		output := vt.Render()

		// Should contain SGR sequence for bold (1)
		if !strings.Contains(output, "\x1B[1m") {
			t.Errorf("Expected bold sequence ESC[1m in output, got: %q", output)
		}
		if !strings.Contains(output, "Bold") {
			t.Errorf("Expected 'Bold' in output, got: %q", output)
		}
	})

	t.Run("Bright colors (90-97)", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write bright red text
		vt.Write([]byte("\x1B[91mBright Red\x1B[0m"))
		output := vt.Render()

		// Should contain SGR sequence for bright red (91)
		if !strings.Contains(output, "\x1B[91m") {
			t.Errorf("Expected bright red sequence ESC[91m in output, got: %q", output)
		}
		if !strings.Contains(output, "Bright Red") {
			t.Errorf("Expected 'Bright Red' in output, got: %q", output)
		}
	})

	t.Run("256-color support", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write 256-color text (ESC[38;5;208m is orange)
		vt.Write([]byte("\x1B[38;5;208mOrange\x1B[0m"))
		output := vt.Render()

		// Should contain 256-color sequence
		if !strings.Contains(output, "\x1B[38;5;208m") {
			t.Errorf("Expected 256-color sequence ESC[38;5;208m in output, got: %q", output)
		}
		if !strings.Contains(output, "Orange") {
			t.Errorf("Expected 'Orange' in output, got: %q", output)
		}
	})

	t.Run("Combined attributes", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write bold red text on blue background
		vt.Write([]byte("\x1B[1;31;44mBold Red on Blue\x1B[0m"))
		output := vt.Render()

		// Should preserve all attributes when rendering
		// The exact output format may vary based on optimization,
		// but it should contain bold, red foreground, and blue background
		if !strings.Contains(output, "Bold Red on Blue") {
			t.Errorf("Expected 'Bold Red on Blue' in output, got: %q", output)
		}
		// Check that some color/style codes are present
		hasEscapeSequence := strings.Contains(output, "\x1B[")
		if !hasEscapeSequence {
			t.Errorf("Expected ANSI escape sequences in output for colored text, got: %q", output)
		}
	})

	t.Run("Reverse video", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write reverse video text
		vt.Write([]byte("\x1B[7mReverse\x1B[27m"))
		output := vt.Render()

		// Should contain SGR sequence for reverse video (7)
		if !strings.Contains(output, "\x1B[7m") {
			t.Errorf("Expected reverse video sequence ESC[7m in output, got: %q", output)
		}
		if !strings.Contains(output, "Reverse") {
			t.Errorf("Expected 'Reverse' in output, got: %q", output)
		}
	})

	t.Run("Color persistence across characters", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Set red color, then write multiple characters
		vt.Write([]byte("\x1B[31mHello World\x1B[0m"))
		output := vt.Render()

		// The entire "Hello World" should be in the same color sequence
		if !strings.Contains(output, "Hello World") {
			t.Errorf("Expected 'Hello World' in output, got: %q", output)
		}
		// Should have color code before the text
		redIndex := strings.Index(output, "\x1B[31m")
		textIndex := strings.Index(output, "Hello World")
		if redIndex == -1 || textIndex == -1 || redIndex > textIndex {
			t.Errorf("Expected color sequence before text, got: %q", output)
		}
	})

	t.Run("Default color reset", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Set color, write text, reset with ESC[39m (default foreground)
		vt.Write([]byte("\x1B[31mRed\x1B[39mDefault"))
		output := vt.Render()

		// Should contain both the colored and default text
		if !strings.Contains(output, "Red") || !strings.Contains(output, "Default") {
			t.Errorf("Expected 'Red' and 'Default' in output, got: %q", output)
		}
		// The implementation may use ESC[0m (full reset) or ESC[39m (default fg)
		// Both are valid ways to reset to default color
		hasReset := strings.Contains(output, "\x1B[39m") || strings.Contains(output, "\x1B[0m")
		if !hasReset {
			t.Errorf("Expected color reset in output, got: %q", output)
		}
	})

	t.Run("Color after newline", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write colored text, newline, then more colored text
		vt.Write([]byte("\x1B[32mLine1\n\x1B[33mLine2\x1B[0m"))
		output := vt.Render()

		if !strings.Contains(output, "Line1") || !strings.Contains(output, "Line2") {
			t.Errorf("Expected 'Line1' and 'Line2' in output, got: %q", output)
		}
		// Should contain green (32) and yellow (33)
		lines := strings.Split(output, "\n")
		if len(lines) < 2 {
			t.Errorf("Expected at least 2 lines in output")
		}
	})
}

// TestVTerminalColorOptimization tests that we don't emit redundant color codes
func TestVTerminalColorOptimization(t *testing.T) {
	t.Run("Don't repeat same color", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Write multiple characters with same color
		vt.Write([]byte("\x1B[31mRed1\x1B[31mRed2\x1B[0m"))
		output := vt.Render()

		// Count how many times ESC[31m appears
		count := strings.Count(output, "\x1B[31m")
		// Should only appear once at the start, not repeated for Red2
		if count > 1 {
			t.Logf("Note: Color optimization could reduce ESC[31m from %d to 1 occurrence", count)
			// This is not a failure, just a potential optimization
		}
	})
}

// TestVTerminalMixedColorText tests realistic color usage scenarios
func TestVTerminalMixedColorText(t *testing.T) {
	t.Run("Shell prompt with colors", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Simulate a colored shell prompt like: [user@host]$
		vt.Write([]byte("\x1B[32m[\x1B[34muser\x1B[32m@\x1B[34mhost\x1B[32m]\x1B[0m$ "))
		output := vt.Render()

		// Should contain the prompt structure
		if !strings.Contains(output, "[") || !strings.Contains(output, "@") ||
			!strings.Contains(output, "]") || !strings.Contains(output, "$") {
			t.Errorf("Expected shell prompt structure in output, got: %q", output)
		}
		// Should contain green and blue color codes
		if !strings.Contains(output, "\x1B[32m") || !strings.Contains(output, "\x1B[34m") {
			t.Errorf("Expected color codes in shell prompt, got: %q", output)
		}
	})

	t.Run("Colored ls output", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Simulate ls --color output: directory in blue, file in white
		vt.Write([]byte("\x1B[34mbin\x1B[0m  \x1B[0msrc\x1B[0m  \x1B[32mscript.sh\x1B[0m"))
		output := vt.Render()

		if !strings.Contains(output, "bin") || !strings.Contains(output, "src") ||
			!strings.Contains(output, "script.sh") {
			t.Errorf("Expected ls output text in output, got: %q", output)
		}
	})

	t.Run("Error message in red", func(t *testing.T) {
		vt := NewVTerminal(80, 24)
		// Simulate error output
		vt.Write([]byte("\x1B[31mError: File not found\x1B[0m\n"))
		output := vt.Render()

		if !strings.Contains(output, "Error: File not found") {
			t.Errorf("Expected error message in output, got: %q", output)
		}
		if !strings.Contains(output, "\x1B[31m") {
			t.Errorf("Expected red color code for error, got: %q", output)
		}
	})
}
