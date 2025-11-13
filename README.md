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

- Manage SSH connections in an interactive Bubble Tea TUI.
- **Dual credential storage modes:**
  - **Local storage** with go-keyring (system keyring integration)
  - **Bitwarden vault** storage via Bitwarden CLI
- Secure credential storage: passwords never stored in plaintext.
- Password-based SSH login automation using passh (Unix) or plink.exe (Windows).
- Key-based SSH authentication.
- Open connections in new tmux windows or current terminal.
- Fullscreen and responsive TUI.

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
    - `Esc` to disconnect.
    - Passwords are supplied securely via passh or plink.exe (never echoed or stored in plaintext).

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
