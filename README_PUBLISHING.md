# Package Distribution Setup

This document describes the automated package distribution system for SSH-X-Term.

## Quick Start

### Prerequisites

1. **Chocolatey API Key** (required for Windows distribution):
   ```bash
   # 1. Create account at https://community.chocolatey.org/
   # 2. Get API key from https://community.chocolatey.org/account
   # 3. Add to GitHub: Settings → Secrets → Actions → New secret
   #    Name: CHOCOLATEY_API_KEY
   #    Value: <your-api-key>
   ```

2. **Homebrew Tap** (optional, for better user experience):
   ```bash
   # Create repository: eugeniofciuvasile/homebrew-tap
   # Copy formula: cp homebrew/ssh-x-term.rb <tap>/Formula/
   # Users can then: brew tap eugeniofciuvasile/tap && brew install ssh-x-term
   ```

### Creating a Release

The simplest workflow:

```bash
git add .
git commit -m "feat: add new feature"
git push origin main
```

This triggers the full pipeline automatically:
1. Build workflow creates GitHub release
2. Homebrew workflow updates formula
3. Chocolatey workflow publishes package
4. npm workflow publishes to npm registry

## Architecture

### Distribution Channels

```
┌─────────────────────────────────────────────────────────────┐
│                      GitHub Release                          │
│  (Source of truth - all binaries published here)            │
└──────────────┬──────────────────────────────────────────────┘
               │
               ├─────► Homebrew Formula Update (macOS/Linux)
               ├─────► Chocolatey Package (Windows)
               └─────► npm Package (Cross-platform wrapper)
```

### Workflow Dependencies

```
Push to main
     │
     ▼
Build Workflow (.github/workflows/go.yml)
     │
     ├─ Build binaries (all platforms)
     ├─ Generate SHA256SUMS
     ├─ Create GitHub Release
     └─ Publish to npm
     │
     ▼
Release Created Event
     │
     ├─────► Homebrew Workflow (.github/workflows/homebrew.yml)
     │       └─ Updates formula with new checksums
     │
     └─────► Chocolatey Workflow (.github/workflows/chocolatey.yml)
             └─ Publishes to Chocolatey repository
```

## Project Structure

```
.github/workflows/
├── go.yml              # Main build pipeline
├── homebrew.yml        # Homebrew formula updater
└── chocolatey.yml      # Chocolatey package publisher

homebrew/
├── ssh-x-term.rb       # Homebrew formula (multi-platform)
├── update-formula.sh   # Manual update helper
└── README.md           # Homebrew-specific docs

chocolatey/
├── ssh-x-term.nuspec              # Package metadata
├── test-package.sh                # Testing helper
├── README.md                      # Chocolatey-specific docs
└── tools/
    ├── chocolateyinstall.ps1      # Install logic
    └── chocolateyuninstall.ps1    # Cleanup logic
```

## Platform Support Matrix

| Platform | Architecture | Package Manager | Binary Name |
|----------|-------------|-----------------|-------------|
| macOS | x86_64 (Intel) | Homebrew | `sxt` |
| macOS | arm64 (Apple Silicon) | Homebrew | `sxt` |
| Linux | x86_64 | Homebrew | `sxt` |
| Linux | arm64 | Homebrew | `sxt` |
| Windows | x86_64 | Chocolatey | `sxt.exe` |

All platforms also supported via npm wrapper.

## Development Workflow

### Testing Changes Locally

Before pushing changes that affect package distribution:

1. **Test Homebrew Formula:**
   ```bash
   # After creating a release locally or on staging branch
   ./homebrew/update-formula.sh 2.0.3
   brew install --build-from-source homebrew/ssh-x-term.rb
   sxt --version
   brew uninstall ssh-x-term
   ```

