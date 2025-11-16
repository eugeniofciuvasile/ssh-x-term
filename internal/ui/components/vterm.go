package components

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/atotto/clipboard"
)

// VTerminal represents a virtual terminal emulator that can render ANSI/VT100 sequences
type VTerminal struct {
	width         int
	height        int
	buffer        [][]cell // Terminal buffer [row][col]
	scrollback    [][]cell // Scrollback buffer for scrolling
	cursorX       int
	cursorY       int
	scrollOffset  int // How many lines scrolled back
	maxScrollback int
	mutex         sync.RWMutex
	savedCursorX  int
	savedCursorY  int
	inEscapeSeq   bool
	escapeSeq     []byte
	attrs         cellAttrs
	defaultAttrs  cellAttrs
	// UTF-8 decoding state
	utf8Buf     []byte // Buffer for incomplete UTF-8 sequences
	utf8BufSize int    // Current size of UTF-8 buffer
	// Mouse selection support
	selectionStart *position
	selectionEnd   *position
}

type position struct {
	x int
	y int
}

type cellAttrs struct {
	fgColor int
	bgColor int
	bold    bool
	reverse bool
}

// cell represents a single terminal cell with character and attributes
type cell struct {
	char  rune
	attrs cellAttrs
}

// NewVTerminal creates a new virtual terminal with specified dimensions
func NewVTerminal(width, height int) *VTerminal {
	// Ensure minimum dimensions
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}

	vt := &VTerminal{
		width:         width,
		height:        height,
		maxScrollback: 10000,
		utf8Buf:       make([]byte, 4), // UTF-8 characters can be up to 4 bytes
		defaultAttrs: cellAttrs{
			fgColor: -1,
			bgColor: -1,
		},
	}
	vt.attrs = vt.defaultAttrs
	vt.initBuffer()
	return vt
}

func (vt *VTerminal) initBuffer() {
	vt.buffer = make([][]cell, vt.height)
	for i := range vt.buffer {
		vt.buffer[i] = make([]cell, vt.width)
		for j := range vt.buffer[i] {
			vt.buffer[i][j] = cell{char: ' ', attrs: vt.defaultAttrs}
		}
	}
	vt.scrollback = make([][]cell, 0, vt.maxScrollback)
}

// Resize changes the terminal dimensions
func (vt *VTerminal) Resize(width, height int) {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()

	if width == vt.width && height == vt.height {
		return
	}

	vt.width = width
	vt.height = height
	vt.initBuffer()
	vt.cursorX = 0
	vt.cursorY = 0
	vt.scrollOffset = 0
}

// Write processes incoming data and updates the terminal buffer
func (vt *VTerminal) Write(data []byte) (int, error) {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()

	for _, b := range data {
		vt.processByte(b)
	}
	return len(data), nil
}

