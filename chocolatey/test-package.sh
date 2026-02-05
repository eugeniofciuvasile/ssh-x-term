#!/bin/bash
# Helper script to test Chocolatey package locally on Windows (via WSL or Git Bash)

set -e

VERSION=${1:-"2.0.3"}

echo "Testing Chocolatey package for version $VERSION"
echo "================================================"

cd chocolatey

# Ensure we have the binary
if [ ! -f "../dist/ssh-x-term-windows-amd64.exe" ]; then
    echo "Error: Binary not found. Build it first with:"
    echo "  GOOS=windows GOARCH=amd64 go build -o dist/ssh-x-term-windows-amd64.exe ./cmd/sxt"
    exit 1
fi

# Compute checksum
if command -v sha256sum &> /dev/null; then
    CHECKSUM=$(sha256sum ../dist/ssh-x-term-windows-amd64.exe | awk '{print $1}')
elif command -v shasum &> /dev/null; then
    CHECKSUM=$(shasum -a 256 ../dist/ssh-x-term-windows-amd64.exe | awk '{print $1}')
else
    echo "Warning: Could not compute checksum. Please do it manually."
    CHECKSUM="PLACEHOLDER_CHECKSUM"
fi

echo "Checksum: $CHECKSUM"

# Update files
sed -i.bak "s/<version>.*<\/version>/<version>$VERSION<\/version>/" ssh-x-term.nuspec
sed -i.bak "s/\$version = '.*'/\$version = '$VERSION'/" tools/chocolateyinstall.ps1
sed -i.bak "s/checksum64     = '.*'/checksum64     = '$CHECKSUM'/" tools/chocolateyinstall.ps1

echo ""
echo "Files updated! Now run on Windows:"
echo ""
echo "  cd chocolatey"
echo "  choco pack"
echo "  choco install ssh-x-term -s . -y"
echo "  sxt --version"
echo ""
echo "To publish:"
echo "  choco push ssh-x-term.$VERSION.nupkg --source https://push.chocolatey.org/ --api-key YOUR_API_KEY"
