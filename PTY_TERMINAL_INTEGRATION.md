# PTY Terminal Integration - Technical Documentation

## Overview
The SSH-X-Term application now integrates a proper PTY (pseudo-terminal) terminal emulator within the Bubble Tea TUI framework, enabling full terminal functionality including mouse support, proper ANSI escape sequence handling, and terminal emulation features.

## Architecture

### Previous Implementation
The old implementation (`terminal_old.go.bak`) used `term.MakeRaw()` to take over the entire terminal, which:
- Exited Bubble Tea's alt screen
- Prevented mouse event handling within Bubble Tea
- Could not support copy/paste operations in the TUI
- Made terminal emulation challenging

### New Implementation

#### Components

1. **PTYSession** (`internal/ssh/pty_session_unix.go` & `pty_session_windows.go`)
   - Cross-platform SSH PTY session management
   - Provides `Read()` and `Write()` methods for I/O
   - Supports terminal resize via `Resize(width, height)` method
   - Properly manages SSH session lifecycle

2. **PTYTerminalComponent** (`internal/ui/components/pty_terminal.go`)
   - Integrates vt10x terminal emulator within Bubble Tea
   - Renders terminal output in Bubble Tea's View()
   - Forwards keyboard events from Bubble Tea to SSH session
   - Updates at 60fps for smooth rendering
   - Handles window resize events
   - Supports ESC key to disconnect

3. **vt10x Terminal Emulator** (external library)
   - Provides VT100/ANSI terminal emulation
   - Parses ANSI escape sequences
   - Maintains terminal state and screen buffer
   - Supports colors, formatting, cursor positioning

## Data Flow

```
User Input → Bubble Tea KeyMsg → PTYTerminalComponent.Update()
    ↓
writeToSession() → converts keys to ANSI sequences
    ↓
PTYSession.Write() → sends to SSH session
    ↓
SSH Server processes input
    ↓
SSH Server sends output
    ↓
PTYSession.Read() ← reads from SSH session
    ↓
readFromSession() goroutine ← continuously reads
    ↓
vt10x.Terminal.Write() → parses ANSI, updates screen buffer
    ↓
TerminalOutputMsg triggered (60fps)
    ↓
PTYTerminalComponent.View() → renderTerminalContent()
    ↓
Reads vt10x screen buffer cell by cell
    ↓
Bubble Tea renders to screen
```

## Key Features

### 1. ANSI Escape Sequence Support
- Full VT100 terminal emulation via vt10x
- Supports colors, bold, underline, etc.
- Cursor positioning and movement
- Screen clearing and scrolling

### 2. Mouse Support Foundation
- Terminal renders within Bubble Tea's event loop
- Mouse events (scroll, click, selection) can be handled
- Foundation for copy/paste operations
- Currently forwards all events to terminal view

### 3. Window Resize Handling
- Detects Bubble Tea WindowSizeMsg events
- Resizes vt10x terminal emulator
- Sends resize signal to SSH PTY
- Smooth terminal size changes

### 4. Cross-Platform Support
- Separate implementations for Unix/Darwin and Windows
- Uses platform-specific SSH terminal handling
- Consistent API across platforms

### 5. Performance
- 60fps update rate (16ms tick)
- Non-blocking I/O with goroutines
- Efficient screen buffer rendering
- Minimal CPU usage with proper timing

## Usage

### Connection Flow
1. User selects SSH connection with "isNewWindow = false"
2. `handleSelectedConnection()` creates `PTYTerminalComponent`
3. Component initializes with `Init()` which starts `startPTYSessionCmd`
4. `NewPTYSession()` creates SSH connection with PTY
5. SSH session starts shell
6. `readFromSession()` goroutine continuously reads output
7. Output feeds into vt10x terminal emulator
8. Terminal updates trigger view re-renders at 60fps
9. User can type, interact with terminal
10. Press ESC to disconnect and return to connection list

### Key Bindings
- **Any key**: Forwards to SSH session (converted to ANSI)
- **Arrow keys**: Converted to ANSI escape sequences (↑ = `\x1b[A`, etc.)
- **Enter**: Sends carriage return (`\r`)
- **Ctrl+C**: Sends interrupt (`\x03`)
- **ESC**: Disconnects from SSH session, returns to menu

## Benefits Over Previous Implementation

1. **Stays in Bubble Tea**: No raw terminal takeover
2. **Mouse Support**: Can handle mouse events within TUI
3. **Better Terminal Emulation**: vt10x provides proper VT100 support
4. **Smooth Rendering**: 60fps updates for responsive feel
5. **Cross-Platform**: Works on Unix/Darwin/Windows
6. **Better Integration**: Fits naturally in Bubble Tea's architecture

## Limitations & Future Enhancements

### Current Limitations
1. Mouse events not yet fully implemented (foundation is there)
2. Copy/paste not yet implemented (can be added)
3. Advanced terminal features may need vt10x improvements

### Potential Enhancements
1. Implement mouse click handling
2. Add text selection with mouse
3. Implement copy/paste operations
4. Add scrollback buffer support
5. Support more terminal features (tabs, etc.)
6. Add terminal bell support
7. Improve performance for very fast output

## Testing

### Manual Testing Steps
1. Build: `go build -o sxt ./cmd/sxt`
2. Run: `./sxt`
3. Select local storage
4. Add an SSH connection
5. Disable "Open in new window" (press 'o')
6. Connect to the SSH server
7. Verify terminal renders correctly
8. Test keyboard input
9. Test window resize
10. Press ESC to disconnect

### Verification Points
- [ ] Terminal displays SSH connection correctly
- [ ] Can type commands and see output
- [ ] Colors and formatting render correctly
- [ ] Arrow keys work for command history
- [ ] Ctrl+C sends interrupt
- [ ] Terminal resizes with window
- [ ] ESC disconnects cleanly
- [ ] Can reconnect after disconnect

## Dependencies

- `github.com/hinshun/vt10x`: VT100 terminal emulator
  - Version: 0.0.0-20220301184237-5011da428d02
  - No known vulnerabilities
  - Provides terminal state management and ANSI parsing

## Security Considerations

- SSH credentials handled securely (via keyring or Bitwarden)
- No plaintext password storage
- Terminal output not logged
- Proper cleanup on disconnect
- CodeQL analysis passed with no alerts

## Conclusion

The PTY terminal integration successfully addresses the requirements:
✅ Integrated terminal component within Bubble Tea
✅ Proper PTY terminal with ANSI support
✅ Cross-platform (Unix/Darwin/Windows)
✅ Foundation for mouse support
✅ Better terminal emulation than raw mode
✅ No security vulnerabilities

The implementation provides a solid foundation for a full-featured terminal experience within the SSH-X-Term TUI application.
