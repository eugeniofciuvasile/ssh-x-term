# pkg/sshutil

This package provides SSH utility functions and a fully functional PTY (pseudo-terminal) terminal implementation for the ssh-x-term project.

## Overview

The `sshutil` package contains utilities for SSH authentication and advanced terminal management with comprehensive PTY support.

## Components

### Authentication (`auth.go`)

Provides SSH authentication utilities:
- `GetKeyAuthMethod()` - SSH key-based authentication
- `GetPasswordAuthMethod()` - Password-based authentication
- `GetAuthMethod()` - Unified authentication method selector

### Terminal Session (`terminal_unix.go`, `terminal_windows.go`)

Basic terminal session management with:
- Raw mode terminal control
- I/O streaming
- Window resize handling (SIGWINCH)
- Cross-platform support

### PTY Terminal (`pty_terminal_unix.go`, `pty_terminal_windows.go`)

**NEW**: Advanced PTY terminal implementation with full terminal emulation support.

#### Features

✅ **Core Terminal Functionality**
- PTY management for interactive shell sessions
- Raw mode terminal support
- Bidirectional I/O streaming with proper buffering

✅ **Exit & Signal Handling**
- EOF detection (Ctrl+D on Unix, Ctrl+D/Ctrl+Z on Windows)
- SIGWINCH: Automatic window resize
- SIGINT (Ctrl+C): Forwarded to remote session
- SIGTERM: Clean shutdown
- Graceful session termination

✅ **Scrollback Buffer**
- Configurable buffer size (default: 10,000 lines)
- Line-by-line output capture
- Thread-safe access with mutex protection
- Efficient memory management

✅ **Mouse Integration** (Unix/Linux/macOS)
- X10 compatibility mode
- Mouse button event tracking
- SGR extended mouse mode
- Text selection and copy-paste support

✅ **Environment Management**
- Default environment (TERM, PATH, HOME)
- Custom environment variables
- Dynamic variable updates

✅ **Logging & Debugging**
- Optional debug mode
- Detailed operation logging
- Error tracking and reporting

✅ **Cross-Platform Support**
- Unix/Linux/macOS: Full feature set
- Windows: Core features with platform adaptations

## Usage

### Basic Example

```go
import "github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"

// Create PTY terminal with default options
terminal, err := sshutil.NewPTYTerminal(
    stdinWriter,
    stdoutReader,
    stderrReader,
    nil, // Use default options
)
if err != nil {
    log.Fatal(err)
}
defer terminal.Close()

// Start the terminal (blocks until session ends)
if err := terminal.Start(); err != nil {
    log.Printf("Session error: %v", err)
}
```

### Advanced Example

```go
// Configure custom options
opts := &sshutil.PTYTerminalOptions{
    Shell: "/bin/zsh",
    Environment: map[string]string{
        "EDITOR": "vim",
        "LANG":   "en_US.UTF-8",
    },
    ScrollbackLines: 20000,
    EnableMouse:     true,
    Debug:           true,
}

terminal, err := sshutil.NewPTYTerminal(stdin, stdout, stderr, opts)
if err != nil {
    log.Fatal(err)
}
defer terminal.Close()

// Set additional environment variable
terminal.SetEnvironment("PATH", "/custom/path:/usr/bin")

// Start terminal
if err := terminal.Start(); err != nil {
    log.Printf("Error: %v", err)
}

// Access scrollback history
scrollback := terminal.GetScrollback()
lines := scrollback.GetLines()
for _, line := range lines {
    fmt.Println(string(line))
}
```

### SSH Integration Example

```go
// Create SSH session
sshSession, err := ssh.NewSession(connConfig)
if err != nil {
    log.Fatal(err)
}
defer sshSession.Close()

// Create PTY terminal using SSH I/O streams
terminal, err := sshutil.NewPTYTerminal(
    sshSession.Stdin(),
    sshSession.Stdout(),
    sshSession.Stderr(),
    &sshutil.PTYTerminalOptions{
        EnableMouse:     true,
        ScrollbackLines: 10000,
        Debug:           false,
    },
)
if err != nil {
    log.Fatal(err)
}
defer terminal.Close()

// Start the interactive terminal
if err := terminal.Start(); err != nil {
    log.Printf("Terminal error: %v", err)
}
```

## Documentation

For detailed documentation and examples, see:

- **[PTY_TERMINAL_GUIDE.md](./PTY_TERMINAL_GUIDE.md)** - Comprehensive integration guide
- **[examples/pty_terminal_example.go](./examples/pty_terminal_example.go)** - Working code examples

## API Reference

### Types

#### `PTYTerminal`
Main terminal type with full PTY support.

**Methods:**
- `NewPTYTerminal(stdin, stdout, stderr, opts) (*PTYTerminal, error)` - Create new terminal
- `Start() error` - Start terminal session (blocks)
- `Close() error` - Clean shutdown
- `GetScrollback() *ScrollbackBuffer` - Access scrollback buffer
- `GetSize() (width, height int)` - Get terminal dimensions
- `SetEnvironment(key, value string)` - Set environment variable
- `GetEnvironment() map[string]string` - Get all environment variables

#### `ScrollbackBuffer`
Thread-safe scrollback buffer for terminal history.

**Methods:**
- `NewScrollbackBuffer(maxLines int) *ScrollbackBuffer` - Create buffer
- `AddLine(line []byte)` - Add line to buffer
- `GetLines() [][]byte` - Get all lines
- `Clear()` - Clear buffer

#### `PTYTerminalOptions`
Configuration options for PTY terminal.

**Fields:**
- `Shell string` - Shell to launch (default: $SHELL or /bin/bash)
- `Environment map[string]string` - Environment variables
- `ScrollbackLines int` - Buffer size (default: 10000)
- `EnableMouse bool` - Enable mouse support (default: true on Unix)
- `Debug bool` - Enable debug logging (default: false)

### Functions

#### `GetTerminalSize() (width, height int, err error)`
Returns the current terminal dimensions.

## Testing

Run tests:
```bash
go test ./pkg/sshutil/...
```

Run specific tests:
```bash
go test ./pkg/sshutil/ -run TestScrollbackBuffer -v
```

## Platform Support

| Feature | Unix/Linux/macOS | Windows |
|---------|-----------------|---------|
| PTY Support | ✅ Full | ✅ Basic |
| Scrollback Buffer | ✅ | ✅ |
| Signal Handling | ✅ Full | ✅ Basic |
| Mouse Support | ✅ | ❌ |
| EOF Detection | ✅ Ctrl+D | ✅ Ctrl+D/Ctrl+Z |
| Window Resize | ✅ SIGWINCH | ⚠️ Limited |

## Security Considerations

- Terminal sessions handle sensitive data - ensure proper cleanup
- Debug mode may log sensitive information - use only in development
- Environment variables are inherited - sanitize before use
- EOF signals properly terminate sessions to prevent hanging connections

## Performance

- Scrollback buffer uses efficient ring buffer
- Thread-safe with minimal lock contention
- Configurable buffer size for memory control
- Goroutines for concurrent I/O streaming

## Contributing

When contributing to this package:

1. Maintain cross-platform compatibility
2. Add tests for new features
3. Update documentation
4. Follow existing code style
5. Consider security implications

## License

This package is part of the ssh-x-term project and is licensed under the MIT License.
