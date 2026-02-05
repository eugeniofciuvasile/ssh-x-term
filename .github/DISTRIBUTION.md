# Distribution Architecture

This document describes the package distribution infrastructure for SSH-X-Term.

## Overview

SSH-X-Term uses a multi-channel distribution strategy to maximize platform coverage and user convenience:

1. **GitHub Releases** - Primary distribution channel (all platforms)
2. **npm** - Cross-platform via Node.js wrapper
3. **Homebrew** - macOS and Linux native package manager
4. **Chocolatey** - Windows package manager

All channels update automatically on release via GitHub Actions workflows.

## Build and Release Pipeline

```
Push to main
     │
     ▼
┌─────────────────────────────────────────────────────┐
│  Build Workflow (.github/workflows/go.yml)          │
│                                                      │
│  1. Version management (auto-increment or manual)   │
│  2. Cross-platform compilation:                     │
│     • Linux: amd64, arm64                           │
│     • macOS: amd64, arm64                           │
│     • Windows: amd64                                │
│  3. SHA256 checksum generation                      │
│  4. GitHub Release creation                         │
│  5. npm publish (via OIDC)                          │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
      Release "published" event
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
┌─────────────────┐   ┌─────────────────┐
│ Homebrew Update │   │ Chocolatey Pub  │
│                 │   │                 │
│ • Download bins │   │ • Download bin  │
│ • Compute SHA   │   │ • Compute SHA   │
│ • Update formula│   │ • Update nuspec │
│ • Commit        │   │ • Package       │
│                 │   │ • Publish       │
└─────────────────┘   └─────────────────┘
```

## Platform Support Matrix

| OS | Arch | Go Target | Binary Name | Package Managers |
|----|----|-----------|-------------|------------------|
| macOS | x86_64 | darwin/amd64 | sxt | Homebrew, npm |
| macOS | ARM64 | darwin/arm64 | sxt | Homebrew, npm |
| Linux | x86_64 | linux/amd64 | sxt | Homebrew, npm |
| Linux | ARM64 | linux/arm64 | sxt | Homebrew, npm |
| Windows | x86_64 | windows/amd64 | sxt.exe | Chocolatey, npm |

## Distribution Channels

### GitHub Releases

**Purpose**: Primary distribution, source of truth for all binaries

**Assets**:
- Platform-specific binaries: `ssh-x-term-{os}-{arch}[.exe]`
- Checksum file: `SHA256SUMS`
- Auto-generated release notes

**Usage**:
```bash
# Direct download and install
curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-linux-amd64
chmod +x sxt
sudo mv sxt /usr/local/bin/
```

### npm Registry

**Purpose**: Cross-platform distribution via Node.js ecosystem

**Package**: `ssh-x-term`

**Mechanism**: Post-install script downloads platform-appropriate binary from GitHub release

**Publishing**: Automated via OIDC (no API key needed)

**Usage**:
```bash
npm install -g ssh-x-term
```

### Homebrew

**Purpose**: Native package management for macOS and Linux

**Formula**: `homebrew/ssh-x-term.rb` (in-tree) or personal tap

**Distribution Options**:
1. In-tree formula (current)
2. Personal tap: `eugeniofciuvasile/homebrew-tap` (recommended)
3. Homebrew core (future, requires 75+ stars)

**Update Mechanism**: Automated workflow downloads binaries, computes checksums, updates formula

**Usage**:
```bash
# Personal tap (once created)
brew tap eugeniofciuvasile/tap
brew install ssh-x-term

# Direct (testing)
brew install --build-from-source homebrew/ssh-x-term.rb
```

### Chocolatey

**Purpose**: Windows package management

**Package**: `ssh-x-term`

**Repository**: Chocolatey Community Repository

**Moderation**: Initial submission requires manual approval (1-3 days)

**Update Mechanism**: Automated workflow packages and publishes on release

**Usage**:
```powershell
choco install ssh-x-term
```

## Security Model

### Binary Integrity

All binaries include SHA256 checksums verified at installation:

1. **GitHub Releases**: `SHA256SUMS` file in release assets
2. **Homebrew**: Checksums embedded in formula, verified by Homebrew
3. **Chocolatey**: Checksum verified by install script
4. **npm**: Checksum verified by install.js script