2. **Test Chocolatey Package:**
   ```bash
   # Build Windows binary
   GOOS=windows GOARCH=amd64 go build -o dist/ssh-x-term-windows-amd64.exe ./cmd/sxt
   
   # Update package files
   ./chocolatey/test-package.sh 2.0.3
   
   # On Windows machine
   cd chocolatey
   choco pack
   choco install ssh-x-term -s . -y
   sxt --version
   choco uninstall ssh-x-term -y
   ```

3. **Verify Installation Methods:**
   ```bash
   ./verify-install.sh
   ```

### Manual Operations

If you need to bypass automation:

**Update Homebrew Formula:**
```bash
./homebrew/update-formula.sh <version>
git add homebrew/ssh-x-term.rb
git commit -m "chore: update homebrew formula to v<version>"
git push
```

**Publish Chocolatey Package:**
```bash
./chocolatey/test-package.sh <version>
cd chocolatey
choco pack
choco push ssh-x-term.<version>.nupkg --source https://push.chocolatey.org/ --api-key $API_KEY
```

## Installation Methods for Users

After publishing, users can install via:

### Homebrew (macOS/Linux)

```bash
# From personal tap (once created)
brew tap eugeniofciuvasile/tap
brew install ssh-x-term

# Or directly (testing)
brew install --build-from-source homebrew/ssh-x-term.rb
```

### Chocolatey (Windows)

```powershell
choco install ssh-x-term
```

### npm (All platforms)

```bash
npm install -g ssh-x-term
```

### Direct Binary Download

```bash
# Linux x86_64
curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-linux-amd64
chmod +x sxt
sudo mv sxt /usr/local/bin/

# macOS arm64
curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-darwin-arm64
chmod +x sxt
sudo mv sxt /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-windows-amd64.exe" -OutFile "sxt.exe"
```

## Monitoring and Maintenance

### Monitoring Releases

After pushing to main:
1. Check GitHub Actions tab for workflow status
2. Verify release was created with all binaries
3. Confirm Homebrew formula was updated (check commit)
4. Monitor Chocolatey package status at https://community.chocolatey.org/packages/ssh-x-term

### Handling Failures

**Build Workflow Fails:**
- Check Go version compatibility
- Verify all target platforms are supported
- Review build logs in GitHub Actions

**Homebrew Workflow Fails:**
- Ensure release assets were uploaded correctly
- Verify formula syntax: `brew audit homebrew/ssh-x-term.rb`
- Check that checksums can be downloaded from release

**Chocolatey Workflow Fails:**
- Verify `CHOCOLATEY_API_KEY` secret is set
- Check API key has not expired
- Review moderation feedback if package was rejected

## Security Considerations

### Checksum Verification

All binaries include SHA256 checksums:
- Generated during build: `dist/SHA256SUMS`
- Embedded in Homebrew formula
- Verified during Chocolatey installation
- Available for manual verification in GitHub releases

### API Key Security

- Chocolatey API key stored as GitHub secret (never in code)
- npm publishing uses OIDC (no long-lived tokens)
- GitHub Actions uses short-lived, scoped tokens

### Supply Chain Security

- All binaries built in GitHub Actions (reproducible)
- Workflows use pinned action versions
- Dependencies managed via go.mod (no lock file needed for Go)

## Future Enhancements

Potential additions to consider:

1. **Debian/Ubuntu Packages**: Use `fpm` to create .deb packages
2. **RPM Packages**: Create packages for Fedora/RHEL/CentOS
3. **AUR Package**: Submit to Arch User Repository
4. **Homebrew Core**: Submit to official Homebrew repository (requires 75+ stars)
5. **Snap/Flatpak**: Universal Linux packaging
6. **Windows Installer**: Create MSI installer for enterprise users

## Documentation

- `PUBLISHING.md` - Detailed guide on publishing process and package managers
- `homebrew/README.md` - Homebrew-specific documentation
- `chocolatey/README.md` - Chocolatey-specific documentation
- This file - Overall architecture and workflows

---

**Maintainer Notes**: Keep this documentation updated as the distribution strategy evolves. When adding new package managers, follow the same pattern: automated workflow + helper scripts + dedicated documentation.