func (vt *VTerminal) processByte(b byte) {
	// Handle escape sequences (these take priority over UTF-8 decoding)
	if vt.inEscapeSeq {
		vt.escapeSeq = append(vt.escapeSeq, b)
		if vt.isEscapeComplete() {
			vt.handleEscapeSequence()
			vt.inEscapeSeq = false
			vt.escapeSeq = nil
		}
		return
	}

	// Start of escape sequence
	if b == 0x1B { // ESC
		// Reset any incomplete UTF-8 sequence when starting escape sequence
		vt.utf8BufSize = 0
		vt.inEscapeSeq = true
		vt.escapeSeq = []byte{b}
		return
	}

	// Handle control characters (0x00-0x1F, 0x7F)
	if b < 0x20 || b == 0x7F {
		// Reset any incomplete UTF-8 sequence when receiving control character
		vt.utf8BufSize = 0

		switch b {
		case 0x00: // NUL - ignore
			// Do nothing
		case 0x0D: // Carriage return (CR, CTRL+M, '\r')
			vt.cursorX = 0
		case 0x0A: // Line feed (LF, CTRL+J, '\n')
			vt.newLine()
		case 0x08: // Backspace (BS, CTRL+H, '\b')
			if vt.cursorX > 0 {
				vt.cursorX--
			}
		case 0x09: // Tab (HT, CTRL+I, '\t')
			vt.cursorX = (vt.cursorX + 8) & ^7
			if vt.cursorX >= vt.width {
				vt.cursorX = vt.width - 1
			}
		case 0x07: // Bell (BEL, CTRL+G) - ignore
			// Do nothing - bell sound not supported in TUI
		case 0x0C: // Form feed (FF, CTRL+L) - typically used for clear screen
			// Some programs use FF as clear screen, we'll just move to new line
			vt.newLine()
		case 0x0B: // Vertical tab (VT) - treat as line feed
			vt.newLine()
		case 0x0E, 0x0F: // Shift Out/Shift In - character set switching, ignore for now
			// These are used for alternate character sets, not commonly used in modern terminals
		case 0x7F: // DEL - delete character (usually same as backspace)
			if vt.cursorX > 0 {
				vt.cursorX--
			}
		}
		// Ignore other control characters (0x01-0x06, 0x10-0x1A, 0x1C-0x1F) that we don't explicitly handle
		return
	}

	// Handle printable characters and UTF-8 sequences
	// Check if this is a single-byte ASCII character (0x20-0x7E)
	if b < 0x80 {
		// Reset any incomplete UTF-8 sequence
		vt.utf8BufSize = 0
		vt.putChar(rune(b))
		return
	}

	// Multi-byte UTF-8 character handling
	// Add byte to UTF-8 buffer
	if vt.utf8BufSize < len(vt.utf8Buf) {
		vt.utf8Buf[vt.utf8BufSize] = b
		vt.utf8BufSize++
	} else {
		// Buffer overflow, reset and treat as invalid
		vt.utf8BufSize = 1
		vt.utf8Buf[0] = b
	}

	// Try to decode UTF-8 sequence
	r, size := utf8.DecodeRune(vt.utf8Buf[:vt.utf8BufSize])

	if r == utf8.RuneError {
		if vt.utf8BufSize >= 4 {
			// We have 4 bytes but still can't decode - invalid UTF-8
			// Reset buffer and skip this sequence
			vt.utf8BufSize = 0
		}
		// Otherwise, wait for more bytes
		return
	}

	// Successfully decoded a rune
	if size > 0 {
		vt.putChar(r)
		// Reset UTF-8 buffer
		vt.utf8BufSize = 0
	}
}

func (vt *VTerminal) putChar(r rune) {
	if vt.cursorY >= vt.height {
		vt.cursorY = vt.height - 1
	}
	if vt.cursorX >= vt.width {
		vt.newLine()
		vt.cursorX = 0
	}

	if vt.cursorY < len(vt.buffer) && vt.cursorX < len(vt.buffer[vt.cursorY]) {
		vt.buffer[vt.cursorY][vt.cursorX] = cell{char: r, attrs: vt.attrs}
	}
	vt.cursorX++
}

func (vt *VTerminal) newLine() {
	vt.cursorY++
	if vt.cursorY >= vt.height {
		// Scroll: move first line to scrollback
		if len(vt.scrollback) >= vt.maxScrollback {
			vt.scrollback = vt.scrollback[1:]
		}
		vt.scrollback = append(vt.scrollback, vt.buffer[0])

		// Shift buffer up
		copy(vt.buffer, vt.buffer[1:])
		vt.buffer[vt.height-1] = make([]cell, vt.width)
		for i := range vt.buffer[vt.height-1] {
			vt.buffer[vt.height-1][i] = cell{char: ' ', attrs: vt.defaultAttrs}
		}
		vt.cursorY = vt.height - 1
	}
}

func (vt *VTerminal) isEscapeComplete() bool {
	if len(vt.escapeSeq) < 2 {
		return false
	}

	// OSC sequences: ESC ] ... BEL or ESC \
	if len(vt.escapeSeq) >= 2 && vt.escapeSeq[1] == ']' {
		// OSC terminated by BEL (0x07)
		if vt.escapeSeq[len(vt.escapeSeq)-1] == 0x07 {
			return true
		}
		// OSC terminated by ST (ESC \)
		if len(vt.escapeSeq) >= 2 {
			if vt.escapeSeq[len(vt.escapeSeq)-2] == 0x1B && vt.escapeSeq[len(vt.escapeSeq)-1] == '\\' {
				return true
			}
		}
		// Prevent infinite growth
		if len(vt.escapeSeq) > 1000 {
			return true
		}
		return false
	}

	// CSI sequences: ESC [ ... [a-zA-Z]
	if len(vt.escapeSeq) >= 2 && vt.escapeSeq[1] == '[' {
		lastByte := vt.escapeSeq[len(vt.escapeSeq)-1]
		// Check if it's a letter (terminator)
		if (lastByte >= 'A' && lastByte <= 'Z') || (lastByte >= 'a' && lastByte <= 'z') {
			return true
		}
		// Prevent infinite growth
		if len(vt.escapeSeq) > 100 {
			return true
		}
		return false
	}

	// Character set designation sequences: ESC ( <char>, ESC ) <char>, ESC * <char>, ESC + <char>
	if len(vt.escapeSeq) >= 2 {
		secondByte := vt.escapeSeq[1]
		if secondByte == '(' || secondByte == ')' || secondByte == '*' || secondByte == '+' {
			// Need 3 bytes for character set designation
			return len(vt.escapeSeq) >= 3
		}
	}

	// Other simple sequences: ESC <char>
	return len(vt.escapeSeq) >= 2
}

