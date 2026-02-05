# SSH-X-Term

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./logo.svg" width="240">
    <source media="(prefers-color-scheme: light)" srcset="./logo.svg" width="240">
    <img alt="SSH-X-Term Logo" src="./logo.svg" width="240">
  </picture>
  <br>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/releases"><img src="https://img.shields.io/github/v/release/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="Latest Release"></a>
  <a href="https://www.npmjs.com/package/ssh-x-term"><img src="https://img.shields.io/npm/v/ssh-x-term?style=flat-square&logo=npm" alt="NPM Version"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/releases"><img src="https://img.shields.io/github/downloads/eugeniofciuvasile/ssh-x-term/total?style=flat-square&color=blue" alt="GitHub Downloads"></a>
  <a href="https://www.npmjs.com/package/ssh-x-term"><img src="https://img.shields.io/npm/dt/ssh-x-term?style=flat-square&logo=npm" alt="NPM Downloads"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/actions"><img src="https://github.com/eugeniofciuvasile/ssh-x-term/actions/workflows/go.yml/badge.svg" alt="Build Status"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/stargazers"><img src="https://img.shields.io/github/stars/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="GitHub Stars"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/LICENSE"><img src="https://img.shields.io/github/license/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="License"></a>
</p>

**SSH-X-Term** is a powerful terminal-based SSH client with a TUI (Text User Interface) built on [Bubble Tea](https://github.com/charmbracelet/bubbletea).  
It seamlessly integrates **SSH connection management**, **SCP/SFTP file transfers**, and **secure credential storage** into a single, responsive interface.

Credentials can be stored securely using your **local system keyring** (via [go-keyring](https://github.com/zalando/go-keyring)) or directly in your **Bitwarden vault**.

---

> **⚠️ BREAKING CHANGE - Version 2.0+**  
> Starting from version 2.0, SSH-X-Term uses **pure Go SSH client** instead of external tools (passh, plink, ssh command).  
> **Old versions (< 2.0)** used external SSH clients and required `passh`/`plink.exe` installation.  
> **New versions (>= 2.0)** have **no external dependencies** - everything is handled by the built-in Go SSH client.
>
> ### What Changed:
> - ✅ **No more passh/plink** - Pure Go SSH implementation
> - ✅ **SSH Agent support** - Automatically uses ssh-agent for key authentication
> - ✅ **Encrypted keys** - Works with encrypted SSH keys via ssh-agent
> - ✅ **Better compatibility** - Works on all platforms without external tools
> - ✅ **Direct connection** - New `-c <connection-id>` flag for instant connections
> - ✅ **xterm-256color** - Full terminal support built-in
>
> If you're upgrading from version < 2.0, your existing configurations will continue to work, but you can now uninstall `passh` and `plink.exe` if you wish.

---

### 📺 Demo & Walkthrough

<div align="center">
  
  [![Watch on YouTube](https://img.shields.io/badge/Watch_on_YouTube-FF0000?style=for-the-badge&logo=youtube&logoColor=white)](https://www.youtube.com/watch?v=C-s-Lh_VdpQ)
  
  ![Demo](media/demo.gif)

</div>

---

## 🚀 Features

### ⚡ Quick Connect Mode (NEW in v2.0)
Lightning-fast connection selection via CLI.
- **Instant Access**: `sxt -l` for minimal UI connection picker
- **Pure Go SSH**: Uses built-in Go SSH client (no external dependencies)
- **Direct Connect**: `sxt -c <connection-id>` connects immediately by ID
- **Auto-Filter**: Start typing immediately - filter activates on first keypress
- **Smart Navigation**: Arrow keys exit filter and navigate list
- **Compact Display**: 10 connections per page
- **No External Tools**: No passh, plink, or ssh command needed
- **SSH Agent Support**: Automatically uses ssh-agent for encrypted keys

### 🖥️ Integrated SSH Terminal
Fully functional terminal emulator built entirely within the TUI.
- **Standards Compliant**: VT100/ANSI escape sequence support for proper rendering.
- **Power User Friendly**: 10,000 line scrollback buffer, mouse & keyboard scrolling.
- **Clipboard**: Text selection with mouse (click & drag), automatic copy, or `Ctrl+C`.
- **Responsive**: Full keyboard support and window resize handling.

### 📂 SCP/SFTP File Manager
Seamlessly transfer files without leaving the app.
- **Dual-pane Interface**: Intuitive Local vs. Remote panel navigation.
- **Full Control**: Upload, Download, Rename, Delete, and Create files/directories.
- **Search**: Recursive file search (`/` key) to find deep files instantly.
- **Secure**: Piggybacks on your existing authenticated SSH session.

### 🔐 Secure Credential Management
- **Local Storage**: Encrypted via system keyring (Keychain, Gnome Keyring, Credential Manager).
- **Bitwarden Integration**: Direct access to your vault via Bitwarden CLI.
- **Zero Plaintext**: Passwords are **never** stored in plaintext on disk.

### ⚡ Automation & Compatibility (v2.0+)
- **Pure Go SSH**: All connections use built-in Go SSH client (no external tools needed)
- **SSH Agent**: Automatic integration with ssh-agent for key authentication
- **Encrypted Keys**: Full support for encrypted SSH keys via ssh-agent
- **Password Auth**: Secure password authentication via system keyring
- **TMUX**: Open connections in new tmux windows automatically
- **xterm-256color**: Full terminal compatibility built-in

---

## 📦 Project Structure

```
ssh-x-term/
├── .github/
│   ├── ISSUE_TEMPLATE/           # GitHub issue templates (bug, feature, etc.)
│   ├── workflows/                # CI/CD definitions (Go build workflow)
│   └── dependabot.yml            # Dependency update automation
│
├── cmd/
│   └── sxt/
│       └── main.go               # Application entry point. Parses CLI flags and launches TUI or Quick Connect.
│
├── internal/
│   ├── cli/                                        # Quick-connect CLI features.
│   │   ├── connector.go                            # Pure Go SSH client connection (v2.0+)
│   │   └── selector.go                             # Connection selection in CLI mode
│   ├── config/                                     # Configuration and credential management
│   │   ├── bitwarden.go                            # Bitwarden vault integration via CLI
│   │   ├── config.go                               # Local config and secure keyring storage
│   │   ├── migrate.go                              # Config format migration between versions
│   │   ├── models.go                               # Configuration data models
│   │   ├── pathutil.go                             # Path resolution helpers
│   │   ├── sshconfig.go                            # SSH config file parsing and generation
│   │   ├── sshconfig_test.go                       # Unit tests for SSH config
│   │   └── storage.go                              # Storage provider interfaces
│   ├── ssh/                                        # Pure Go SSH client implementation (v2.0+)
│   │   ├── client.go                               # Core SSH client with auth and keyring
│   │   ├── interactive.go                          # Interactive terminal session (v2.0+)
│   │   ├── session_bubbletea_unix.go               # Bubble Tea SSH session (Unix)
│   │   ├── session_bubbletea_windows.go            # Bubble Tea SSH session (Windows)
│   │   ├── sftp.go                                 # SFTP file transfer logic
│   │   ├── sftp_unix.go                            # Unix-specific SFTP features
│   │   └── sftp_windows.go                         # Windows-specific SFTP features
│   └── ui/                                         # Bubble Tea TUI (Text UI): models, logic, and components.
│       ├── components/
│       │   ├── bitwarden_collection_list.go        # Bitwarden collection picker.
│       │   ├── bitwarden_config.go                 # Bitwarden CLI configuration form.
│       │   ├── bitwarden_login_form.go             # Bitwarden login UI.
│       │   ├── bitwarden_organization_list.go      # Organization select for vault.
│       │   ├── bitwarden_unlock_form.go            # Bitwarden vault unlock UI.
│       │   ├── connection_list.go                  # List and picker of SSH connections.
│       │   ├── delete_confirmation.go              # Confirm deletion UI dialog.
│       │   ├── form.go                             # Common connection add/edit form.
│       │   ├── scp_manager.go                      # SCP/SFTP dual pane file manager.
│       │   ├── ssh_passphrase_form.go              # UI for passphrase input.
│       │   ├── storage_select.go                   # Local vs Bitwarden credential storage UI.
│       │   ├── styles.go                           # Common style definitions.
│       │   ├── terminal.go                         # Terminal emulation inside TUI.
│       │   ├── terminal_test.go                    # Terminal tests.
│       │   ├── vterm.go                            # Bubble Tea virtual terminal emulator.
│       │   ├── vterm_color_test.go                 # Color support tests for terminal.
│       │   └── vterm_test.go                       # Virtual terminal tests.
│       ├── connection_handler.go                   # Connection lifecycle management logic.
│       ├── model.go                                # Main UI state model: app state, active component, etc.
│       ├── update.go                               # Update logic for events in the UI.
│       └── view.go                                 # UI rendering logic.
│
├── pkg/
│   └── sshutil/
│       ├── auth.go                   # Authentication helper/utilities (e.g. key parsing).
│       └── list_ssh_keys.go          # Helpers to find and list private SSH keys.
│
├── demo.sh                    # Shell script to demo app or CLI usage.
├── go.mod                     # Go module dependencies.
├── go.sum                     # Go module package checksums.
├── index.js                   # Node.js entry point for npm package: downloads binary, runs app.
├── install.js                 # Node.js install script auto-downloads proper binary for platform/arch.
├── LICENSE                    # MIT License.
├── package.json               # npm package metadata.
├── FLOW.md                    # Detailed flow/features/architecture of application.
├── CONTRIBUTING.md            # Contribution guidelines.
├── IMPLEMENTATION.md          # Technical implementation details.
├── MIGRATION.md               # Migration instructions for config/data upgrade between versions.
├── COLOR_SUPPORT.md           # Documentation for color/terminal support.
├── logo.svg                   # Project logo.
├── media/
│   ├── demo.gif               # Demo GIF for UI.
│   └── demo.mp4               # Demo video for UI.
└── README.md                  # Main documentation.

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

---

## 🛠️ Prerequisites

- **Go 1.24+** (for building from source)
- **System Keyring** (for local password storage):
  - 🍎 **macOS**: Keychain (built-in)
  - 🐧 **Linux**: Secret Service API (`gnome-keyring`, `kwallet`, etc.)
  - 🪟 **Windows**: Credential Manager (built-in)
- **SSH Agent** (optional, for encrypted key support):
  - Run `ssh-agent` and add keys with `ssh-add` to avoid passphrase prompts
- **External Tools**:
  - **Bitwarden CLI (`bw`)** — optional, for Bitwarden vault credential management ([install guide](https://bitwarden.com/help/cli/))
  - **tmux** — optional, for multi-window SSH sessions ([install guide](https://github.com/tmux/tmux/wiki/Installing))

**⚠️ Note for versions < 2.0:**
- Old versions required `passh` (Unix) and `plink.exe` (Windows) - these are **NO LONGER NEEDED** in v2.0+
- If upgrading from v1.x, you can safely remove these tools

## 📚 System dependencies

**Version 2.0+** requires minimal system dependencies:

- `tmux` (optional, for multi-window sessions)
- System keyring support (for secure local password storage)
- `bitwarden-cli` (optional, npm package: `@bitwarden/cli`, install globally: `npm install -g @bitwarden/cli`)

**⚠️ NO LONGER NEEDED in v2.0+:**
- ~~`passh`~~ - Replaced by built-in Go SSH client
- ~~`plink.exe`~~ - Replaced by built-in Go SSH client  
- ~~`ssh` command~~ - Replaced by built-in Go SSH client

### Linux (Debian/Ubuntu):

```sh
sudo apt update
sudo apt install -y tmux gnome-keyring
npm install -g @bitwarden/cli  # optional
```

### macOS (with Homebrew):

```sh
brew install tmux  # optional
npm install -g @bitwarden/cli  # optional
```

### Windows:

- Install `tmux` via WSL if needed (optional)
- Install Bitwarden CLI with: `npm install -g @bitwarden/cli` (optional)
- Windows Credential Manager is used by go-keyring and is built-in

### SSH Agent Setup (for encrypted keys):

```sh
# Start ssh-agent
eval $(ssh-agent)

# Add your encrypted keys
ssh-add ~/.ssh/id_rsa
ssh-add ~/.ssh/id_ed25519

# Now sxt will use these keys without asking for passphrases!
```

---

## 📥 Installation

### Option 1: Install using npm (Recommended)

The easiest way to install is via the [npm package](https://www.npmjs.com/package/ssh-x-term):

```sh
# Install globally
npm install -g ssh-x-term

# Run
sxt
```

> This automatically attempts to install required dependencies (`bw`, `passh`, `tmux`) if missing.

### Option 2: Build from source

1. **Clone & Build**:
   ```sh
   git clone https://github.com/eugeniofciuvasile/ssh-x-term.git
   cd ssh-x-term
   go build -o sxt ./cmd/sxt
   ```

2. **Install with Go**:
   ```sh
   go install github.com/eugeniofciuvasile/ssh-x-term/cmd/sxt@latest
   ```

### Option 3: Pre-built Binary

Download the latest binary from the [Releases Page](https://github.com/eugeniofciuvasile/ssh-x-term/releases).

---

## 🎮 Usage

### First Time Setup

Before using SSH-X-Term, initialize it once:

```sh
sxt -i
```

This will:
- Migrate any old JSON configuration (if exists)
- Create backup of existing SSH config
- Set up SSH-X-Term for use

### Standard Mode (Full TUI)

1. **Start the App**:
   ```sh
   sxt
   ```

2. **First Run Setup**:
   - Choose **Local Storage** (System Keyring) or **Bitwarden** (Vault).

3. **Manage Connections**:
   - `a` : **Add** a new connection.
   - `e` : **Edit** selected connection.
   - `d` : **Delete** connection.
   - `s` : Open **SCP/SFTP File Manager**.
   - `o` : Toggle **TMUX** mode (open in new window).
   - `Enter` : **Connect** (start SSH session).

4. **📂 Inside SCP Manager**:
   - `Tab` : Switch between **Local** ↔️ **Remote** panels.
   - `Enter` : **Enter** folder.
   - `Backspace` : **Exit** folder.
   - `c` : **Change** folder.
   - `g` : **Get**/**Download** file/folder.
   - `u` : **Upload** file/folder.
   - `n` : Create **New** file/folder.
   - `r` : **Rename** file.
   - `d` : **Delete** file.
   - `/` : **Search** recursively.

5. **🖥️ Inside SSH Session**:
   - `PgUp` / `PgDn` : Scroll history.
   - `Ctrl+D` : Send EOF.
   - `Esc` `Esc` (Double Press) : Disconnect and return to menu.

### Quick Connect Mode (CLI)

**Fast connection selection without the full TUI** (v2.0+):

```sh
# First time: Initialize SSH-X-Term
sxt -i

# Quick connect with selection menu
sxt -l

# Direct connect by connection ID (instant, no menu)
sxt -c <connection-id>
```

**Note**: You must run `sxt -i` once before using `sxt -l` or `sxt -c` to initialize and migrate your configuration.

#### Quick Connect (`sxt -l`)
This displays a minimal connection list where you can:
- **Start typing immediately** to filter connections by name
- Use **arrow keys** to navigate (exits filter mode and navigates)
- Press **Enter** while filtering to apply filter, or to connect when not filtering
- Press **Esc** to clear filter (if filtering) or quit
- Press **Ctrl+C** to quit immediately
- **10 connections per page** with pagination

#### Direct Connect (`sxt -c <id>`)
Connect instantly to a saved connection by ID:
```sh
# Get connection ID from error message or SSH config
sxt -c sky-central-1_1234567890

# Or use tab completion if your shell supports it
sxt -c <tab>
```

**Features** (v2.0+):
- ✅ **Pure Go SSH** - No external tools needed (passh, plink, ssh)
- ✅ **SSH Agent integration** - Automatically uses ssh-agent for encrypted keys
- ✅ **Password from keyring** - Retrieves passwords securely from system keyring
- ✅ **Auto-filter on keypress** (no need to press `/`)
- ✅ **Arrow keys exit filter and navigate**
- ✅ **Full terminal support** - xterm-256color built-in
- ✅ **Window resize** - Handles terminal resize properly
- ✅ **Works everywhere** - No tmux requirement, any terminal

**Using with encrypted SSH keys:**
```sh
# Add your keys to ssh-agent once
eval $(ssh-agent)
ssh-add ~/.ssh/id_rsa

# Now connections work without passphrase prompts!
sxt -l
# or
sxt -c <connection-id>
```

---

## ⚙️ Configuration

| Storage Mode | Details |
|--------------|---------|
| **Local** | • Config at `~/.ssh/config` (standard SSH config file)<br>• Passwords stored in **System Keyring**.<br>• Metadata stored as comments in SSH config.<br>• Fully compatible with standard SSH tools. |
| **Bitwarden** | • Secrets stored in your **Bitwarden Vault**.<br>• Requires `bw` CLI.<br>• Supports Organizations & Collections. |

**Note:** If you're upgrading from a previous version that used JSON config (`~/.config/ssh-x-term/ssh-x-term.json`), your connections will be automatically migrated to the SSH config format on first run.

---

## 🛡️ Security & Disclaimer

**SSH-X-Term** is an independent open-source project released under the [MIT License](LICENSE).

- **Credentials**: Your passwords/keys are handled securely via system APIs or Bitwarden. They are **never** logged or stored in plaintext files.
- **Responsibility**: The safe handling of your credentials is ultimately your responsibility. The authors bear no liability for data loss or compromise.
- **Affiliation**: Not affiliated with Bubble Tea, Bitwarden, or PuTTY.

---

## 👏 Credits

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — The TUI framework.
- [go-keyring](https://github.com/zalando/go-keyring) — Secure keyring integration.
- [Bitwarden CLI](https://bitwarden.com/help/cli/) — Vault management.
- [passh](https://github.com/clarkwang/passh) & [PuTTY](https://www.chiark.greenend.org.uk/~sgtatham/putty/latest.html) — Auth automation.
