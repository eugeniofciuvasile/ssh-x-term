# SSH-X-Term

SSH-X-Term is a powerful terminal-based SSH client with a TUI (Text User Interface) built on [Bubble Tea](https://github.com/charmbracelet/bubbletea).  
It lets you manage SSH connections, securely store credentials using Bitwarden, and connect to remote servers with both password and key-based authentication.  
Cross-platform features include support for sshpass (Unix), plink.exe (Windows), and full tmux integration.

![Screenshot](https://github.com/user-attachments/assets/a545d09b-2101-4c6d-b5b9-377b2d554d57)

## Features

- Manage SSH connections in an interactive Bubble Tea TUI.
- Secure credential storage and retrieval via Bitwarden CLI.
- Password-based SSH login automation using sshpass (Unix) or plink.exe (Windows).
- Key-based SSH authentication.
- Open connections in new tmux windows or current terminal.
- Fullscreen and responsive TUI.

## Project Structure

```
ssh-x-term/
├── cmd/
│   └── sxt/
│       └── main.go                        # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go                      # Configuration handling
│   │   └── models.go                      # Configuration data models
│   ├── ssh/
│   │   ├── client.go                      # SSH client implementation
│   │   ├── session_unix.go                # SSH session management (Unix)
│   │   └── session_windows.go             # SSH session management (Windows)
│   └── ui/
│       ├── components/
│       │   ├── bitwarden_config.go        # Bitwarden CLI configuration form/component
│       │   ├── bitwarden_login_form.go    # Bitwarden login form component
│       │   ├── bitwarden_unlock_form.go   # Bitwarden unlock form component
│       │   ├── connection_list.go         # List of SSH connections
│       │   ├── form.go                    # Form for adding/editing connections
│       │   ├── storage_select.go          # Credential storage selection (Bitwarden/etc.)
│       │   └── terminal.go                # Terminal component for SSH sessions
│       ├── model.go                       # Main UI model
│       ├── update.go                      # Update logic for UI
│       └── view.go                        # View rendering logic
├── pkg/
│   └── sshutil/
│       ├── auth.go                        # Authentication utilities (sshpass/plink, etc.)
│       ├── terminal_unix.go               # Terminal utilities (Unix)
│       └── terminal_windows.go            # Terminal utilities (Windows)
├── go.mod
├── go.sum
└── README.md
```

**Note:**  
- Bitwarden integration is handled via several UI components:  
  - `bitwarden_config.go`, `bitwarden_login_form.go`, `bitwarden_unlock_form.go` for configuration, login, and unlock flows.  
  - `storage_select.go` lets users choose Bitwarden or other credential storage.
 
**Flow chart**
- [FLOW](https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/FLOW.md)

## Prerequisites

- **Go 1.24+**
- **Bitwarden CLI (`bw`)** — for credential management ([install guide](https://bitwarden.com/help/cli/))
- **sshpass** — for password authentication on Unix ([install with your package manager](https://linux.die.net/man/1/sshpass))
- **tmux** — recommended for multi-window SSH sessions ([install guide](https://github.com/tmux/tmux/wiki/Installing))
- **plink.exe** — for password authentication on Windows ([download from PuTTY](https://www.chiark.greenend.org.uk/~sgtatham/putty/latest.html))
- **(Optional) ssh client** — `ssh` should be available on your system

**Ensure all required binaries are available in your `$PATH`.**

## System dependencies

ssh-x-term requires the following system tools to be installed:

- `tmux`
- `sshpass`
- `bitwarden-cli` (npm package: `@bitwarden/cli`, install globally: `npm install -g @bitwarden/cli`)

### Linux (Debian/Ubuntu):

```sh
sudo apt update
sudo apt install -y tmux sshpass
npm install -g @bitwarden/cli
```

### macOS (with Homebrew):

```sh
brew install tmux
# See https://gist.github.com/arunoda/7790979 for sshpass on macOS
npm install -g @bitwarden/cli
```

### Windows:

- Install `tmux` and `sshpass` via WSL/Cygwin or use alternatives.
- Install Bitwarden CLI with: `npm install -g @bitwarden/cli`

## Installation

### Option 1: Install using npm (Recommended)

The easiest way to install SSH-X-Term is using npm:

```sh
# Install globally
npm install -g ssh-x-term

# Run the command
sxt
```

This will automatically download the appropriate binary for your platform and set up the command.

The npm installer also attempts to install required dependencies (`bw`, `sshpass`, `tmux`) if they are not already available in your system's `$PATH`.

---

### Option 2: Build from source

Ensure you have **Go 1.21+** installed. You can use either the Go from your package manager or [install manually](https://go.dev/dl/).  
If you manually install Go, add the following to your shell config (`~/.bashrc`, `~/.zshrc`, etc.):

```sh
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
```

Then:

```sh
# Clone and build the project
git clone https://github.com/eugeniofciuvasile/ssh-x-term.git
cd ssh-x-term
go build -o sxt ./cmd/sxt
```

Or install globally with Go:

```sh
go install github.com/eugeniofciuvasile/ssh-x-term/cmd/sxt@latest
```

Make sure `$GOPATH/bin` is in your `$PATH` to use `sxt` from anywhere.

---

### Option 3: Download pre-built binary

You can download the pre-built binary for your platform from the [Releases](https://github.com/eugeniofciuvasile/ssh-x-term/releases) page.

After downloading:

```sh
chmod +x sxt
mv sxt /usr/local/bin/   # or any location in your PATH
```

## Usage
1. Run the app:
```sh
./sxt
# or, if installed globally:
sxt
```
    
2. **Manage SSH connections:**
    - Press `a` to add, `e` to edit, `d` to delete a connection.
    - Press `o` to toggle opening connections in a new tmux window.
    - Press `Enter` to connect.
    - Use arrow keys to navigate.
    - All credentials are stored/retrieved using Bitwarden.

3. **Connection Form:**
    - Fill in fields as prompted.
    - `Tab` to navigate, `Ctrl+p` to toggle auth type, `Enter` to submit, `Esc` to cancel.

4. **SSH Session:**
    - `Esc` to disconnect.
    - Passwords are supplied securely via sshpass or plink.exe (never echoed or stored in plaintext).

## Configuration

Config is stored at: `~/.config/ssh-x-term/ssh-x-term.json`  
Connection secrets are stored in your Bitwarden vault.

## Security Notes

- **Passwords are only handled via secure subprocesses (`sshpass`, `plink.exe`) and Bitwarden.**
- **No plaintext passwords are ever written to disk or logs.**

## License

[MIT](LICENSE)

## Disclaimer

SSH-X-Term is an independent open-source project released under the MIT License.  
It is **not affiliated with, endorsed by, or supported by** any of the credited projects, including Bubble Tea, Bitwarden, sshpass, PuTTY/plink, or any other third-party software listed above.

**Security Notice:**  
SSH-X-Term integrates with external tools for SSH and credential management.  
The safe handling, storage, and security of your credentials (including passwords and keys) is ultimately your responsibility.  
By using this software, you agree that the author and contributors bear **no liability** for any potential loss, compromise, or misuse of credentials or data.

For details, see the [MIT License](LICENSE).

## Credits

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Bitwarden CLI](https://bitwarden.com/help/cli/)
- [sshpass](https://linux.die.net/man/1/sshpass)
- [PuTTY/plink.exe](https://www.chiark.greenend.org.uk/~sgtatham/putty/latest.html)