func (vt *VTerminal) handleEscapeSequence() {
	if len(vt.escapeSeq) < 2 {
		return
	}

	// OSC sequences - just ignore them (e.g., window title changes)
	if vt.escapeSeq[1] == ']' {
		// Silently ignore OSC sequences
		return
	}

	// CSI sequences
	if vt.escapeSeq[1] == '[' {
		vt.handleCSI()
		return
	}

	// Character set designation sequences: ESC ( <char>, ESC ) <char>, ESC * <char>, ESC + <char>
	// These are used to switch between character sets (ASCII, line drawing, etc.)
	// For example: ESC(B = select ASCII for G0, ESC(0 = select line drawing for G0
	if len(vt.escapeSeq) >= 3 {
		secondByte := vt.escapeSeq[1]
		if secondByte == '(' || secondByte == ')' || secondByte == '*' || secondByte == '+' {
			// Character set designation - silently ignore for now
			// In a full implementation, we would track the character set and use it
			// for proper rendering of line drawing characters (box drawing, etc.)
			// Common values: 'B' = ASCII, '0' = DEC Special Graphics, 'A' = UK charset
			return
		}
	}

	// Simple escape sequences
	switch vt.escapeSeq[1] {
	case 'M': // Reverse index (move up)
		if vt.cursorY > 0 {
			vt.cursorY--
		}
	case '7': // Save cursor position (DECSC)
		vt.savedCursorX = vt.cursorX
		vt.savedCursorY = vt.cursorY
	case '8': // Restore cursor position (DECRC)
		vt.cursorX = vt.savedCursorX
		vt.cursorY = vt.savedCursorY
	case 'D': // Index (move down, scroll if at bottom) - IND
		vt.newLine()
	case 'E': // Next line (CR + LF) - NEL
		vt.cursorX = 0
		vt.newLine()
	case 'H': // Tab set - HTS (ignore for now)
		// Would set a tab stop at current cursor position
	case 'c': // Reset (RIS) - full reset
		vt.clearInternal() // Use internal version to avoid deadlock
	}
}