### Supply Chain Security

- Builds run in GitHub Actions (auditable, reproducible)
- No long-lived secrets (OIDC for npm, short-lived tokens for GitHub)
- Chocolatey API key stored as GitHub secret (scoped access)
- All workflows use pinned action versions

### Checksum Generation

```bash
# During build
cd dist
sha256sum ssh-x-term-* > SHA256SUMS

# Verification
sha256sum -c SHA256SUMS
```

## Workflow Configuration

### Build Workflow

**File**: `.github/workflows/go.yml`

**Triggers**:
- Push to main branch
- Manual dispatch (with version override)

**Key Steps**:
1. Version determination (auto-increment or manual)
2. npm version bump and commit
3. Tag creation and push
4. Cross-platform Go builds
5. Checksum generation
6. GitHub release creation
7. npm publish (OIDC)

**Outputs**:
- `new_tag`: Created version tag (consumed by dependent workflows)

### Homebrew Workflow

**File**: `.github/workflows/homebrew.yml`

**Trigger**: Release published

**Key Steps**:
1. Download release binaries (all platforms supported by formula)
2. Compute SHA256 checksums
3. Update formula with new version, URLs, checksums
4. Commit and push to main branch

**Dependencies**: Requires release assets to exist

### Chocolatey Workflow

**File**: `.github/workflows/chocolatey.yml`

**Trigger**: Release published

**Key Steps**:
1. Download Windows binary
2. Compute SHA256 checksum
3. Update nuspec and install script
4. Package .nupkg
5. Publish to Chocolatey (if release event, not workflow_dispatch)

**Dependencies**:
- Requires `CHOCOLATEY_API_KEY` secret
- Requires Windows runner

**Optional**: Manual workflow dispatch for testing

## Maintenance Procedures

### Adding a New Platform

1. Add target to build workflow:
   ```yaml
   GOOS=<os> GOARCH=<arch> go build -o dist/ssh-x-term-<os>-<arch> ./cmd/sxt
   ```

2. Update Homebrew formula if applicable:
   ```ruby
   on_<os> do
     if Hardware::CPU.<arch>?
       url "..."
       sha256 "..."
     end
   end
   ```

3. Update Chocolatey if Windows variant

4. Update npm install.js to recognize new platform

### Updating Go Version

1. Update build workflow:
   ```yaml
   - uses: actions/setup-go@v4
     with:
       go-version: '1.26'  # New version
   ```

2. Test locally:
   ```bash
   go version
   go build ./cmd/sxt
   ```

3. Update documentation if breaking changes

### Rotating Chocolatey API Key

1. Generate new key at https://community.chocolatey.org/account
2. Update GitHub secret: Settings → Secrets → Actions → `CHOCOLATEY_API_KEY`
3. Old key can be revoked immediately (workflows use secret value)

## Monitoring

### Release Status

After push to main:
1. Check GitHub Actions for workflow status
2. Verify release creation with all assets
3. Confirm formula update (new commit in main)
4. Monitor Chocolatey moderation status

### Package Status

| Channel | Status Check |
|---------|-------------|
| GitHub | https://github.com/eugeniofciuvasile/ssh-x-term/releases |
| npm | https://www.npmjs.com/package/ssh-x-term |
| Homebrew | `brew info ssh-x-term` (if tapped) |
| Chocolatey | https://community.chocolatey.org/packages/ssh-x-term |

### Failure Alerts

GitHub Actions sends notifications on workflow failures:
- Check email associated with GitHub account
- Or monitor Actions tab directly

## Future Enhancements

Potential distribution channels to consider:

1. **Snap/Flatpak**: Universal Linux packaging
2. **Debian/Ubuntu Packages**: Native .deb packages
3. **RPM Packages**: Fedora/RHEL/CentOS support
4. **AUR**: Arch Linux User Repository
5. **Docker Hub**: Containerized distribution
6. **Homebrew Core**: Official Homebrew repository (requires maturity)

Each addition should follow the same pattern:
- Automated workflow
- Helper scripts for manual operations
- Dedicated documentation
- Integration testing

---

**Last Updated**: 2026-02-05
**Maintainer**: Eugen Iofciu Vasile
