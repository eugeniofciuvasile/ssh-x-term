# Chocolatey Package

This directory contains the Chocolatey package configuration for distributing SSH-X-Term on Windows.

## Package Structure

```
chocolatey/
├── ssh-x-term.nuspec              # Package metadata (ID, version, authors, etc.)
└── tools/
    ├── chocolateyinstall.ps1      # Installation script
    └── chocolateyuninstall.ps1    # Cleanup script
```

### Package Metadata (nuspec)

Defines package identity, dependencies, and metadata:
- Package ID: `ssh-x-term`
- Version: Synchronized with Git tags
- Authors, description, tags, project URLs
- File inclusions (only tools directory)

**Note**: The package uses remote download (not embedded binary) to keep package size small and ensure single source of truth from GitHub releases.

### Installation Script

The `chocolateyinstall.ps1` script:
1. Downloads Windows binary from GitHub release using `Get-ChocolateyWebFile`
2. Verifies SHA256 checksum
3. Saves directly as `sxt.exe` (no rename needed)
4. Chocolatey automatically creates shim for command-line access

**Key Change**: Uses `Get-ChocolateyWebFile` instead of `Install-ChocolateyZipPackage` since the download is an .exe file, not a .zip archive. This prevents extraction errors.

### Uninstallation Script

The `chocolateyuninstall.ps1` script:
1. Removes binary shim
2. Cleans up installation directory

## Automated Publishing

The `.github/workflows/chocolatey.yml` workflow publishes automatically on each release:

1. Downloads Windows binary from release
2. Computes SHA256 checksum
3. Updates nuspec and install script
4. Packages .nupkg file
5. Publishes to Chocolatey Community Repository

**Prerequisites**: `CHOCOLATEY_API_KEY` secret must be set in GitHub repository.

## Manual Publishing

If you need to publish manually:

```bash
# Update package files with version and checksum
./test-package.sh <version>

# On Windows machine
cd chocolatey
choco pack
choco push ssh-x-term.<version>.nupkg --source https://push.chocolatey.org/ --api-key $API_KEY
```

## Local Testing

Test the package before publishing:

```powershell
# Build package
cd chocolatey
choco pack

# Install locally
choco install ssh-x-term -s . -y

# Verify installation
sxt --version

# Check shim
where sxt

# Cleanup
choco uninstall ssh-x-term -y
```

## Moderation Process

### Initial Submission

First package submission requires moderation:
1. Automated virus scanning runs immediately
2. Automated package validation
3. Manual moderator review
4. Approval typically within 1-3 business days

Monitor status at: https://community.chocolatey.org/packages/ssh-x-term

### Package Updates

Subsequent updates:
- Trusted packages may auto-publish after validation
- Significant changes may trigger re-moderation
- Faster turnaround than initial submission

## Common Issues

### Antivirus False Positive (Trojan.Malware.300983.susgen)

**Cause**: Go binaries without code signing can trigger false positives in Windows Defender and other antivirus software.

**Solution**:
1. The binary is safe and built from open-source code via GitHub Actions
2. Add an exclusion for `sxt.exe` in your antivirus software temporarily
3. Submit the binary to your antivirus vendor as a false positive
4. The package includes VERIFICATION.txt and LICENSE.txt to help moderators verify authenticity
5. Long-term: Consider code signing the Windows binary (requires certificate)

**Windows Defender Exclusion** (run as Administrator):
```powershell
Add-MpPreference -ExclusionPath "C:\ProgramData\chocolatey\lib\ssh-x-term\tools\sxt.exe"
```

### Binary Not Found After Install ("Cannot find file at sxt.exe")

**Cause**: Antivirus quarantined the file after installation, or `Install-ChocolateyZipPackage` was used incorrectly for an .exe file.

**Solution** (Fixed in latest version):
- Now uses `Get-ChocolateyWebFile` which correctly downloads the .exe
- Downloads directly to `sxt.exe` (no rename needed)
- Reduces time window for antivirus to interfere
- Add antivirus exclusion before installing (see above)

**If already installed and broken**:
```powershell
# Uninstall broken package
choco uninstall ssh-x-term -y

# Add exclusion
Add-MpPreference -ExclusionPath "C:\ProgramData\chocolatey\lib\ssh-x-term\tools\sxt.exe"

# Reinstall
choco install ssh-x-term -y
```

### Checksum Mismatch

**Cause**: Incorrect or outdated checksum in install script

**Solution**:
```powershell
(Get-FileHash -Algorithm SHA256 -Path "ssh-x-term-windows-amd64.exe").Hash
```
Update checksum in `tools/chocolateyinstall.ps1`

### Package Validation Failed

**Cause**: Metadata errors or script issues

**Solution**:
1. Review validation errors at package moderation page
2. Test locally: `choco install ssh-x-term -s . -y`
3. Fix errors and resubmit

## Helper Scripts

### test-package.sh

Updates package metadata and checksums:

```bash
./test-package.sh <version>
```

Performs:
- Version update in nuspec
- Checksum computation for Windows binary
- Install script update

## Security Considerations

### Checksum Verification

Every installation verifies SHA256 checksum before executing the binary. This ensures:
- Binary integrity
- Protection against tampering
- Supply chain security

### Direct Binary Download

Package downloads binaries directly from GitHub releases (not bundled in .nupkg). This:
- Reduces package size
- Maintains single source of truth
- Simplifies updates

### No Elevated Permissions

Installation does not require administrator privileges for the binary itself. Chocolatey handles shim creation with appropriate permissions.

## Files Reference

| File | Purpose |
|------|---------|
| `ssh-x-term.nuspec` | Package metadata and configuration |
| `tools/chocolateyinstall.ps1` | Installation logic and binary setup |
| `tools/chocolateyuninstall.ps1` | Cleanup and shim removal |
| `tools/VERIFICATION.txt` | Verification instructions for moderators |
| `tools/LICENSE.txt` | Project license (helps with antivirus) |
| `test-package.sh` | Helper script for version/checksum updates |
| `README.md` | This file |

## References

- [Chocolatey Package Creation](https://docs.chocolatey.org/en-us/create/create-packages)
- [PowerShell Functions](https://docs.chocolatey.org/en-us/create/functions)
- [Moderation Process](https://docs.chocolatey.org/en-us/community-repository/moderation)