func (vt *VTerminal) handleCSI() {
	if len(vt.escapeSeq) < 3 {
		return
	}

	// Extract command and parameters
	seq := string(vt.escapeSeq[2:])
	if len(seq) == 0 {
		return
	}

	cmd := seq[len(seq)-1]
	params := seq[:len(seq)-1]

	// Parse numeric parameters
	args := parseCSIParams(params)

	switch cmd {
	case 'H', 'f': // Cursor position
		row, col := 1, 1
		if len(args) > 0 {
			row = args[0]
		}
		if len(args) > 1 {
			col = args[1]
		}
		vt.cursorY = row - 1
		vt.cursorX = col - 1
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}
		if vt.cursorY >= vt.height {
			vt.cursorY = vt.height - 1
		}
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}

	case 'A': // Cursor up
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.cursorY -= n
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}

	case 'B': // Cursor down
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.cursorY += n
		if vt.cursorY >= vt.height {
			vt.cursorY = vt.height - 1
		}

	case 'C': // Cursor forward
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.cursorX += n
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}

	case 'D': // Cursor backward
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.cursorX -= n
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}

	case 'J': // Erase display
		mode := 0
		if len(args) > 0 {
			mode = args[0]
		}
		vt.eraseDisplay(mode)

	case 'K': // Erase line
		mode := 0
		if len(args) > 0 {
			mode = args[0]
		}
		vt.eraseLine(mode)

	case 'L': // Insert lines
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.insertLines(n)

	case 'M': // Delete lines
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.deleteLines(n)

	case 'P': // Delete characters
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.deleteChars(n)

	case 'X': // Erase characters (without moving cursor)
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.eraseChars(n)

	case 'd': // Vertical position absolute
		row := 1
		if len(args) > 0 && args[0] > 0 {
			row = args[0]
		}
		vt.cursorY = max(row-1, 0)
		if vt.cursorY >= vt.height {
			vt.cursorY = vt.height - 1
		}

	case 'G', '`': // Horizontal position absolute
		col := 1
		if len(args) > 0 && args[0] > 0 {
			col = args[0]
		}
		vt.cursorX = max(col-1, 0)
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}

	case 's': // Save cursor position (ANSI.SYS)
		vt.savedCursorX = vt.cursorX
		vt.savedCursorY = vt.cursorY

	case 'u': // Restore cursor position (ANSI.SYS)
		vt.cursorX = vt.savedCursorX
		vt.cursorY = vt.savedCursorY

	case '@': // Insert blank characters (ICH)
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		vt.insertChars(n)

	case 'S': // Scroll up (SU)
		n := 1
		if len(args) > 0 && args[0] > 0 {
			n = args[0]
		}
		// Scroll the screen up by n lines
		for i := 0; i < n; i++ {
			vt.newLine()
		}

	case 'T': // Scroll down (SD)
		// Scroll down is rarely used and complex to implement
		// Just ignore for now
		_ = args // Suppress unused variable warning

	case 'h': // Set mode
		// Handle various DEC private modes with '?' prefix
		if params != "" && params[0] == '?' {
			// Private mode set - just ignore for now
			// Common ones: ?25 = cursor visible, ?1049 = alternate screen buffer
		}

	case 'l': // Reset mode
		// Handle various DEC private modes with '?' prefix
		if params != "" && params[0] == '?' {
			// Private mode reset - just ignore for now
		}

	case 'm': // SGR - Select Graphic Rendition
		vt.handleSGR(args)
	}
}

func (vt *VTerminal) eraseDisplay(mode int) {
	switch mode {
	case 0: // Erase from cursor to end of screen
		// Clear current line from cursor to end
		if vt.cursorY < len(vt.buffer) {
			for x := vt.cursorX; x < vt.width; x++ {
				if x < len(vt.buffer[vt.cursorY]) {
					vt.buffer[vt.cursorY][x] = cell{char: ' ', attrs: vt.defaultAttrs}
				}
			}
			// Clear all lines below
			for y := vt.cursorY + 1; y < vt.height; y++ {
				for x := 0; x < vt.width; x++ {
					if y < len(vt.buffer) && x < len(vt.buffer[y]) {
						vt.buffer[y][x] = cell{char: ' ', attrs: vt.defaultAttrs}
					}
				}
			}
		}
	case 1: // Erase from start of screen to cursor
		// Clear all lines above
		for y := 0; y < vt.cursorY; y++ {
			for x := 0; x < vt.width; x++ {
				if y < len(vt.buffer) && x < len(vt.buffer[y]) {
					vt.buffer[y][x] = cell{char: ' ', attrs: vt.defaultAttrs}
				}
			}
		}
		// Clear current line from start to cursor
		if vt.cursorY < len(vt.buffer) {
			for x := 0; x <= vt.cursorX; x++ {
				if x < len(vt.buffer[vt.cursorY]) {
					vt.buffer[vt.cursorY][x] = cell{char: ' ', attrs: vt.defaultAttrs}
				}
			}
		}
	case 2, 3: // Erase entire screen
		for y := 0; y < vt.height; y++ {
			for x := 0; x < vt.width; x++ {
				if y < len(vt.buffer) && x < len(vt.buffer[y]) {
					vt.buffer[y][x] = cell{char: ' ', attrs: vt.defaultAttrs}
				}
			}
		}
		if mode == 2 {
			vt.cursorX = 0
			vt.cursorY = 0
		}
	}
}

