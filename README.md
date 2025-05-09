# SSH-X-Term

SSH-X-Term is a terminal-based SSH client with a text-based user interface (TUI) built using the Bubble Tea framework. It allows you to manage SSH connections, connect to remote servers, and provides a seamless terminal experience within your terminal.

## Features

- Manage SSH connections with a user-friendly interface
- Save and organize SSH connections
- Connect to SSH servers with password or key-based authentication
- Fullscreen terminal UI for SSH sessions

## SSH-X-Term Project Structure

```
ssh-x-term/
├── cmd/
│   └── ssh-x-term/
│       └── main.go            # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go          # Configuration handling
│   │   └── models.go          # Configuration data models
│   ├── ssh/
│   │   ├── client.go          # SSH client implementation
│   │   └── session.go         # SSH session management
│   └── ui/
│       ├── components/
│       │   ├── connection_list.go    # List of SSH connections
│       │   ├── form.go               # Form for adding/editing connections
│       │   └── terminal.go           # Terminal component for SSH sessions
│       ├── model.go                  # Main UI model
│       ├── update.go                 # Update logic for UI
│       └── view.go                   # View rendering logic
├── pkg/
│   └── sshutil/
│       ├── auth.go                   # Authentication utilities
│       └── terminal.go               # Terminal utilities
├── go.mod
├── go.sum
└── README.md
```

## Installation

### Prerequisites

- Go 1.18 or higher

### Building from source

1. Clone this repository:
   ```
   git clone https://github.com/eugeniofciuvasile/ssh-x-term.git
   cd ssh-x-term
   ```

2. Build the application:
   ```
   go build -o ssh-x-term ./cmd/ssh-x-term
   ```

3. Install the application (optional):
   ```
   go install ./cmd/ssh-x-term
   ```

## Usage

### Basic Usage

Simply run the application:

```
./ssh-x-term
```

Or if installed via `go install`:

```
ssh-x-term
```

### Managing Connections

- Press `a` to add a new SSH connection
- Press `e` to edit the selected connection
- Press `d` to delete the selected connection
- Press `o` to open next connections in a new window terminal
- Press `Enter` to connect to the selected connection
- Use arrow keys to navigate the connection list

### Connection Form

- Fill in the connection details
- Press `Tab` to navigate between fields
- Press `Ctrl+p` to toggle between password and key authentication
- Press `Enter` on the Submit button to save
- Press `Esc` to cancel

### SSH Terminal

- Press `Esc` to close the terminal and return to the connection list

## Configuration

SSH-X-Term stores its configuration in `~/.config/ssh-x-term/ssh-x-term.json`.

## License

[MIT](LICENSE)

## Credits

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Bubbles](https://github.com/charmbracelet/bubbles)
- [crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh)
