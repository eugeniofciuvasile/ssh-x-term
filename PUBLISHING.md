# Package Distribution Guide

This document outlines the automated publishing pipeline for distributing SSH-X-Term to multiple package managers.

## 📦 Homebrew (macOS & Linux)

### Option 1: Personal Tap (Recommended for Start)

1. **Create a Homebrew tap repository:**
   ```bash
   # Create a new GitHub repository named: homebrew-tap
   # Repository should be: eugeniofciuvasile/homebrew-tap
   ```

2. **Copy the formula:**
   ```bash
   cp homebrew/ssh-x-term.rb <tap-repo>/Formula/ssh-x-term.rb
   ```

3. **Users install via:**
   ```bash
   brew tap eugeniofciuvasile/tap
   brew install ssh-x-term
   ```

4. **Automate updates:**
   - The `.github/workflows/homebrew.yml` workflow automatically updates the formula on each release

### Option 2: Homebrew Core (For Popular Packages)

1. **Requirements:**
   - 75+ GitHub stars
   - 30+ days old
   - Active maintenance
   - Notable userbase

2. **Submit PR to homebrew-core:**
   ```bash
   # Fork homebrew-core
   # Add your formula to Formula/
   # Submit PR
   ```

### Automated Updates

On each release, `.github/workflows/homebrew.yml` automatically:
1. Downloads release binaries
2. Computes SHA256 checksums
3. Updates formula with new version and checksums
4. Commits changes to main branch

### Manual Updates (If Needed)

```bash
./homebrew/update-formula.sh <version>
```

This script handles checksum computation and formula updates.

## Chocolatey Distribution

### Prerequisites

1. Create account at https://community.chocolatey.org/
2. Generate API key from account settings
3. Add GitHub secret:
   - Repository Settings → Secrets and variables → Actions
   - Name: `CHOCOLATEY_API_KEY`
   - Value: `<your-api-key>`

### Package Structure

```
chocolatey/
├── ssh-x-term.nuspec              # Package metadata
└── tools/
    ├── chocolateyinstall.ps1      # Installation logic
    └── chocolateyuninstall.ps1    # Cleanup logic
```

### Automated Publishing

The `.github/workflows/chocolatey.yml` workflow handles:
1. Binary download from release
2. SHA256 checksum computation
3. Metadata and script updates
4. Package creation (.nupkg)
5. Publishing to Chocolatey Community Repository

### Manual Publishing (If Needed)

```bash
# Update package files
./chocolatey/test-package.sh <version>

# On Windows: package and publish
cd chocolatey
choco pack
choco push ssh-x-term.<version>.nupkg --source https://push.chocolatey.org/ --api-key $API_KEY
```

### First Submission Notes

Initial package submission requires moderator approval:
- Automated security scanning runs immediately
- Manual review typically completes within 1-3 business days
- Subsequent updates publish faster (often automated)
- Monitor status at: https://community.chocolatey.org/packages/ssh-x-term

## CI/CD Pipeline

### Build Workflow (`.github/workflows/go.yml`)

Triggers on push to main:
1. Auto-increments version (or uses manual input)
2. Builds binaries for all platforms:
   - Linux: amd64, arm64
   - macOS: amd64 (Intel), arm64 (Apple Silicon)
   - Windows: amd64
3. Generates SHA256SUMS file
4. Creates GitHub release with all assets
5. Publishes to npm (via OIDC)

### Publishing Workflows

**Homebrew** (`.github/workflows/homebrew.yml`)
- **Trigger**: Release published
- **Actions**: Download binaries → Compute checksums → Update formula → Commit
- **Output**: Updated `homebrew/ssh-x-term.rb` in main branch

**Chocolatey** (`.github/workflows/chocolatey.yml`)
- **Trigger**: Release published
- **Actions**: Download binary → Compute checksum → Update metadata → Package → Publish
- **Output**: Published package to Chocolatey Community Repository

## Release Process

### Automated Release (Recommended)

```bash
git add .
git commit -m "feat: your feature description"
git push origin main
```

The build workflow automatically:
1. Increments patch version
2. Updates package.json
3. Creates git tag
4. Builds all platform binaries
5. Creates GitHub release
6. Triggers publishing workflows

### Manual Release (Advanced)

```bash
# Trigger workflow manually with specific version
# GitHub Actions → Build → Run workflow → Input version
```

### Post-Release Verification

After release creation:
- ✅ Homebrew formula updated in ~2 minutes
- ✅ Chocolatey package published (pending moderation for first submission)
- ✅ npm package published via OIDC
- ✅ GitHub release created with binaries and checksums

## Testing

### Local Testing

**Homebrew Formula:**
```bash
brew install --build-from-source homebrew/ssh-x-term.rb
sxt --version
brew uninstall ssh-x-term
```

**Chocolatey Package:**
```powershell
cd chocolatey
choco pack
choco install ssh-x-term -s . -y
sxt --version
choco uninstall ssh-x-term -y
```

### Verification Script

```bash
./verify-install.sh
```

Tests all installation methods and reports status.

## Troubleshooting

### Homebrew

| Issue | Solution |
|-------|----------|
| SHA256 mismatch | Run `./homebrew/update-formula.sh <version>` |
| Formula not found | Verify tap is added: `brew tap eugeniofciuvasile/tap` |
| Install fails | Check syntax: `brew audit homebrew/ssh-x-term.rb` |

### Chocolatey

| Issue | Solution |
|-------|----------|
| Checksum mismatch | Recompute: `Get-FileHash -Algorithm SHA256 <file>` |
| Package rejected | Review feedback at chocolatey.org moderation page |
| Install fails | Check logs: `C:\ProgramData\chocolatey\logs\chocolatey.log` |

### Workflows

| Issue | Solution |
|-------|----------|
| Homebrew workflow fails | Verify release assets exist, check GitHub Actions logs |
| Chocolatey workflow fails | Verify `CHOCOLATEY_API_KEY` secret is set correctly |
| Build workflow fails | Check Go version compatibility, review build logs |

## References

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Chocolatey Package Creation](https://docs.chocolatey.org/en-us/create/create-packages)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