func (vt *VTerminal) eraseLine(mode int) {
	if vt.cursorY >= len(vt.buffer) {
		return
	}

	switch mode {
	case 0: // Erase from cursor to end of line
		for x := vt.cursorX; x < vt.width; x++ {
			if x < len(vt.buffer[vt.cursorY]) {
				vt.buffer[vt.cursorY][x] = cell{char: ' ', attrs: vt.defaultAttrs}
			}
		}
	case 1: // Erase from start of line to cursor
		for x := 0; x <= vt.cursorX; x++ {
			if x < len(vt.buffer[vt.cursorY]) {
				vt.buffer[vt.cursorY][x] = cell{char: ' ', attrs: vt.defaultAttrs}
			}
		}
	case 2: // Erase entire line
		for x := 0; x < vt.width; x++ {
			if x < len(vt.buffer[vt.cursorY]) {
				vt.buffer[vt.cursorY][x] = cell{char: ' ', attrs: vt.defaultAttrs}
			}
		}
	}
}

func (vt *VTerminal) handleSGR(args []int) {
	if len(args) == 0 {
		args = []int{0}
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case 0: // Reset
			vt.attrs = vt.defaultAttrs
		case 1: // Bold
			vt.attrs.bold = true
		case 2: // Dim/faint (treat as normal for now)
			// Not commonly supported
		case 3: // Italic (ignore for now)
			// Not commonly supported in terminals
		case 4: // Underline (ignore for now)
			// Would require additional attribute tracking
		case 5, 6: // Blink slow/rapid (ignore)
			// Not supported in TUI
		case 7: // Reverse
			vt.attrs.reverse = true
		case 8: // Conceal/hidden (ignore)
			// Not commonly used
		case 9: // Crossed out (ignore)
			// Not commonly supported
		case 21: // Double underline (ignore)
			// Not commonly supported
		case 22: // Not bold, not dim
			vt.attrs.bold = false
		case 23: // Not italic
			// Ignore
		case 24: // Not underlined
			// Ignore
		case 25: // Not blinking
			// Ignore
		case 27: // Not reverse
			vt.attrs.reverse = false
		case 28: // Not concealed
			// Ignore
		case 29: // Not crossed out
			// Ignore
		// Foreground colors 30-37 (standard colors)
		case 30, 31, 32, 33, 34, 35, 36, 37:
			vt.attrs.fgColor = arg - 30
		// Extended foreground color (256-color or RGB)
		case 38:
			if i+1 < len(args) {
				if args[i+1] == 5 && i+2 < len(args) {
					// 256-color: ESC[38;5;Nm
					vt.attrs.fgColor = args[i+2]
					i += 2
				} else if args[i+1] == 2 && i+4 < len(args) {
					// RGB color: ESC[38;2;R;G;Bm
					// For simplicity, map to closest standard color
					// In a full implementation, we'd track RGB values
					i += 4
				}
			}
		case 39: // Default foreground
			vt.attrs.fgColor = -1
		// Background colors 40-47 (standard colors)
		case 40, 41, 42, 43, 44, 45, 46, 47:
			vt.attrs.bgColor = arg - 40
		// Extended background color (256-color or RGB)
		case 48:
			if i+1 < len(args) {
				if args[i+1] == 5 && i+2 < len(args) {
					// 256-color: ESC[48;5;Nm
					vt.attrs.bgColor = args[i+2]
					i += 2
				} else if args[i+1] == 2 && i+4 < len(args) {
					// RGB color: ESC[48;2;R;G;Bm
					i += 4
				}
			}
		case 49: // Default background
			vt.attrs.bgColor = -1
		// Bright foreground colors 90-97
		case 90, 91, 92, 93, 94, 95, 96, 97:
			vt.attrs.fgColor = arg - 90 + 8
		// Bright background colors 100-107
		case 100, 101, 102, 103, 104, 105, 106, 107:
			vt.attrs.bgColor = arg - 100 + 8
		}
		i++
	}
}

// insertLines inserts n blank lines at cursor position, scrolling down
func (vt *VTerminal) insertLines(n int) {
	if vt.cursorY >= vt.height || n <= 0 {
		return
	}

	// Limit n to available space
	if vt.cursorY+n > vt.height {
		n = vt.height - vt.cursorY
	}

	// Move existing lines down
	for i := vt.height - 1; i >= vt.cursorY+n; i-- {
		if i < len(vt.buffer) && i-n < len(vt.buffer) && i-n >= 0 {
			copy(vt.buffer[i], vt.buffer[i-n])
		}
	}

	// Clear the inserted lines
	for i := 0; i < n && vt.cursorY+i < vt.height; i++ {
		if vt.cursorY+i < len(vt.buffer) {
			for x := 0; x < vt.width; x++ {
				if x < len(vt.buffer[vt.cursorY+i]) {
					vt.buffer[vt.cursorY+i][x] = cell{char: ' ', attrs: vt.defaultAttrs}
				}
			}
		}
	}
}

