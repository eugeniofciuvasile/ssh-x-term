# Distribution System

SSH-X-Term uses an automated multi-channel distribution pipeline to publish releases across GitHub, npm, Homebrew, and Chocolatey.

## Documentation Index

### Quick Reference

- **[DISTRIBUTION_QUICKSTART.md](DISTRIBUTION_QUICKSTART.md)** - Day-to-day operations guide
- **[README_PUBLISHING.md](README_PUBLISHING.md)** - Architecture overview and workflows

### Detailed Guides

- **[PUBLISHING.md](PUBLISHING.md)** - Complete publishing process documentation
- **[.github/DISTRIBUTION.md](.github/DISTRIBUTION.md)** - Technical architecture details
- **[.github/README.md](.github/README.md)** - GitHub Actions workflows reference

### Package Manager Specifics

- **[homebrew/README.md](homebrew/README.md)** - Homebrew formula management
- **[chocolatey/README.md](chocolatey/README.md)** - Chocolatey package management

## Quick Start

### Setup (Once)

1. Add Chocolatey API key to GitHub Secrets:
   ```
   Repository → Settings → Secrets → Actions → CHOCOLATEY_API_KEY
   ```

2. (Optional) Create Homebrew tap for easier installation

### Daily Usage

```bash
# Make changes
git commit -m "feat: new feature"
git push

# Automation handles:
# - Version bump
# - Build all platforms
# - Create GitHub release
# - Update package managers
```

## Architecture

```
GitHub Release (source of truth)
        │
        ├──► Homebrew (macOS/Linux)
        ├──► Chocolatey (Windows)
        └──► npm (cross-platform)
```

## Platform Support

| OS | Architectures | Package Managers |
|----|---------------|------------------|
| macOS | x86_64, ARM64 | Homebrew, npm |
| Linux | x86_64, ARM64 | Homebrew, npm |
| Windows | x86_64 | Chocolatey, npm |

## Installation (Users)

```bash
# Homebrew (macOS/Linux)
brew tap eugeniofciuvasile/tap
brew install ssh-x-term

# Chocolatey (Windows)
choco install ssh-x-term

# npm (all platforms)
npm install -g ssh-x-term
```

## Troubleshooting

| Issue | Reference |
|-------|-----------|
| Workflow failures | [.github/README.md](.github/README.md) |
| Homebrew problems | [homebrew/README.md](homebrew/README.md) |
| Chocolatey issues | [chocolatey/README.md](chocolatey/README.md) |
| General questions | [PUBLISHING.md](PUBLISHING.md) |

## Maintenance

### Regular Tasks

- Monitor GitHub Actions for failures
- Review Chocolatey moderation feedback (first submission)
- Update documentation as distribution strategy evolves

### Adding New Channels

1. Create workflow in `.github/workflows/`
2. Trigger on `release: types: [published]`
3. Download binaries from release
4. Package and publish
5. Document in relevant README

## References

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Chocolatey Package Guidelines](https://docs.chocolatey.org/en-us/create/create-packages)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

---

**Start with DISTRIBUTION_QUICKSTART.md for practical day-to-day guidance.**
