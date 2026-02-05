#!/bin/bash
# Installation verification script for SSH-X-Term
# Tests all supported installation methods and reports status

set -e

readonly RESET='\033[0m'
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'

print_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BLUE}$1${RESET}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo ""
}

check_package_manager() {
    local name=$1
    local command=$2
    local check_cmd=$3
    
    if ! command -v "$command" &> /dev/null; then
        echo -e "${YELLOW}⊘${RESET} $name not available on this system"
        return 1
    fi
    
    echo -n "Checking $name... "
    if eval "$check_cmd" &> /dev/null; then
        echo -e "${GREEN}✓ installed${RESET}"
        return 0
    else
        echo -e "${RED}✗ not installed${RESET}"
        return 1
    fi
}

check_executable() {
    echo -n "Checking sxt executable... "
    if command -v sxt &> /dev/null; then
        echo -e "${GREEN}✓ found${RESET}"
        local version
        version=$(sxt --version 2>&1 || echo "version unavailable")
        echo "  Location: $(which sxt)"
        echo "  Version: $version"
        return 0
    else
        echo -e "${RED}✗ not found${RESET}"
        return 1
    fi
}

print_installation_guide() {
    print_header "Installation Options"
    
    case "$OSTYPE" in
        darwin*)
            echo "macOS detected. Install via:"
            echo ""
            echo "  # Homebrew (recommended)"
            echo "  brew tap eugeniofciuvasile/tap"
            echo "  brew install ssh-x-term"
            echo ""
            echo "  # npm"
            echo "  npm install -g ssh-x-term"
            echo ""
            echo "  # Direct download"
            local arch
            arch=$(uname -m)
            if [[ "$arch" == "arm64" ]]; then
                echo "  curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-darwin-arm64"
            else
                echo "  curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-darwin-amd64"
            fi
            echo "  chmod +x sxt"
            echo "  sudo mv sxt /usr/local/bin/"
            ;;
        linux*)
            echo "Linux detected. Install via:"
            echo ""
            echo "  # Homebrew"
            echo "  brew tap eugeniofciuvasile/tap"
            echo "  brew install ssh-x-term"
            echo ""
            echo "  # npm"
            echo "  npm install -g ssh-x-term"
            echo ""
            echo "  # Direct download"
            local arch
            arch=$(uname -m)
            if [[ "$arch" == "aarch64" ]] || [[ "$arch" == "arm64" ]]; then
                echo "  curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-linux-arm64"
            else
                echo "  curl -L -o sxt https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-linux-amd64"
            fi
            echo "  chmod +x sxt"
            echo "  sudo mv sxt /usr/local/bin/"
            ;;
        msys*|win32*|cygwin*)
            echo "Windows detected. Install via:"
            echo ""
            echo "  # Chocolatey (recommended)"
            echo "  choco install ssh-x-term"
            echo ""
            echo "  # npm"
            echo "  npm install -g ssh-x-term"
            echo ""
            echo "  # Direct download (PowerShell)"
            echo "  Invoke-WebRequest -Uri 'https://github.com/eugeniofciuvasile/ssh-x-term/releases/latest/download/ssh-x-term-windows-amd64.exe' -OutFile 'sxt.exe'"
            ;;
        *)
            echo "Unknown OS. Install via:"
            echo ""
            echo "  # npm (cross-platform)"
            echo "  npm install -g ssh-x-term"
            ;;
    esac
}

main() {
    print_header "SSH-X-Term Installation Verification"
    
    local status=0
    
    # Check package managers
    case "$OSTYPE" in
        darwin*|linux*)
            check_package_manager "Homebrew" "brew" "brew info ssh-x-term" || true
            ;;
        msys*|win32*|cygwin*)
            check_package_manager "Chocolatey" "choco" "choco list --local-only ssh-x-term | grep -q ssh-x-term" || true
            ;;
    esac
    
    check_package_manager "npm" "npm" "npm list -g ssh-x-term" || true
    
    echo ""
    
    # Check executable
    if ! check_executable; then
        status=1
        echo ""
        print_installation_guide
    fi
    
    echo ""
    print_header "Verification Complete"
    
    if [[ $status -eq 0 ]]; then
        echo -e "${GREEN}✓ SSH-X-Term is installed and ready to use${RESET}"
        echo ""
        echo "Run 'sxt' to start the application"
    else
        echo -e "${YELLOW}⚠ SSH-X-Term is not installed${RESET}"
        echo ""
        echo "Follow the installation guide above"
    fi
    
    echo ""
    
    return $status
}

main "$@"