// deleteLines deletes n lines at cursor position, scrolling up
func (vt *VTerminal) deleteLines(n int) {
	if vt.cursorY >= vt.height || n <= 0 {
		return
	}

	// Limit n to available lines
	if vt.cursorY+n > vt.height {
		n = vt.height - vt.cursorY
	}

	// Move lines up
	for i := vt.cursorY; i < vt.height-n; i++ {
		if i < len(vt.buffer) && i+n < len(vt.buffer) {
			copy(vt.buffer[i], vt.buffer[i+n])
		}
	}

	// Clear the bottom lines
	for i := vt.height - n; i < vt.height; i++ {
		if i >= 0 && i < len(vt.buffer) {
			for x := 0; x < vt.width; x++ {
				if x < len(vt.buffer[i]) {
					vt.buffer[i][x] = cell{char: ' ', attrs: vt.defaultAttrs}
				}
			}
		}
	}
}

// deleteChars deletes n characters at cursor position, shifting line left
func (vt *VTerminal) deleteChars(n int) {
	if vt.cursorY >= len(vt.buffer) || n <= 0 {
		return
	}

	line := vt.buffer[vt.cursorY]
	if vt.cursorX >= len(line) {
		return
	}

	// Limit n to remaining characters on line
	if vt.cursorX+n > len(line) {
		n = len(line) - vt.cursorX
	}

	// Shift characters left
	for i := vt.cursorX; i < len(line)-n; i++ {
		line[i] = line[i+n]
	}

	// Fill the end with spaces
	for i := len(line) - n; i < len(line); i++ {
		line[i] = cell{char: ' ', attrs: vt.defaultAttrs}
	}
}

// insertChars inserts n blank characters at cursor position, shifting line right
func (vt *VTerminal) insertChars(n int) {
	if vt.cursorY >= len(vt.buffer) || n <= 0 {
		return
	}

	line := vt.buffer[vt.cursorY]
	if vt.cursorX >= len(line) {
		return
	}

	// Limit n to remaining space on line
	if vt.cursorX+n > len(line) {
		n = len(line) - vt.cursorX
	}

	// Shift characters right
	for i := len(line) - 1; i >= vt.cursorX+n; i-- {
		line[i] = line[i-n]
	}

	// Fill the inserted positions with spaces
	for i := vt.cursorX; i < vt.cursorX+n && i < len(line); i++ {
		line[i] = cell{char: ' ', attrs: vt.defaultAttrs}
	}
}

// eraseChars erases n characters at cursor position (replaces with spaces)
func (vt *VTerminal) eraseChars(n int) {
	if vt.cursorY >= len(vt.buffer) || n <= 0 {
		return
	}

	line := vt.buffer[vt.cursorY]

	// Erase up to n characters or end of line
	for i := 0; i < n && vt.cursorX+i < len(line); i++ {
		line[vt.cursorX+i] = cell{char: ' ', attrs: vt.defaultAttrs}
	}
}

func parseCSIParams(params string) []int {
	if params == "" {
		return nil
	}

	parts := strings.Split(params, ";")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		var num int
		fmt.Sscanf(part, "%d", &num)
		result = append(result, num)
	}

	return result
}

