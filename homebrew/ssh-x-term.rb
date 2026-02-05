class SshXTerm < Formula
  desc "TUI to handle multiple SSH connections simultaneously"
  homepage "https://github.com/eugeniofciuvasile/ssh-x-term"
  version "2.0.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.9/ssh-x-term-darwin-arm64"
      sha256 "8d14c9eb7f97ea41289222eb7ec5c478fa33691bcbe7514bf13d4ea173179d35"
    else
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.9/ssh-x-term-darwin-amd64"
      sha256 "e840b4aba0a47c371c31bdd5106db88a1debcae8aaae23350cdfe7a4580c00fd"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.9/ssh-x-term-linux-arm64"
      sha256 "bf3e7bc929fc938108595b8c078e036c1d9fe951c0da786cfce3d4bdb40f02b7"
    else
      url "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v2.0.9/ssh-x-term-linux-amd64"
      sha256 "69b0d92788a627658f6a458fb1e30d5feb6e903c996d69a0eaad288b8dfb5916"
    end
  end

  def install
    bin.install Dir["ssh-x-term-*"].first => "sxt"
  end

  test do
    system "#{bin}/sxt", "--version"
  end
end
