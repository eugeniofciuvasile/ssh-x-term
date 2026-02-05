# SSH-X-Term

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./logo.svg" width="240">
    <source media="(prefers-color-scheme: light)" srcset="./logo.svg" width="240">
    <img alt="SSH-X-Term Logo" src="./logo.svg" width="240">
  </picture>
  <br>
  <!-- Releases -->
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/releases"><img src="https://img.shields.io/github/v/release/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="Latest Release"></a>
  <!-- Homebrew -->
  <a href="https://github.com/eugeniofciuvasile/homebrew-tap"><img src="https://img.shields.io/badge/homebrew-available-brightgreen?style=flat-square&logo=homebrew" alt="Homebrew Tap"></a>
  <!-- Chocolatey -->
  <a href="https://community.chocolatey.org/packages/ssh-x-term"><img src="https://img.shields.io/chocolatey/v/ssh-x-term?style=flat-square&logo=chocolatey" alt="Chocolatey Version"></a>
  <!-- NPM -->
  <a href="https://www.npmjs.com/package/ssh-x-term"><img src="https://img.shields.io/npm/v/ssh-x-term?style=flat-square&logo=npm" alt="NPM Version"></a>
  <!-- Downloads -->
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/releases"><img src="https://img.shields.io/github/downloads/eugeniofciuvasile/ssh-x-term/total?style=flat-square&color=blue" alt="GitHub Downloads"></a>
  <a href="https://www.npmjs.com/package/ssh-x-term"><img src="https://img.shields.io/npm/dt/ssh-x-term?style=flat-square&logo=npm" alt="NPM Downloads"></a>
  <!-- CI -->
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/actions"><img src="https://github.com/eugeniofciuvasile/ssh-x-term/actions/workflows/go.yml/badge.svg" alt="Build Status"></a>
  <!-- Meta -->
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/stargazers"><img src="https://img.shields.io/github/stars/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="GitHub Stars"></a>
  <a href="https://github.com/eugeniofciuvasile/ssh-x-term/blob/main/LICENSE"><img src="https://img.shields.io/github/license/eugeniofciuvasile/ssh-x-term?style=flat-square" alt="License"></a>
</p>

---

**SSH-X-Term** is a modern, terminal-based SSH client with a rich TUI (Text User Interface) built on
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

As of **version 2.0**, SSH-X-Term is a **fully self-contained SSH client** implemented entirely in Go.
There are **no external SSH tools or wrappers** involved — all SSH, SCP, SFTP, and terminal handling is built in.

It combines **SSH connection management**, **interactive terminals**, **SCP/SFTP file transfers**, and
**secure credential storage** into a single, fast, cross-platform application.

Credentials can be stored securely using your **local system keyring** or directly in your
**Bitwarden vault**.

---

## ✨ What SSH-X-Term 2.0 Is

* ✅ **Pure Go SSH client** — no `ssh`, no `passh`, no `plink`
* ✅ **Cross-platform** — identical behavior on Linux, macOS, and Windows
* ✅ **Built-in terminal emulator** — full xterm-256color support
* ✅ **SSH Agent integration** — encrypted keys supported via `ssh-agent`
* ✅ **First-class TUI** — fast, keyboard-driven, and mouse-aware

---

## 📺 Demo & Walkthrough

<div align="center">

[![Watch on YouTube](https://img.shields.io/badge/Watch_on_YouTube-FF0000?style=for-the-badge\&logo=youtube\&logoColor=white)](https://www.youtube.com/watch?v=C-s-Lh_VdpQ)

![Demo](media/demo.gif)

</div>

---

## 🚀 Features

### ⚡ Quick Connect Mode

Fast SSH access without launching the full TUI.

* `sxt -l` — minimal interactive connection selector
* `sxt -c <connection-id>` — instant connection by ID
* Start typing immediately to filter connections
* Arrow keys exit filter and navigate
* 10 connections per page
* Fully interactive terminal with resize support

### 🖥️ Integrated SSH Terminal

* VT100 / ANSI escape sequence compliant
* Full **xterm-256color** support
* 10,000-line scrollback buffer
* Mouse and keyboard scrolling
* Text selection and clipboard copy
* Graceful window resize handling

### 📂 SCP / SFTP File Manager

* Dual-pane Local ↔ Remote interface
* Upload, download, rename, delete
* Create files and directories
* Recursive search (`/`)
* Uses the active authenticated SSH session

### 🔐 Secure Credential Management

* **Local storage** via system keyring

  * macOS Keychain
  * Linux Secret Service
  * Windows Credential Manager
* **Bitwarden integration** via Bitwarden CLI
* Passwords are never stored in plaintext

### ⚙️ SSH Authentication

* SSH Agent (recommended for encrypted keys)
* Encrypted private keys supported via `ssh-agent`
* Password authentication via system keyring
* Compatible with standard OpenSSH config

---

## 📦 Project Structure

(Structure unchanged — see repository tree for details)

---

## 🛠️ Prerequisites

### Required

* **Go 1.24+** (only if building from source)
* **System Keyring** (for local password storage)

### Optional

* **SSH Agent** (recommended for encrypted SSH keys)
* **Bitwarden CLI (`bw`)** — for Bitwarden vault support
* **tmux** — open SSH sessions in new tmux windows

> ⚠️ SSH-X-Term 2.0+ has **no external SSH dependencies**.
> You do not need `ssh`, `passh`, `plink`, or PuTTY.

---

## 📥 Installation

### Option 1: Install via npm (Recommended)

```sh
npm install -g ssh-x-term
sxt
```

> The npm installer downloads the correct prebuilt binary for your platform.
> Only optional tools (`bw`, `tmux`) may be suggested.

### Option 2: Build from source

```sh
git clone https://github.com/eugeniofciuvasile/ssh-x-term.git
cd ssh-x-term
go build -o sxt ./cmd/sxt
```

Or:

```sh
go install github.com/eugeniofciuvasile/ssh-x-term/cmd/sxt@latest
```

### Option 3: Prebuilt Binary

Download from the GitHub Releases page.

---

## 🎮 Usage

### First-Time Initialization

```sh
sxt -i
```

This will:

* Initialize configuration
* Migrate any existing SSH-X-Term data
* Prepare SSH config metadata

### Full TUI Mode

```sh
sxt
```

Key actions:

* `a` — Add connection
* `e` — Edit connection
* `d` — Delete connection
* `s` — Open SCP/SFTP manager
* `o` — Toggle tmux mode
* `Enter` — Connect

### Quick Connect (CLI)

```sh
sxt -l
sxt -c <connection-id>
```

---

## ⚙️ Configuration

| Storage   | Description                                                |
| --------- | ---------------------------------------------------------- |
| Local     | SSH config at `~/.ssh/config`, passwords in system keyring |
| Bitwarden | Secrets stored in Bitwarden vault via `bw` CLI             |

SSH-X-Term stores metadata as comments in your standard SSH config and remains fully compatible with OpenSSH tools.

---

## 🔑 SSH Agent Setup (Recommended)

```sh
eval $(ssh-agent)
ssh-add ~/.ssh/id_ed25519
```

Once added, SSH-X-Term can use encrypted keys without prompting for passphrases.

---

## 🛡️ Security & Disclaimer

SSH-X-Term is released under the MIT License.

* Credentials are never logged or written in plaintext
* All secrets are handled via OS APIs or Bitwarden
* Always ensure your system, SSH keys, and Bitwarden vault are properly secured

---

## 👏 Credits

* Bubble Tea — TUI framework
* go-keyring — Secure credential storage
* Bitwarden CLI — Vault integration
* OpenSSH — Protocol reference and compatibility
