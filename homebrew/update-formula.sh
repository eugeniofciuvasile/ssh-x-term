#!/bin/bash
# Helper script to update Homebrew formula with checksums

set -e

VERSION=${1:-"2.0.3"}
TAG="v$VERSION"

echo "Updating Homebrew formula for version $VERSION"
echo "==============================================="

REPO_URL="https://github.com/eugeniofciuvasile/ssh-x-term"

echo "Downloading binaries..."
mkdir -p /tmp/brew-checksums
cd /tmp/brew-checksums

curl -sL -o darwin-amd64 "$REPO_URL/releases/download/$TAG/ssh-x-term-darwin-amd64"
curl -sL -o darwin-arm64 "$REPO_URL/releases/download/$TAG/ssh-x-term-darwin-arm64"
curl -sL -o linux-amd64 "$REPO_URL/releases/download/$TAG/ssh-x-term-linux-amd64"
curl -sL -o linux-arm64 "$REPO_URL/releases/download/$TAG/ssh-x-term-linux-arm64"

echo "Computing SHA256 checksums..."
SHA_DARWIN_AMD64=$(sha256sum darwin-amd64 | awk '{print $1}')
SHA_DARWIN_ARM64=$(sha256sum darwin-arm64 | awk '{print $1}')
SHA_LINUX_AMD64=$(sha256sum linux-amd64 | awk '{print $1}')
SHA_LINUX_ARM64=$(sha256sum linux-arm64 | awk '{print $1}')

echo ""
echo "Checksums:"
echo "  darwin-amd64: $SHA_DARWIN_AMD64"
echo "  darwin-arm64: $SHA_DARWIN_ARM64"
echo "  linux-amd64:  $SHA_LINUX_AMD64"
echo "  linux-arm64:  $SHA_LINUX_ARM64"
echo ""

cd - > /dev/null

# Update formula
FORMULA="homebrew/ssh-x-term.rb"

sed -i.bak "s/version \".*\"/version \"$VERSION\"/" "$FORMULA"
sed -i.bak "s|/v[0-9.]\+/|/$TAG/|g" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_AMD64_SHA256/$SHA_DARWIN_AMD64/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_ARM64_SHA256/$SHA_DARWIN_ARM64/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_LINUX_AMD64_SHA256/$SHA_LINUX_AMD64/" "$FORMULA"
sed -i.bak "s/PLACEHOLDER_LINUX_ARM64_SHA256/$SHA_LINUX_ARM64/" "$FORMULA"

rm -f "$FORMULA.bak"

echo "✅ Formula updated: $FORMULA"
echo ""
echo "To test locally:"
echo "  brew install --build-from-source $FORMULA"
echo "  sxt --version"
echo ""
echo "To publish to your tap:"
echo "  git add $FORMULA"
echo "  git commit -m 'chore: update formula to $TAG'"
echo "  git push"

# Cleanup
rm -rf /tmp/brew-checksums
