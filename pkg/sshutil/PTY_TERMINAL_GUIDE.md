# PTY Terminal Integration Guide

## Overview

The `pkg/sshutil` package now provides a fully functional PTY terminal implementation with the following features:

- **Core Terminal Shell Functionality**: Manages PTY interactions for shell sessions
- **Exit Functionality**: Detects EOF signals (Ctrl+D) and handles graceful termination
- **Scrolling Support**: Integrated scrollback buffer for terminal history
- **Mouse Integration**: Mouse event handling for text selection and copy-paste (Unix/Linux/macOS)
- **Signal Handling**: Properly handles SIGWINCH (resize), SIGINT (Ctrl+C), SIGTERM, and EOF
- **Environment Setup**: Configurable environment variables (PATH, TERM, etc.)
- **Logging and Debugging**: Built-in debug logging for troubleshooting

## Architecture

The new implementation provides two platform-specific files:

- `pty_terminal_unix.go` - Full-featured implementation for Unix-like systems (Linux, macOS, BSD)
- `pty_terminal_windows.go` - Windows-compatible implementation with core features

## Usage

### Basic Usage

```go
import (
    "github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
)

// Create a new PTY terminal with default options
terminal, err := sshutil.NewPTYTerminal(
    stdinWriter,  // io.Writer for stdin
    stdoutReader, // io.Reader for stdout
    stderrReader, // io.Reader for stderr
    nil,          // Use default options
)
if err != nil {
    log.Fatalf("Failed to create PTY terminal: %v", err)
}

// Start the terminal session
if err := terminal.Start(); err != nil {
    log.Printf("Terminal session error: %v", err)
}

// Clean up
terminal.Close()
```

### Advanced Usage with Options

```go
import (
    "github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
)

// Configure terminal options
opts := &sshutil.PTYTerminalOptions{
    Shell: "/bin/zsh",  // Use zsh instead of default shell
    Environment: map[string]string{
        "CUSTOM_VAR": "value",
        "EDITOR":     "vim",
    },
    ScrollbackLines: 20000,  // Increase scrollback buffer
    EnableMouse:     true,   // Enable mouse support
    Debug:           true,   // Enable debug logging
}

terminal, err := sshutil.NewPTYTerminal(
    stdinWriter,
    stdoutReader,
    stderrReader,
    opts,
)
if err != nil {
    log.Fatalf("Failed to create PTY terminal: %v", err)
}

// Set additional environment variables after creation
terminal.SetEnvironment("LC_ALL", "en_US.UTF-8")

// Start the terminal
if err := terminal.Start(); err != nil {
    log.Printf("Terminal session error: %v", err)
}

// Clean up
terminal.Close()
```

### Accessing Scrollback History

```go
// Get the scrollback buffer
scrollback := terminal.GetScrollback()

// Retrieve all lines
lines := scrollback.GetLines()
for _, line := range lines {
    fmt.Printf("History: %s\n", string(line))
}

// Clear the scrollback buffer
scrollback.Clear()
```

### Querying Terminal Size

```go
width, height := terminal.GetSize()
fmt.Printf("Terminal size: %dx%d\n", width, height)
```

## Integration with SSH Sessions

The PTY terminal can be integrated with SSH sessions from the `internal/ssh` package:

```go
import (
    "github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
    "github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
)

// Create SSH session
sshSession, err := ssh.NewSession(connConfig)
if err != nil {
    log.Fatalf("Failed to create SSH session: %v", err)
}

// Create PTY terminal using SSH session I/O streams
terminal, err := sshutil.NewPTYTerminal(
    sshSession.Stdin(),
    sshSession.Stdout(),
    sshSession.Stderr(),
    &sshutil.PTYTerminalOptions{
        EnableMouse: true,
        Debug:       false,
    },
)
if err != nil {
    log.Fatalf("Failed to create terminal: %v", err)
}

// Start the terminal (blocks until session ends)
if err := terminal.Start(); err != nil {
    log.Printf("Terminal error: %v", err)
}

// Clean up
terminal.Close()
sshSession.Close()
```

## Features in Detail

### 1. EOF Detection

The terminal automatically detects EOF signals:
- **Unix/Linux/macOS**: Ctrl+D (ASCII 4)
- **Windows**: Ctrl+D or Ctrl+Z (ASCII 4 or 26)

When multiple EOF signals are detected, the session terminates gracefully.

### 2. Signal Handling

The following signals are handled:

