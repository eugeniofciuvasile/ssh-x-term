# Homebrew Formula

This directory contains the Homebrew formula for distributing SSH-X-Term on macOS and Linux.

## Formula Structure

```ruby
class SshXTerm < Formula
  desc "TUI to handle multiple SSH connections simultaneously"
  homepage "https://github.com/eugeniofciuvasile/ssh-x-term"
  version "2.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "..." # Apple Silicon binary
      sha256 "..."
    else
      url "..." # Intel binary
      sha256 "..."
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "..." # ARM64 binary
      sha256 "..."
    else
      url "..." # x86_64 binary
      sha256 "..."
    end
  end

  def install
    bin.install Dir["ssh-x-term-*"].first => "sxt"
  end

  test do
    system "#{bin}/sxt", "--version"
  end
end
```

## Automated Updates

The `.github/workflows/homebrew.yml` workflow updates this formula automatically on each release:

1. Downloads release binaries for all platforms
2. Computes SHA256 checksums
3. Updates formula with new version and checksums
4. Commits changes to main branch

No manual intervention required.

## Manual Updates

If automation fails or you need to update manually:

```bash
./update-formula.sh <version>
```

This script:
- Downloads binaries from the specified release
- Computes SHA256 checksums
- Updates `ssh-x-term.rb` with new values

## Testing

Test the formula locally before pushing:

```bash
# Install from local formula
brew install --build-from-source ssh-x-term.rb

# Verify installation
sxt --version

# Cleanup
brew uninstall ssh-x-term
```

## Distribution Strategies

### Current: In-Tree Formula

Formula lives in this repository. Users install via:

```bash
brew install --build-from-source /path/to/ssh-x-term/homebrew/ssh-x-term.rb
```

**Pros**: Low maintenance, easy testing
**Cons**: Less discoverable, manual path required

### Recommended: Personal Tap

Create a separate tap repository:

```bash
# 1. Create repository: eugeniofciuvasile/homebrew-tap
# 2. Add formula:
mkdir -p Formula
cp ssh-x-term.rb Formula/

# 3. Users install:
brew tap eugeniofciuvasile/tap
brew install ssh-x-term
```

**Pros**: Professional distribution, easier installation, better discoverability
**Cons**: Extra repository to maintain

### Future: Homebrew Core

Once the project meets criteria:
- 75+ GitHub stars
- 30+ days old
- Active maintenance
- Notable userbase

Submit a PR to `Homebrew/homebrew-core` for widest distribution.

## Files

- `ssh-x-term.rb` - Homebrew formula (multi-platform support)
- `update-formula.sh` - Manual update helper script
- `README.md` - This file

## References

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Ruby API](https://rubydoc.brew.sh/Formula)
- [Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
