class SshXTerm < Formula
  desc "TUI to handle multiple SSH connections simultaneously"
  homepage "https://github.com/eugeniofciuvasile/ssh-x-term"
  version "2.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.3/ssh-x-term-darwin-arm64"
      sha256 "PLACEHOLDER_ARM64_SHA256"
    else
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.3/ssh-x-term-darwin-amd64"
      sha256 "PLACEHOLDER_AMD64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.3/ssh-x-term-linux-arm64"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    else
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.3/ssh-x-term-linux-amd64"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install Dir["ssh-x-term-*"].first => "sxt"
  end

  test do
    system "#{bin}/sxt", "--version"
  end
end
