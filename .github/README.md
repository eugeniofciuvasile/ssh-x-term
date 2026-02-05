# GitHub Actions Workflows

This directory contains CI/CD automation for building and distributing SSH-X-Term.

## Workflows

### build.yml → go.yml (Main Build Pipeline)

**Trigger**: Push to main, manual dispatch

**Purpose**: Build binaries, create releases, publish to npm

**Process**:
1. Version management (auto-increment or manual)
2. Cross-compile for all platforms
3. Generate checksums
4. Create GitHub release
5. Publish to npm (OIDC)

**Outputs**: GitHub release with binaries and SHA256SUMS

### homebrew.yml (Homebrew Formula Updater)

**Trigger**: Release published

**Purpose**: Update Homebrew formula with new version and checksums

**Process**:
1. Download release binaries
2. Compute SHA256 checksums
3. Update formula file
4. Commit to main branch

**Dependencies**: Requires release assets from build workflow

### chocolatey.yml (Chocolatey Publisher)

**Trigger**: Release published, manual dispatch

**Purpose**: Package and publish to Chocolatey Community Repository

**Process**:
1. Download Windows binary
2. Compute SHA256 checksum
3. Update nuspec and install script
4. Package .nupkg
5. Publish (requires CHOCOLATEY_API_KEY secret)

**Dependencies**: Requires release assets and Chocolatey API key

### dependabot.yml (Dependency Updates)

**Trigger**: Scheduled

**Purpose**: Keep dependencies up to date

**Scope**:
- Go modules
- GitHub Actions versions

## Secrets Required

| Secret Name | Used By | Purpose |
|-------------|---------|---------|
| `GITHUB_TOKEN` | All workflows | Automatic, scoped to repository |
| `CHOCOLATEY_API_KEY` | chocolatey.yml | Publish to Chocolatey repository |
| npm token | go.yml | Not needed - uses OIDC |

## Workflow Dependencies

```
push to main
     │
     ▼
  go.yml (build)
     │
     ├─ Creates release
     │
     ▼
  Release published event
     │
     ├──► homebrew.yml
     │    └─ Updates formula
     │
     └──► chocolatey.yml
          └─ Publishes package
```

## Adding a New Workflow

When adding new distribution channels:

1. Create workflow file in this directory
2. Trigger on `release: types: [published]`
3. Download binaries from `${{ github.event.release.tag_name }}`
4. Add secrets if needed
5. Document in this README
6. Update DISTRIBUTION.md

## Monitoring

- View workflow runs: Repository → Actions tab
- Check specific workflow: Click workflow name in Actions
- View logs: Click on workflow run, then job name
- Failed workflows: Check email or Actions tab

## Testing Workflows

### Build Workflow

Test locally before pushing:
```bash
# Cross-compile locally
GOOS=linux GOARCH=amd64 go build -o dist/ssh-x-term-linux-amd64 ./cmd/sxt
GOOS=windows GOARCH=amd64 go build -o dist/ssh-x-term-windows-amd64.exe ./cmd/sxt
GOOS=darwin GOARCH=amd64 go build -o dist/ssh-x-term-darwin-amd64 ./cmd/sxt
GOOS=linux GOARCH=arm64 go build -o dist/ssh-x-term-linux-arm64 ./cmd/sxt
GOOS=darwin GOARCH=arm64 go build -o dist/ssh-x-term-darwin-arm64 ./cmd/sxt

# Generate checksums
cd dist && sha256sum ssh-x-term-* > SHA256SUMS
```

### Homebrew Workflow

Test formula update:
```bash
./homebrew/update-formula.sh <version>
brew install --build-from-source homebrew/ssh-x-term.rb
```

### Chocolatey Workflow

Test package creation:
```bash
./chocolatey/test-package.sh <version>
# On Windows:
cd chocolatey && choco pack
```

## Troubleshooting

### Workflow Not Triggering

- Verify trigger conditions match event
- Check workflow file syntax (YAML valid)
- Ensure workflow is on default branch

### Secrets Not Available

- Verify secret exists: Settings → Secrets → Actions
- Check secret name matches workflow reference
- Secrets are not available in forks (security)

### Permission Denied

- Check workflow permissions: Settings → Actions → General
- Verify GITHUB_TOKEN has required scopes
- Some operations require repository admin

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Workflow Syntax](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [OIDC with npm](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
