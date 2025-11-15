# SSH-X-Term

<p>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./logo.svg" width="240">
    <source media="(prefers-color-scheme: light)" srcset="./logo.svg" width="240">
    <img alt="SSH-X-Term Logo" src="./logo.svg" width="240">
  </picture>
  <br>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/releases"><img src="https://img.shields.io/github/v/release/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="Latest Release"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/actions"><img src="https://github.com/eugeniofciuvasile/ssh-x-term/actions/workflows/go.yml/badge.svg" alt="Build Status"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/stargazers"><img src="https://img.shields.io/github/stars/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="GitHub Stars"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/LICENSE"><img src="https://img.shields.io/github/license/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="License"></a>
</p>

SSH-X-Term is a powerful terminal-based SSH client with a TUI (Text User Interface) built on [Bubble Tea](https://github.com/charmbracelet/bubbletea).  
It lets you manage SSH connections and securely store credentials using either **local system keyring** (via [go-keyring](https://github.com/zalando/go-keyring)) or **Bitwarden vault**, and connect to remote servers with both password and key-based authentication.  
Cross-platform features include support for passh (Unix), plink.exe (Windows), and full tmux integration.

![Screenshot](https://github.com/user-attachments/assets/a545d09b-2101-4c6d-b5b9-377b2d554d57)

## Features

- **Integrated SSH Terminal**: Fully functional terminal emulator within Bubble Tea
  - VT100/ANSI escape sequence support for proper terminal rendering
  - Scrollback buffer (10,000 lines) with keyboard and mouse scrolling
  - Text selection with mouse (click and drag)
  - Copy to clipboard support (Ctrl+C or automatic on selection)
  - Full keyboard support (arrow keys, home, end, function keys, etc.)
  - Window resize handling
  - Works entirely within the TUI (no external terminal takeover)
- Manage SSH connections in an interactive Bubble Tea TUI
- **Dual credential storage modes:**
  - **Local storage** with go-keyring (system keyring integration)
  - **Bitwarden vault** storage via Bitwarden CLI
- Secure credential storage: passwords never stored in plaintext
- Password-based SSH login automation using passh (Unix) or plink.exe (Windows)
- Key-based SSH authentication
- Open connections in new tmux windows or integrated terminal
- Fullscreen and responsive TUI

## Project Structure

```
ssh-x-term/
├── cmd/
│   └── sxt/
│       └── main.go                                # Application entry point
├── internal/
│   ├── config/
│   │   ├── bitwarden.go                          # Bitwarden vault integration and management
│   │   ├── config.go                             # Local configuration handling with go-keyring
│   │   ├── models.go                             # Configuration data models
│   │   ├── pathutil.go                           # Path utilities (home directory expansion)
│   │   └── storage.go                            # Storage interface definition
│   ├── ssh/
│   │   ├── client.go                             # SSH client implementation
│   │   ├── session_unix.go                       # SSH session management (Unix)
│   │   └── session_windows.go                    # SSH session management (Windows)
│   └── ui/
│       ├── components/
│       │   ├── bitwarden_collection_list.go      # Bitwarden collection selector
│       │   ├── bitwarden_config.go               # Bitwarden CLI configuration form
│       │   ├── bitwarden_login_form.go           # Bitwarden login form component
│       │   ├── bitwarden_organization_list.go    # Bitwarden organization selector
│       │   ├── bitwarden_unlock_form.go          # Bitwarden unlock form component
│       │   ├── connection_list.go                # List of SSH connections
│       │   ├── form.go                           # Form for adding/editing connections
│       │   ├── storage_select.go                 # Credential storage selection (Local/Bitwarden)
│       │   └── terminal.go                       # Terminal component for SSH sessions
│       ├── connection_handler.go                 # Connection lifecycle management
│       ├── model.go                              # Main UI model and state
│       ├── update.go                             # Update logic for UI events
│       └── view.go                               # View rendering logic
├── pkg/
│   └── sshutil/
│       ├── auth.go                               # Authentication utilities (passh/plink, etc.)
│       ├── terminal_unix.go                      # Terminal utilities (Unix)
│       └── terminal_windows.go                   # Terminal utilities (Windows)
├── go.mod                                        # Go module dependencies
├── go.sum                                        # Go module checksums
├── package.json                                  # npm package configuration
├── index.js                                      # npm entry point
├── install.js                                    # npm post-install script
├── LICENSE                                       # MIT License
├── CONTRIBUTING.md                               # Contribution guidelines
├── FLOW.md                                       # Application flow documentation
└── README.md                                     # This file
```

**Note:**  
- **Local Storage Mode**: Uses `go-keyring` to securely store passwords in the system keyring (see `config.go`)
- **Bitwarden Integration**: Managed through several components:
  - `bitwarden.go` handles vault operations via Bitwarden CLI
  - `bitwarden_config.go`, `bitwarden_login_form.go`, `bitwarden_unlock_form.go` for authentication flows
  - `bitwarden_organization_list.go`, `bitwarden_collection_list.go` for organizational vault management
  - `storage_select.go` lets users choose between Local (with go-keyring) or Bitwarden storage
 
**Flow chart**
- [FLOW](https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/FLOW.md)

## Prerequisites

- **Go 1.24+**
- **System keyring support** — for secure local password storage via go-keyring:
  - **macOS**: Keychain (built-in)
  - **Linux**: Secret Service API (`gnome-keyring`, `kwallet`, or compatible)
  - **Windows**: Credential Manager (built-in)
- **Bitwarden CLI (`bw`)** — optional, for Bitwarden vault credential management ([install guide](https://bitwarden.com/help/cli/))
- **passh** — for password authentication on Unix ([compile it from here](https://github.com/clarkwang/passh))
- **tmux** — recommended for multi-window SSH sessions ([install guide](https://github.com/tmux/tmux/wiki/Installing))
- **plink.exe** — for password authentication on Windows ([download from PuTTY](https://www.chiark.greenend.org.uk/~sgtatham/putty/latest.html))
- **(Optional) ssh client** — `ssh` should be available on your system

**Ensure all required binaries are available in your `$PATH`.**

## System dependencies

ssh-x-term requires the following system tools to be installed:

- `tmux`
- `passh`
- System keyring support (for secure local password storage)
- `bitwarden-cli` (optional, npm package: `@bitwarden/cli`, install globally: `npm install -g @bitwarden/cli`)

### Linux (Debian/Ubuntu):

```sh
sudo apt update
sudo apt install -y tmux gnome-keyring
npm install -g @bitwarden/cli
# follow github repo https://github.com/clarkwang/passh to compile passh
```

**Note**: For Linux, ensure you have a keyring daemon running (e.g., `gnome-keyring-daemon` or `kwallet`) for go-keyring to work.

### macOS (with Homebrew):

```sh
brew install tmux
npm install -g @bitwarden/cli
# follow github repo https://github.com/clarkwang/passh to compile passh
```

**Note**: macOS uses Keychain by default, which is already available.

### Windows:

- Install `tmux` and `passh` via WSL/Cygwin or use alternatives.
- Install Bitwarden CLI with: `npm install -g @bitwarden/cli`
- Windows Credential Manager is used by go-keyring and is built-in.

## Installation

### Option 1: Install using npm (Recommended)

The easiest way to install SSH-X-Term is using npm [npm package](https://www.npmjs.com/package/ssh-x-term):

```sh
# Install globally
npm install -g ssh-x-term

# Run the command
sxt
```

This will automatically download the appropriate binary for your platform and set up the command.

The npm installer also attempts to install required dependencies (`bw`, `passh`, `tmux`) if they are not already available in your system's `$PATH`.

---

### Option 2: Build from source

Ensure you have **Go 1.24+** installed. You can use either the Go from your package manager or [install manually](https://go.dev/dl/).  
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
    
2. **First run:** Choose your credential storage mode:
    - **Local Storage**: Uses system keyring (Keychain/Secret Service/Credential Manager)
    - **Bitwarden**: Uses Bitwarden vault (requires `bw` CLI and authentication)

3. **Manage SSH connections:**
    - Press `a` to add, `e` to edit, `d` to delete a connection.
    - Press `o` to toggle opening connections in a new tmux window.
    - Press `Enter` to connect.
    - Use arrow keys to navigate.
    - Credentials are stored securely based on your chosen storage mode.

4. **Connection Form:**
    - Fill in fields as prompted.
    - `Tab` to navigate, `Ctrl+p` to toggle auth type, `Enter` to submit, `Esc` to cancel.

5. **SSH Session:**
    - Fully integrated terminal within Bubble Tea UI
    - **Navigation:**
      - `Esc` to disconnect and return to connection list
      - `Ctrl+D` to send EOF (End of File) signal
    - **Scrolling:**
      - `PgUp` / `PgDn` to scroll up/down by 10 lines
      - `Shift+Up` / `Shift+Down` for scrolling
      - `Ctrl+Home` to scroll to top
      - `Ctrl+End` to scroll to bottom
      - Mouse wheel for scrolling
    - **Text Selection & Copy:**
      - Click and drag with mouse to select text
      - `Ctrl+C` to copy selected text (or send interrupt if no selection)
      - `Ctrl+Shift+C` to force copy selection
      - Selected text is automatically copied to clipboard on mouse release
    - **Terminal Features:**
      - VT100/ANSI escape sequence support
      - 10,000 line scrollback buffer
      - Window resize support
      - Full keyboard support (arrow keys, home, end, etc.)
    - Passwords are supplied securely (never echoed or stored in plaintext).

## Configuration

SSH-X-Term supports two credential storage modes:

### Local Storage (Default)
- Config is stored at: `~/.config/ssh-x-term/ssh-x-term.json`  
- **Passwords are stored securely in your system keyring** via `go-keyring`:
  - **macOS**: Stored in Keychain
  - **Linux**: Stored via Secret Service API (gnome-keyring/kwallet)
  - **Windows**: Stored in Credential Manager
- Connection metadata (host, port, username, etc.) is saved in the JSON file
- Passwords are **never** stored in plaintext in the JSON file

### Bitwarden Vault Storage
- Connection secrets are stored in your Bitwarden vault
- Requires Bitwarden CLI (`bw`) to be installed and configured
- Supports both personal vaults and organization collections

## Common Workflows

### Quick Start: First Connection

1. **Launch SSH-X-Term**:
   ```bash
   sxt
   ```

2. **Choose Storage Backend**:
   - Select "Local Storage" for getting started quickly
   - Or "Bitwarden" if you want vault-based credential management

3. **Add Your First Connection**:
   - Press `a` to add a new connection
   - Fill in the details:
     - **Name**: My Server (friendly name)
     - **Host**: example.com
     - **Port**: 22
     - **Username**: your-username
     - **Auth Type**: Press `Ctrl+p` to toggle between Password/Key
     - **Password** (if using password auth): your-password
     - **Key File** (if using key auth): ~/.ssh/id_rsa
   - Press `Enter` to save

4. **Connect**:
   - Highlight your connection with arrow keys
   - Press `Enter` to connect
   - You're now in an interactive SSH terminal!

5. **Work in the Terminal**:
   - Type commands as you would in any terminal
   - Scroll through history with `PgUp`/`PgDn`
   - Select and copy text by clicking and dragging
   - Press `Esc` when done to return to the connection list

### Using Bitwarden Storage

1. **Initial Setup**:
   - Launch `sxt` and select "Bitwarden" as storage backend
   - Enter your Bitwarden server URL (or leave default for bitwarden.com)
   - Enter your email address

2. **Login and Unlock**:
   - Enter your master password when prompted
   - If 2FA is enabled, enter your 2FA code
   - Unlock your vault when prompted

3. **Organization Setup** (Optional):
   - If you use organizations, select your organization
   - Choose a collection to store connections
   - Or select "Personal Vault" for individual use

4. **Manage Connections**:
   - Connections are now stored in your Bitwarden vault
   - They sync across all your devices
   - Team members with access to the collection can see shared connections

### Working with Multiple Connections

1. **Open in tmux Windows**:
   - Press `o` to toggle "Open in new terminal" mode
   - When enabled, connections open in new tmux windows
   - Switch between windows with `Ctrl+b` then `w` (tmux commands)
   - Each connection runs in its own window

2. **Integrated Terminal Mode**:
   - When "Open in new terminal" is disabled
   - Connections open in the integrated Bubble Tea terminal
   - Only one connection active at a time
   - Press `Esc` to return to connection list and switch

### Editing and Managing Connections

- **Edit**: Press `e` on a highlighted connection
- **Delete**: Press `d` on a highlighted connection (confirms before deleting)
- **Switch Storage**: Press `s` to change storage backend (Local ↔ Bitwarden)
- **Quit**: Press `Ctrl+c` from the connection list

## Troubleshooting

### Keyring Issues (Linux)

**Problem**: "Failed to retrieve password from keyring" or "keyring not available"

**Solution**:
1. Ensure you have a keyring daemon installed:
   ```bash
   sudo apt install gnome-keyring
   ```

2. Check if the keyring daemon is running:
   ```bash
   ps aux | grep gnome-keyring-daemon
   ```

3. Start the keyring daemon if not running:
   ```bash
   gnome-keyring-daemon --start --components=secrets
   ```

4. For headless systems or servers, consider using Bitwarden storage instead:
   - Bitwarden doesn't require a GUI keyring
   - Better for remote/server environments

**Alternative**: Set up gnome-keyring to start automatically:
```bash
# Add to ~/.bashrc or ~/.profile
if [ -z "$DBUS_SESSION_BUS_ADDRESS" ]; then
    eval $(dbus-launch --sh-syntax)
fi
eval $(gnome-keyring-daemon --start --components=secrets)
export SSH_AUTH_SOCK
```

### Bitwarden CLI Issues

**Problem**: "Bitwarden CLI (bw) is not installed or not in your PATH"

**Solution**:
1. Install Bitwarden CLI globally with npm:
   ```bash
   npm install -g @bitwarden/cli
   ```

2. Verify installation:
   ```bash
   bw --version
   ```

3. If `bw` is still not found, add npm global bin to PATH:
   ```bash
   export PATH="$PATH:$(npm config get prefix)/bin"
   ```

**Problem**: "You are not logged in" or "Vault is locked"

**Solution**:
1. SSH-X-Term will automatically prompt you to log in
2. If issues persist, try logging in manually:
   ```bash
   bw login
   bw unlock
   ```

3. Check Bitwarden status:
   ```bash
   bw status
   ```

### SSH Connection Issues

**Problem**: "Failed to connect to SSH server" or "Authentication failed"

**Solution**:
1. Verify host and port are correct
2. Test connection manually:
   ```bash
   ssh -p 22 username@hostname
   ```

3. For key-based auth, ensure:
   - Key file path is absolute (e.g., `/home/user/.ssh/id_rsa`)
   - Key file has correct permissions: `chmod 600 ~/.ssh/id_rsa`
   - Key is not password-protected (or use ssh-agent)

4. For password auth, ensure:
   - `passh` (Unix) or `plink.exe` (Windows) is installed
   - Password is correct in keyring/vault

**Problem**: "passh: not found" (Linux/macOS)

**Solution**:
1. Install passh from source:
   ```bash
   git clone https://github.com/clarkwang/passh.git
   cd passh
   cc -o passh passh.c
   sudo cp passh /usr/local/bin/
   ```

2. Verify installation:
   ```bash
   passh -V
   ```

### Terminal Display Issues

**Problem**: Terminal output looks garbled or colors are wrong

**Solution**:
1. Check your terminal emulator supports 256 colors:
   ```bash
   echo $TERM
   # Should show xterm-256color or similar
   ```

2. Set TERM if needed:
   ```bash
   export TERM=xterm-256color
   ```

3. Try resizing the terminal window (forces re-render)

**Problem**: Terminal is too small or content is cut off

**Solution**:
1. Resize your terminal window (SSH-X-Term adapts automatically)
2. Minimum recommended size: 80x24
3. For best experience: 120x40 or larger

### tmux Integration Issues

**Problem**: "Failed to start tmux session"

**Solution**:
1. Ensure tmux is installed:
   ```bash
   sudo apt install tmux     # Debian/Ubuntu
   brew install tmux         # macOS
   ```

2. SSH-X-Term will fall back to integrated terminal mode if tmux is unavailable
3. Check `~/.config/ssh-x-term/sxt.log` for details

**Problem**: Already inside tmux when launching

**Solution**:
- SSH-X-Term detects if already in tmux and skips auto-launch
- This is normal behavior when running inside an existing tmux session

### Performance Issues

**Problem**: Slow or laggy terminal output

**Solution**:
1. Large outputs (e.g., `cat` large files) may be slow
2. Use paging commands: `less`, `more`
3. Reduce scrollback buffer (requires code modification)
4. Consider using tmux mode for better performance with multiple connections

**Problem**: High memory usage

**Solution**:
1. Each connection uses ~6-7 MB for scrollback buffer
2. Close unused connections (press `Esc` in terminal)
3. This is normal for terminal emulators with large scrollback

### File Paths and Permissions

**Problem**: "Failed to create config directory"

**Solution**:
1. Ensure you have write permissions to `~/.config/`:
   ```bash
   ls -la ~/.config/
   mkdir -p ~/.config/ssh-x-term
   ```

2. Check file permissions:
   ```bash
   chmod 755 ~/.config/ssh-x-term
   ```

**Problem**: "Failed to read key file"

**Solution**:
1. Use absolute paths for key files: `/home/user/.ssh/id_rsa`
2. Not relative paths: `~/.ssh/id_rsa` (use full path)
3. Ensure key file exists and is readable:
   ```bash
   ls -l /home/user/.ssh/id_rsa
   chmod 600 /home/user/.ssh/id_rsa
   ```

### Getting Help

If you encounter issues not covered here:

1. **Check logs**: `~/.config/ssh-x-term/sxt.log`
2. **Enable debug logging**:
   ```bash
   SSH_X_TERM_LOG=/tmp/sxt-debug.log sxt
   ```

3. **Report issues**: [GitHub Issues](https://github.com/eugeniofciuvasile/ssh-x-term/issues)
   - Include OS and version
   - Include relevant log excerpts
   - Describe steps to reproduce

4. **Check documentation**:
   - [FLOW.md](FLOW.md) - Application architecture and state flows
   - [IMPLEMENTATION.md](IMPLEMENTATION.md) - Technical implementation details
   - [CONTRIBUTING.md](CONTRIBUTING.md) - Development guidelines

## Security Notes

- **Local Storage Mode**: Passwords are stored securely using **go-keyring**, which integrates with your system's native credential storage:
  - macOS: Keychain
  - Linux: Secret Service API (gnome-keyring, kwallet, etc.)
  - Windows: Credential Manager
- **Bitwarden Mode**: Credentials are managed via Bitwarden CLI and stored in your encrypted vault
- **SSH Authentication**: Passwords are supplied securely via subprocesses (`passh`, `plink.exe`) and never echoed or logged
- **No plaintext passwords**: Passwords are **never** written to disk in plaintext (not in config files or logs)

## License

[MIT](LICENSE)

## Disclaimer

SSH-X-Term is an independent open-source project released under the MIT License.  
It is **not affiliated with, endorsed by, or supported by** any of the credited projects, including Bubble Tea, Bitwarden, passh, PuTTY/plink, or any other third-party software listed above.

**Security Notice:**  
SSH-X-Term integrates with external tools for SSH and credential management.  
The safe handling, storage, and security of your credentials (including passwords and keys) is ultimately your responsibility.  
By using this software, you agree that the author and contributors bear **no liability** for any potential loss, compromise, or misuse of credentials or data.

For details, see the [MIT License](LICENSE).

## Credits

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework
- [go-keyring](https://github.com/zalando/go-keyring) — Secure system keyring integration
- [Bitwarden CLI](https://bitwarden.com/help/cli/) — Bitwarden vault management
- [passh](https://github.com/clarkwang/passh) — Password-based SSH automation (Unix)
- [PuTTY/plink.exe](https://www.chiark.greenend.org.uk/~sgtatham/putty/latest.html) — Password-based SSH automation (Windows)