// Render returns the visible terminal content as a string
func (vt *VTerminal) Render() string {
	vt.mutex.RLock()
	defer vt.mutex.RUnlock()

	var buf bytes.Buffer
	linesRendered := 0

	startLine := 0
	endLine := vt.height

	// Determine if we should show the cursor (only when not scrolled back)
	showCursor := vt.scrollOffset == 0 && vt.cursorY >= 0 && vt.cursorY < vt.height && vt.cursorX >= 0 && vt.cursorX < vt.width

	// If scrolled back, show scrollback content
	if vt.scrollOffset > 0 {
		scrollbackLen := len(vt.scrollback)
		if vt.scrollOffset > scrollbackLen {
			vt.scrollOffset = scrollbackLen
		}

		// Show scrollback lines
		scrollbackStart := scrollbackLen - vt.scrollOffset
		for i := scrollbackStart; i < scrollbackLen && (i-scrollbackStart) < vt.height; i++ {
			vt.renderLine(&buf, vt.scrollback[i], false, -1)
			buf.WriteRune('\n')
			linesRendered++
		}

		// Fill remaining lines with buffer if needed
		remainingLines := vt.height - (scrollbackLen - scrollbackStart)
		for i := 0; i < remainingLines && i < len(vt.buffer); i++ {
			vt.renderLine(&buf, vt.buffer[i], false, -1)
			buf.WriteRune('\n')
			linesRendered++
		}
	} else {
		// Show current buffer with cursor
		for i := startLine; i < endLine && i < len(vt.buffer); i++ {
			line := vt.buffer[i]

			// Render line with visual cursor if this is the cursor line
			if showCursor && i == vt.cursorY {
				vt.renderLine(&buf, line, true, vt.cursorX)
			} else {
				vt.renderLine(&buf, line, false, -1)
			}

			buf.WriteRune('\n')
			linesRendered++
		}
	}

	// Fill any remaining lines with empty space to ensure we use the full height
	for linesRendered < vt.height {
		buf.WriteString(strings.Repeat(" ", vt.width))
		buf.WriteRune('\n')
		linesRendered++
	}

	return buf.String()
}

// renderLine renders a single line with color attributes
func (vt *VTerminal) renderLine(buf *bytes.Buffer, line []cell, showCursor bool, cursorX int) {
	var currentAttrs cellAttrs
	currentAttrs.fgColor = -1
	currentAttrs.bgColor = -1

	for j := 0; j < vt.width; j++ {
		var c cell
		if j < len(line) {
			c = line[j]
		} else {
			c = cell{char: ' ', attrs: vt.defaultAttrs}
		}

		// Check if this is the cursor position
		isCursor := showCursor && j == cursorX

		// If cursor position, apply inverse video
		var attrs cellAttrs
		if isCursor {
			attrs = c.attrs
			attrs.reverse = !attrs.reverse
		} else {
			attrs = c.attrs
		}

		// Apply attributes if they changed
		if attrs.fgColor != currentAttrs.fgColor || attrs.bgColor != currentAttrs.bgColor ||
			attrs.bold != currentAttrs.bold || attrs.reverse != currentAttrs.reverse {

			// Reset to default if needed
			if attrs.fgColor == -1 && attrs.bgColor == -1 && !attrs.bold && !attrs.reverse {
				buf.WriteString("\x1B[0m")
			} else {
				// Build SGR sequence
				var sgr []string

				// Handle reverse video
				if attrs.reverse != currentAttrs.reverse {
					if attrs.reverse {
						sgr = append(sgr, "7")
					} else {
						sgr = append(sgr, "27")
					}
				}

				// Handle bold
				if attrs.bold != currentAttrs.bold {
					if attrs.bold {
						sgr = append(sgr, "1")
					} else {
						sgr = append(sgr, "22")
					}
				}

				// Handle foreground color
				if attrs.fgColor != currentAttrs.fgColor {
					if attrs.fgColor == -1 {
						sgr = append(sgr, "39")
					} else if attrs.fgColor < 8 {
						sgr = append(sgr, fmt.Sprintf("%d", 30+attrs.fgColor))
					} else if attrs.fgColor < 16 {
						sgr = append(sgr, fmt.Sprintf("%d", 90+attrs.fgColor-8))
					} else {
						sgr = append(sgr, fmt.Sprintf("38;5;%d", attrs.fgColor))
					}
				}

				// Handle background color
				if attrs.bgColor != currentAttrs.bgColor {
					if attrs.bgColor == -1 {
						sgr = append(sgr, "49")
					} else if attrs.bgColor < 8 {
						sgr = append(sgr, fmt.Sprintf("%d", 40+attrs.bgColor))
					} else if attrs.bgColor < 16 {
						sgr = append(sgr, fmt.Sprintf("%d", 100+attrs.bgColor-8))
					} else {
						sgr = append(sgr, fmt.Sprintf("48;5;%d", attrs.bgColor))
					}
				}

				if len(sgr) > 0 {
					buf.WriteString(fmt.Sprintf("\x1B[%sm", strings.Join(sgr, ";")))
				}
			}

			currentAttrs = attrs
		}

		// Write the character
		buf.WriteRune(c.char)
	}

	// Reset attributes at end of line
	if currentAttrs.fgColor != -1 || currentAttrs.bgColor != -1 || currentAttrs.bold || currentAttrs.reverse {
		buf.WriteString("\x1B[0m")
	}
}

