package components

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

// VTerminal represents a virtual terminal emulator that can render ANSI/VT100 sequences
type VTerminal struct {
	width          int
	height         int
	buffer         [][]rune // Terminal buffer [row][col]
	scrollback     [][]rune // Scrollback buffer for scrolling
	cursorX        int
	cursorY        int
	scrollOffset   int // How many lines scrolled back
	maxScrollback  int
	mutex          sync.RWMutex
	savedCursorX   int
	savedCursorY   int
	inEscapeSeq    bool
	escapeSeq      []byte
	attrs          cellAttrs
	defaultAttrs   cellAttrs
}

type cellAttrs struct {
	fgColor int
	bgColor int
	bold    bool
	reverse bool
}

// NewVTerminal creates a new virtual terminal with specified dimensions
func NewVTerminal(width, height int) *VTerminal {
	vt := &VTerminal{
		width:         width,
		height:        height,
		maxScrollback: 10000,
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
	vt.buffer = make([][]rune, vt.height)
	for i := range vt.buffer {
		vt.buffer[i] = make([]rune, vt.width)
		for j := range vt.buffer[i] {
			vt.buffer[i][j] = ' '
		}
	}
	vt.scrollback = make([][]rune, 0, vt.maxScrollback)
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
	// Handle escape sequences
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
		vt.inEscapeSeq = true
		vt.escapeSeq = []byte{b}
		return
	}

	// Handle control characters
	switch b {
	case '\r': // Carriage return
		vt.cursorX = 0
	case '\n': // Line feed
		vt.newLine()
	case '\b': // Backspace
		if vt.cursorX > 0 {
			vt.cursorX--
		}
	case '\t': // Tab
		vt.cursorX = (vt.cursorX + 8) & ^7
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}
	case 0x07: // Bell - ignore
	default:
		if b >= 32 { // Printable character
			vt.putChar(rune(b))
		}
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
		vt.buffer[vt.cursorY][vt.cursorX] = r
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
		vt.buffer[vt.height-1] = make([]rune, vt.width)
		for i := range vt.buffer[vt.height-1] {
			vt.buffer[vt.height-1][i] = ' '
		}
		vt.cursorY = vt.height - 1
	}
}

func (vt *VTerminal) isEscapeComplete() bool {
	if len(vt.escapeSeq) < 2 {
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

	// Other simple sequences: ESC <char>
	return len(vt.escapeSeq) >= 2
}

func (vt *VTerminal) handleEscapeSequence() {
	if len(vt.escapeSeq) < 2 {
		return
	}

	// CSI sequences
	if vt.escapeSeq[1] == '[' {
		vt.handleCSI()
		return
	}

	// Simple escape sequences
	switch vt.escapeSeq[1] {
	case 'M': // Reverse index (move up)
		if vt.cursorY > 0 {
			vt.cursorY--
		}
	case '7': // Save cursor position
		vt.savedCursorX = vt.cursorX
		vt.savedCursorY = vt.cursorY
	case '8': // Restore cursor position
		vt.cursorX = vt.savedCursorX
		vt.cursorY = vt.savedCursorY
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
					vt.buffer[vt.cursorY][x] = ' '
				}
			}
			// Clear all lines below
			for y := vt.cursorY + 1; y < vt.height; y++ {
				for x := 0; x < vt.width; x++ {
					if y < len(vt.buffer) && x < len(vt.buffer[y]) {
						vt.buffer[y][x] = ' '
					}
				}
			}
		}
	case 1: // Erase from start of screen to cursor
		// Clear all lines above
		for y := 0; y < vt.cursorY; y++ {
			for x := 0; x < vt.width; x++ {
				if y < len(vt.buffer) && x < len(vt.buffer[y]) {
					vt.buffer[y][x] = ' '
				}
			}
		}
		// Clear current line from start to cursor
		if vt.cursorY < len(vt.buffer) {
			for x := 0; x <= vt.cursorX; x++ {
				if x < len(vt.buffer[vt.cursorY]) {
					vt.buffer[vt.cursorY][x] = ' '
				}
			}
		}
	case 2, 3: // Erase entire screen
		for y := 0; y < vt.height; y++ {
			for x := 0; x < vt.width; x++ {
				if y < len(vt.buffer) && x < len(vt.buffer[y]) {
					vt.buffer[y][x] = ' '
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
				vt.buffer[vt.cursorY][x] = ' '
			}
		}
	case 1: // Erase from start of line to cursor
		for x := 0; x <= vt.cursorX; x++ {
			if x < len(vt.buffer[vt.cursorY]) {
				vt.buffer[vt.cursorY][x] = ' '
			}
		}
	case 2: // Erase entire line
		for x := 0; x < vt.width; x++ {
			if x < len(vt.buffer[vt.cursorY]) {
				vt.buffer[vt.cursorY][x] = ' '
			}
		}
	}
}

func (vt *VTerminal) handleSGR(args []int) {
	if len(args) == 0 {
		args = []int{0}
	}

	for _, arg := range args {
		switch arg {
		case 0: // Reset
			vt.attrs = vt.defaultAttrs
		case 1: // Bold
			vt.attrs.bold = true
		case 7: // Reverse
			vt.attrs.reverse = true
		case 22: // Not bold
			vt.attrs.bold = false
		case 27: // Not reverse
			vt.attrs.reverse = false
		// Foreground colors 30-37
		case 30, 31, 32, 33, 34, 35, 36, 37:
			vt.attrs.fgColor = arg - 30
		case 39: // Default foreground
			vt.attrs.fgColor = -1
		// Background colors 40-47
		case 40, 41, 42, 43, 44, 45, 46, 47:
			vt.attrs.bgColor = arg - 40
		case 49: // Default background
			vt.attrs.bgColor = -1
		}
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

	startLine := 0
	endLine := vt.height

	// If scrolled back, show scrollback content
	if vt.scrollOffset > 0 {
		scrollbackLen := len(vt.scrollback)
		if vt.scrollOffset > scrollbackLen {
			vt.scrollOffset = scrollbackLen
		}

		// Show scrollback lines
		scrollbackStart := scrollbackLen - vt.scrollOffset
		for i := scrollbackStart; i < scrollbackLen && (i-scrollbackStart) < vt.height; i++ {
			buf.WriteString(string(vt.scrollback[i]))
			buf.WriteRune('\n')
		}

		// Fill remaining lines with buffer if needed
		remainingLines := vt.height - (scrollbackLen - scrollbackStart)
		for i := 0; i < remainingLines && i < len(vt.buffer); i++ {
			buf.WriteString(string(vt.buffer[i]))
			buf.WriteRune('\n')
		}
	} else {
		// Show current buffer
		for i := startLine; i < endLine && i < len(vt.buffer); i++ {
			buf.WriteString(string(vt.buffer[i]))
			buf.WriteRune('\n')
		}
	}

	return buf.String()
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
	vt.initBuffer()
	vt.cursorX = 0
	vt.cursorY = 0
	vt.scrollOffset = 0
}