- **SIGWINCH**: Automatic terminal resize detection and handling
- **SIGINT (Ctrl+C)**: Forwarded to the remote session (doesn't terminate local terminal)
- **SIGTERM**: Clean shutdown of the terminal session

### 3. Scrollback Buffer

The scrollback buffer captures all terminal output line-by-line:

```go
scrollback := terminal.GetScrollback()

// Get all lines
lines := scrollback.GetLines()

// Clear the buffer
scrollback.Clear()

// The buffer automatically maintains a maximum number of lines
// (default: 10,000 lines, configurable via PTYTerminalOptions)
```

### 4. Mouse Support (Unix/Linux/macOS)

When enabled, the terminal supports:
- X10 compatibility mode
- Mouse button event tracking
- SGR extended mouse mode

This enables text selection and copy-paste functionality in compatible terminal emulators.

### 5. Environment Variables

The terminal sets up a default environment with:
- `TERM=xterm-256color`
- `COLORTERM=truecolor`
- `PATH` (inherited or default)
- `HOME` (inherited)

Additional variables can be set via options or the `SetEnvironment()` method.

### 6. Debug Logging

Enable debug mode to see detailed logging:

```go
opts := &sshutil.PTYTerminalOptions{
    Debug: true,
}
```

Debug logs include:
- Terminal initialization
- Signal events
- I/O operations
- Error conditions
- Session lifecycle events

## Platform Differences

### Unix/Linux/macOS (`pty_terminal_unix.go`)
- Full PTY support
- Complete signal handling (SIGWINCH, SIGINT, SIGTERM)
- Mouse support with X10 and SGR modes
- Shell process management

### Windows (`pty_terminal_windows.go`)
- Core terminal functionality
- Limited mouse support
- Basic signal handling
- Ctrl+Z support for EOF

## Thread Safety

The PTY terminal implementation is thread-safe:
- Scrollback buffer uses mutexes for concurrent access
- Signal handling uses channels and context for cancellation
- I/O operations use goroutines with proper synchronization

## Error Handling

The terminal handles various error conditions:
- I/O errors (EOF, connection closed)
- Signal errors
- Terminal state errors

All errors are logged (when debug mode is enabled) and handled gracefully.

## Best Practices

1. **Always call Close()**: Ensure proper cleanup by calling `Close()` when done
2. **Use context**: The terminal uses context for cancellation - respect the context lifecycle
3. **Check errors**: Always check error returns from `NewPTYTerminal()` and `Start()`
4. **Enable debug mode during development**: Use `Debug: true` to troubleshoot issues
5. **Configure scrollback size**: Adjust `ScrollbackLines` based on memory constraints
6. **Set appropriate environment**: Configure `PATH` and other variables as needed

## Example: Complete SSH Terminal

```go
package main

import (
    "log"
    "os"
    
    "github.com/eugeniofciuvasile/ssh-x-term/internal/config"
    "github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
    "github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
)

func main() {
    // Configure SSH connection
    connConfig := config.SSHConnection{
        Host:     "example.com",
        Port:     22,
        Username: "user",
        UsePassword: false,
        KeyFile:  "~/.ssh/id_rsa",
    }
    
    // Create SSH session
    sshSession, err := ssh.NewSession(connConfig)
    if err != nil {
        log.Fatalf("SSH connection failed: %v", err)
    }
    defer sshSession.Close()
    
    // Create PTY terminal
    terminal, err := sshutil.NewPTYTerminal(
        sshSession.Stdin(),
        sshSession.Stdout(),
        sshSession.Stderr(),
        &sshutil.PTYTerminalOptions{
            Shell:           "/bin/bash",
            ScrollbackLines: 10000,
            EnableMouse:     true,
            Debug:           false,
            Environment: map[string]string{
                "LANG": "en_US.UTF-8",
            },
        },
    )
    if err != nil {
        log.Fatalf("Terminal creation failed: %v", err)
    }
    defer terminal.Close()
    
    // Start the terminal (blocks until session ends)
    log.Println("Starting SSH terminal session...")
    if err := terminal.Start(); err != nil {
        log.Printf("Terminal session ended with error: %v", err)
        os.Exit(1)
    }
    
    log.Println("SSH terminal session ended successfully")
}
```

## Troubleshooting

### Terminal doesn't respond to input
- Check that stdin/stdout/stderr pipes are properly connected
- Verify terminal is in raw mode
- Enable debug logging to see I/O operations

### Resize not working
- Ensure SIGWINCH is being delivered to the process
- Check that the terminal file descriptor is valid
- Verify terminal size can be retrieved with `term.GetSize()`

### Mouse not working
- Mouse support is only available on Unix-like systems
- Verify your terminal emulator supports mouse tracking
- Check that `EnableMouse` is set to `true`

### Scrollback not capturing output
- Scrollback captures line-by-line (on newline characters)
- Check buffer size isn't too small
- Verify output contains newline characters

## Future Enhancements

Potential areas for future development:
- Bidirectional scrollback navigation
- Search within scrollback buffer
- Configurable line buffering modes
- Advanced mouse gesture support
- Copy/paste clipboard integration
- Terminal multiplexing support