// GetCursorPosition returns the current cursor position
func (vt *VTerminal) GetCursorPosition() (int, int) {
	vt.mutex.RLock()
	defer vt.mutex.RUnlock()
	return vt.cursorX, vt.cursorY
}

// ScrollUp scrolls the view up by n lines
func (vt *VTerminal) ScrollUp(n int) {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()

	vt.scrollOffset += n
	maxScroll := len(vt.scrollback)
	if vt.scrollOffset > maxScroll {
		vt.scrollOffset = maxScroll
	}
}

// ScrollDown scrolls the view down by n lines
func (vt *VTerminal) ScrollDown(n int) {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()

	vt.scrollOffset -= n
	if vt.scrollOffset < 0 {
		vt.scrollOffset = 0
	}
}

// IsScrolledBack returns true if the terminal is scrolled back
func (vt *VTerminal) IsScrolledBack() bool {
	vt.mutex.RLock()
	defer vt.mutex.RUnlock()
	return vt.scrollOffset > 0
}

// ScrollToBottom scrolls to the bottom of the buffer
func (vt *VTerminal) ScrollToBottom() {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()
	vt.scrollOffset = 0
}

// Clear clears the terminal buffer
func (vt *VTerminal) Clear() {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()
	vt.clearInternal()
}

// clearInternal clears the terminal buffer without locking (for internal use)
func (vt *VTerminal) clearInternal() {
	vt.initBuffer()
	vt.cursorX = 0
	vt.cursorY = 0
	vt.scrollOffset = 0
}

// StartSelection begins a text selection at the given position
func (vt *VTerminal) StartSelection(x, y int) {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()
	vt.selectionStart = &position{x: x, y: y}
	vt.selectionEnd = nil
}

// UpdateSelection updates the selection end position
func (vt *VTerminal) UpdateSelection(x, y int) {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()
	if vt.selectionStart != nil {
		vt.selectionEnd = &position{x: x, y: y}
	}
}

// ClearSelection clears the current selection
func (vt *VTerminal) ClearSelection() {
	vt.mutex.Lock()
	defer vt.mutex.Unlock()
	vt.selectionStart = nil
	vt.selectionEnd = nil
}

// CopySelection copies the selected text to the clipboard
func (vt *VTerminal) CopySelection() error {
	vt.mutex.RLock()
	defer vt.mutex.RUnlock()

	if vt.selectionStart == nil || vt.selectionEnd == nil {
		return fmt.Errorf("no selection")
	}

	// Normalize selection (ensure start is before end)
	startY, startX := vt.selectionStart.y, vt.selectionStart.x
	endY, endX := vt.selectionEnd.y, vt.selectionEnd.x

	if startY > endY || (startY == endY && startX > endX) {
		startY, endY = endY, startY
		startX, endX = endX, startX
	}

	var text strings.Builder

	// Single line selection
	if startY == endY {
		if startY < len(vt.buffer) {
			for x := startX; x <= endX && x < len(vt.buffer[startY]); x++ {
				text.WriteRune(vt.buffer[startY][x].char)
			}
		}
	} else {
		// Multi-line selection
		// First line
		if startY < len(vt.buffer) {
			for x := startX; x < len(vt.buffer[startY]); x++ {
				text.WriteRune(vt.buffer[startY][x].char)
			}
			text.WriteRune('\n')
		}

		// Middle lines
		for y := startY + 1; y < endY && y < len(vt.buffer); y++ {
			for x := 0; x < len(vt.buffer[y]); x++ {
				text.WriteRune(vt.buffer[y][x].char)
			}
			text.WriteRune('\n')
		}

		// Last line
		if endY < len(vt.buffer) {
			for x := 0; x <= endX && x < len(vt.buffer[endY]); x++ {
				text.WriteRune(vt.buffer[endY][x].char)
			}
		}
	}

	selectedText := strings.TrimRight(text.String(), " \n")
	return clipboard.WriteAll(selectedText)
}

// HasSelection returns true if there is an active selection
func (vt *VTerminal) HasSelection() bool {
	vt.mutex.RLock()
	defer vt.mutex.RUnlock()
	return vt.selectionStart != nil && vt.selectionEnd != nil
}
